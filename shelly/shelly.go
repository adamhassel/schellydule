// Package shelly implements calls to a shelly unit using HTTP RPC
package shelly

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	url2 "net/url"
	"time"

	"github.com/adamhassel/errors"
	"github.com/robfig/cron/v3"
)

// Schedule is a top-level shelly schedule collection
type Schedule struct {
	Jobs Schedules `json:"jobs"`
}

// JobSpec is a Shelly schedule trigger
type JobSpec struct {
	Id       int    `json:"id"`
	Enable   bool   `json:"enable"`
	Timespec string `json:"timespec"`
	Calls    []Call `json:"calls"`
}

// Call is what happens when the schedule time rolls over
type Call struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

type Schedules []JobSpec

type State bool

const (
	StateOn  State = true
	StateOff State = false
)

func GetSchedules(dest fmt.Stringer) (Schedules, error) {
	url := fmt.Sprintf("http://%s/rpc/Schedule.List", dest.String())
	r, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	var schedules Schedule
	err = json.Unmarshal(body, &schedules)

	return schedules.Jobs, err
}

// Time returns the timestamp for the job (with today's date)
func (j JobSpec) Time() (time.Time, error) {
	today := time.Now().Truncate(24 * time.Hour) // truncate to midnight today
	t, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Second).Parse(j.Timespec)
	if err != nil {
		return time.Time{}, err
	}
	return t.Next(today), nil
}

func enableDisableSchedules(dest fmt.Stringer, enable bool, ids ...int) error {
	for _, id := range ids {
		if err := DoRPCCall(dest, "Schedule.Update", map[string]string{"id": fmt.Sprintf("%d", id), "enable": fmt.Sprintf("%t", enable)}); err != nil {
			return err
		}
	}
	return nil
}

func EnableSchedules(dest fmt.Stringer, ids ...int) error {
	return enableDisableSchedules(dest, true, ids...)
}

func DisableSchedules(dest fmt.Stringer, ids ...int) error {
	return enableDisableSchedules(dest, false, ids...)
}

// TurnOn sets the Shelly's switch to the "On" state
func TurnOn(dest fmt.Stringer) error {
	return SetSwitch(dest, StateOn)
}

// TurnOff sets the Shelly's switch to the "Off" state
func TurnOff(dest fmt.Stringer) error {
	return SetSwitch(dest, StateOff)
}

// SetSwitch sets the Shelly's switch to the given state
func SetSwitch(dest fmt.Stringer, state State) error {
	return DoRPCCall(dest, "Switch.Set", map[string]string{"id": "0", "on": fmt.Sprintf("%t", state)})
}

func DeleteAllSchedules(dest fmt.Stringer) error {
	return DoRPCCall(dest, "Schedule.DeleteAll", nil)
}

func CreateSchedule(dest fmt.Stringer, schedule Schedule) error {
	return nil
}

// Do an RPC call towards the shelly
func DoRPCCall(dest fmt.Stringer, method string, options map[string]string) error {
	u := url2.URL{
		Scheme: "http",
		Host:   dest.String(),
		Path:   "rpc/" + method,
	}
	if options != nil {
		for k, v := range options {
			u.Query().Add(k, v)
		}
	}
	r, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(r.Body)
		add := errors.New(string(body))
		if err != nil {
			add = errors.Wrap(add, fmt.Errorf("\nAdditionally, an error occurred while reading return body: %w", err))
		}
		return fmt.Errorf("RPC call returned %s (%d) %w", r.Status, r.StatusCode, add)
	}
	return nil
}

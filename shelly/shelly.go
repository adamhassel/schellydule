// Package shelly implements calls to a shelly unit using HTTP RPC
package shelly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/adamhassel/errors"
	"github.com/adamhassel/schedule"
	"github.com/robfig/cron/v3"
	"github.com/tidwall/gjson"
)

const cronFormat = "05 04 15 * * MON,TUE,WED,THU,FRI,SAT,SUN"

const refresherID = 42

// Schedule is a top-level shelly schedule collection
type Schedule struct {
	Jobs Schedules `json:"jobs"`
}

type Schedules []JobSpec

// JobSpec is a Shelly schedule trigger
type JobSpec struct {
	Id       int    `json:"id,omitempty"`
	Enable   bool   `json:"enable"`
	Timespec string `json:"timespec"`
	Calls    []Call `json:"calls"`
}

// Call is what the job should do
type Call struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

type State bool

const (
	StateOn  State = true
	StateOff State = false
)

func GetSchedules(ip fmt.Stringer) (Schedules, error) {
	body, _, err := DoRPCCall(ip, "GET", "Schedule.List", nil, nil)
	if err != nil {
		return Schedules{}, err
	}
	var schedules Schedule
	fmt.Println(string(body))
	err = json.Unmarshal(body, &schedules)
	return schedules.Jobs, err
}

// ShellySchedule converts a schedule.Schedule to something a Shelly can understand.
func ShellySchedule(in schedule.Schedule, enable bool) Schedule {
	var out Schedule
	out.Jobs = make(Schedules, 0, len(in)*2)
	for _, se := range in {
		on := JobSpec{
			Enable:   enable,
			Timespec: se.Start.Format(cronFormat),
			Calls: []Call{{
				Method: "Switch.Set",
				Params: map[string]interface{}{
					"id":   0,
					"on":   true,
					"cost": se.Cost,
				}},
			},
		}
		off := JobSpec{
			Enable:   enable,
			Timespec: se.Stop.Format(cronFormat),
			Calls: []Call{{
				Method: "Switch.Set",
				Params: map[string]interface{}{
					"id": 0,
					"on": false,
				}},
			},
		}
		out.Jobs = append(out.Jobs, on, off)
	}
	return out
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

// Methods returns the Methods the job calls
func (j JobSpec) Methods() []string {
	if len(j.Calls) == 0 {
		return nil
	}
	rv := make([]string, 0, len(j.Calls))
	for _, c := range j.Calls {
		rv = append(rv, c.Method)
	}
	return rv
}

// HasMethod returns true if m is a method in j
func (j JobSpec) HasMethod(m string) bool {
	return stringInSlice(m, j.Methods())
}

func enableDisableSchedules(dest fmt.Stringer, enable bool, ids ...int) error {
	for _, id := range ids {
		if _, _, err := DoGet(dest, "Schedule.Update", map[string]string{"id": fmt.Sprintf("%d", id), "enable": fmt.Sprintf("%t", enable)}); err != nil {
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
	_, _, err := DoGet(dest, "Switch.Set", map[string]string{"id": "0", "on": fmt.Sprintf("%t", state)})
	return err
}

func DeleteAllSchedules(dest fmt.Stringer) error {
	_, _, err := DoGet(dest, "Schedule.DeleteAll", nil)
	return err
}

func CreateSchedule(dest fmt.Stringer, s Schedule) error {
	for _, j := range s.Jobs {
		reqBody, err := json.Marshal(j)
		if err != nil {
			return err
		}
		fmt.Println(string(reqBody))
		if _, _, err := DoRPCCall(dest, "POST", "Schedule.Create", nil, reqBody); err != nil {
			return err
		}
	}
	return nil
}

func getOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}

// CreateScheduleRefresherSchedule will make sure that the schedules are refreshed every day @ 23.55
func CreateScheduleRefresherSchedule(dest fmt.Stringer, myPort int) error {
	t := schedule.Hour(time.Now(), 0).Add(1 * time.Minute)
	ip, err := getOutboundIP()
	if err != nil {
		return err
	}
	refresh := JobSpec{
		Id:     refresherID,
		Enable: true,
		// Set the timespec to be tomorrow at 23:30: First add 24 hours, to be sure it's tomorrow. Then truncate to midnight, and finally add 23:30
		Timespec: t.Format(cronFormat),
		Calls: []Call{{
			Method: "HTTP.Get",
			Params: map[string]interface{}{
				"url": fmt.Sprintf("http://%s:%d/renewSchedules", ip.String(), myPort),
			},
		}},
	}
	reqBody, err := json.Marshal(refresh)
	if err != nil {
		return err
	}
	_, _, err = DoRPCCall(dest, "POST", "Schedule.Create", nil, reqBody)
	return err
}

// Get input state returns true if the controller input is on, false otherwise
func GetInputState(dest fmt.Stringer) (bool, error) {
	body, _, err := DoGet(dest, "Shelly.GetStatus", nil)
	if err != nil {
		return false, err
	}
	return gjson.GetBytes(body, "input:0.state").Bool(), nil
}

// DoRPCCall calls RPC endpoints towards the Shelly. Returns body (or nil if empty), http response code and an error
func DoRPCCall(dest fmt.Stringer, httpMethod, method string, options map[string]string, reqBody []byte) ([]byte, int, error) {
	u := url.URL{
		Scheme: "http",
		Host:   dest.String(),
		Path:   "rpc/" + method,
	}
	if len(options) > 0 {
		values := u.Query()
		for k, v := range options {
			values.Add(k, v)
		}
		u.RawQuery = values.Encode()
	}
	fmt.Println(u.String())
	req, err := http.NewRequest(httpMethod, u.String(), bytes.NewReader(reqBody))
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if r.StatusCode != http.StatusOK {
		add := errors.New(string(body))
		if err != nil {
			add = errors.Wrap(add, fmt.Errorf("\nAdditionally, an error occurred while reading return body: %w", err))
		}
		return body, r.StatusCode, fmt.Errorf("RPC call returned %s (%d) %w", r.Status, r.StatusCode, add)
	}
	return body, http.StatusOK, nil
}

func DoGet(dest fmt.Stringer, method string, options map[string]string) ([]byte, int, error) {
	return DoRPCCall(dest, "GET", method, options, nil)
}

func stringInSlice(s string, sl []string) bool {
	for _, e := range sl {
		if strings.EqualFold(s, e) {
			return true
		}
	}
	return false
}

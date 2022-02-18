// Package shelly implements calls to a shelly unit using HTTP RPC
package shelly

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func GetSchedules() (Schedules, error) {
	url := "http://192.168.0.25/rpc/Schedule.List"
	r, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	var schedules Schedule
	err = json.Unmarshal(body, &schedules)

	return schedules.Jobs, err
}

func enableDisableSchedules(enable bool, ids ...int) error {
	url := "http://192.168.0.25/rpc/Schedule.Update"
	for _, id := range ids {
		r, err := http.Get(url + fmt.Sprintf("?id=%d&enable=%t", id, enable))
		if err != nil {
			return err
		}
		defer r.Body.Close()
		if r.StatusCode != http.StatusOK {
			body, err := io.ReadAll(r.Body)
			add := string(body)
			if err != nil {
				add = fmt.Sprintf("\nAdditionally, an error occurred while reading return body: %s", err)
			}
			return fmt.Errorf("RPC call returned %s (%d) %s", r.Status, r.StatusCode, add)
		}
	}
	return nil
}

func EnableSchedules(ids ...int) error {
	return enableDisableSchedules(true, ids...)
}

func DisableSchedules(ids ...int) error {
	return enableDisableSchedules(false, ids...)
}

func TurnOn() error {
	url := "http://192.168.0.25/rpc/Switch.Set?id=0&on=true"
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		body, err := io.ReadAll(r.Body)
		add := string(body)
		if err != nil {
			add = fmt.Sprintf("\nAdditionally, an error occurred while reading return body: %s", err)
		}
		return fmt.Errorf("RPC call returned %s (%d) %s", r.Status, r.StatusCode, add)
	}
	return nil
}

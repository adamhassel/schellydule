package schellydule

import (
	"errors"
	"fmt"
	"sort"
	"time"

	errors2 "github.com/adamhassel/errors"
	sch "github.com/adamhassel/schedule"
	"github.com/adamhassel/schellydule/shelly"
	"github.com/robfig/cron/v3"
)

type state uint

const (
	stateOn state = iota
	stateOff
)

func (s state) String() string {
	switch s {
	case stateOn:
		return "on"
	case stateOff:
		return "off"
	}
	return "<unknown>"
}

func (s state) State() shelly.State {
	switch s {
	case stateOn:
		return shelly.StateOn
	case stateOff:
		return shelly.StateOff
	default:
		return shelly.StateOff
	}
}

func parseBool(s bool) state {
	if s {
		return stateOn
	}
	return stateOff
}

type schedule struct {
	// setState is the state the schedule sets
	setState state
	trigger  time.Time
}

// State returns the setstate the schedule sets
func (s schedule) State() state {
	return s.setState
}

// TriggerTime returns the time the schedules is triggered
func (s schedule) TriggerTime() time.Time {
	return s.trigger
}

type schedules []schedule

type PairedSchedule struct {
	On  time.Time
	Off time.Time
}

func ParseSchedule(s shelly.JobSpec) (schedule, error) {
	var rv schedule
	for _, c := range s.Calls {
		if c.Method == "switch.set" {
			state, ok := c.Params["on"].(bool)
			if ok {
				rv.setState = parseBool(state)
				break
			}
		}
	}

	t, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Second).Parse(s.Timespec)
	if err != nil {
		return schedule{}, err
	}
	midnight, _ := time.Parse("060102", time.Now().Format("060102"))
	rv.trigger = t.Next(midnight)

	return rv, nil
}

func (ss schedules) Paired() ([]PairedSchedule, error) {
	var rv = make([]PairedSchedule, 0, len(ss)/2)
	// we need to work on the assumption that schedules are returned pairwise, i.e. 'on' followed by 'off'
	if len(ss)%2 != 0 {
		return nil, errors.New("uneven list item numbers")
	}
	fmt.Printf("%+v\n", ss)
	// because we know the list is even-length, we can at least do the first things
	// without checking (since it can be 0 or 2+ long, and if zero, the loop never
	// iterates)
	for i := 0; i < len(ss); {
		if ss[i].setState != stateOn || ss[i+1].setState != stateOff {
			return nil, fmt.Errorf("unexpected state in elements: idx %d was %s (expected 'On'), idx %d was %s (expected 'Off')",
				i, ss[i].setState.String(), i+1, ss[i+1].setState.String())
		}
		rv = append(rv, PairedSchedule{
			On:  ss[i].trigger,
			Off: ss[i+1].trigger,
		})
		i += 2
	}
	return rv, nil
}

func ScheduleToPaired(in shelly.Schedules) ([]PairedSchedule, error) {
	var ss schedules
	for _, s := range in {
		tmp, err := ParseSchedule(s)
		if err != nil {
			return nil, err
		}
		ss = append(ss, tmp)
	}
	return ss.Paired()
}

/*
func PowerPricesSchedule(h sch.HourPrices) []PairedSchedule {
	e := make([]PairedSchedule, 0)
	var ss PairedSchedule
	var l = len(h)
	var new bool
	for i, hp := range h {
		if i == 0 || new {
			ss.On, _ = time.Parse("15", fmt.Sprintf("%d", hp.Hour))
			new = false
		}
		if i == l-1 {
			// last iteration
			ss.Off, _ = time.Parse("15", fmt.Sprintf("%d", hp.Hour+1))
			e = append(e, ss)
			break
		}
		// If the next entry is the next hour, just skip.
		if h[i+1].Hour == hp.Hour+1 {
			continue
		}
		ss.Off, _ = time.Parse("15", fmt.Sprintf("%d", hp.Hour+1))
		e = append(e, ss)
		new = true
	}
	return e
}
*/

// FindMatching will search s for the JobSpec mathcing j. For example, if j is an
// 'on' Jobspec, it will return the Jobspec that turns it back off. If j is an
// 'off' JobSpec, it'll return the jobspec that turned it on
func FindMatching(j shelly.JobSpec, s shelly.Schedules) (shelly.JobSpec, error) {
	sched, err := ParseSchedule(j)
	if err != nil {
		return shelly.JobSpec{}, err
	}
	searchstate := !sched.State().State()
	indexes := make([]int, 0)
	for i, e := range s {
		job, err := ParseSchedule(e)
		if err != nil {
			return shelly.JobSpec{}, err
		}
		if job.State().State() != searchstate {
			continue
		}
		switch shelly.State(searchstate) {
		case shelly.StateOff:
			if job.TriggerTime().After(sched.TriggerTime()) {
				indexes = append(indexes, i)
				continue
			}
		case shelly.StateOn:
			if job.TriggerTime().Before(sched.TriggerTime()) {
				indexes = append(indexes, i)
				continue
			}
		}
	}
	// find the correct time (which is the one with the highest/lowest, depending on mathing) in the list
	if len(indexes) == 0 {
		return shelly.JobSpec{}, errors2.New("no matching jobspec")
	}
	sort.Ints(indexes)
	var i int
	switch shelly.State(searchstate) {
	case shelly.StateOff:
		i = indexes[0]
	case shelly.StateOn:
		i = indexes[len(indexes)-1]
	}
	return s[i], nil
}

// convert a list of cronjobs to a schedule.Schedule (a list of start/stop times)
func Schedule(s shelly.Schedules) (sch.Schedule, error) {
	var rv = make(sch.Schedule, 0, len(s)/2)
	for _, job := range s {
		js, err := ParseSchedule(job)
		if js.State().State() == shelly.StateOff {
			continue // We're ignoring off switches in this context, since we'll find them by matching from the on switches
		}
		if err != nil {
			return nil, err
		}
		var e sch.Entry
		match, err := FindMatching(job, s)
		if err != nil {
			return nil, err
		}

		e.Start = js.TriggerTime()
		e.Stop, err = match.Time()
		if err != nil {
			return nil, err
		}
		rv = append(rv, e)
	}
	return rv, nil
}

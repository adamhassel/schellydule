package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/adamhassel/errors"
	"github.com/adamhassel/power"
	"github.com/adamhassel/power/httpapi"
	"github.com/adamhassel/schedule"
	"github.com/adamhassel/schellydule"
	"github.com/adamhassel/schellydule/config"
	contx "github.com/adamhassel/schellydule/contx"
	"github.com/adamhassel/schellydule/shelly"
)

const defaultPort = 8080

var confFile string

//var conf config.Config
var port int

var ErrInvalidIP = errors.New("invalid IP in query")

func init() {
	flag.StringVar(&confFile, "c", "schedule.conf", "location of configuration file.")
	flag.IntVar(&port, "p", defaultPort, "port to listen on")
}

func main() {
	flag.Parse()
	conf, err := config.LoadConfig(confFile)
	if err != nil {
		log.Fatalf("error reading conf: %s", err)
	}
	if conf.MID() == "" || conf.Token() == "" {
		log.Fatal("MID or Token invalid")
	}

	if p := conf.Port(); p != 0 && port != defaultPort {
		port = p
	}
	http.HandleFunc("/enableSchedules", enableScheduleHandler)
	http.HandleFunc("/disableSchedules", disableScheduleHandler)
	http.HandleFunc("/renewSchedules", renewSchedulesHandler)
	http.HandleFunc("/showSchedules", showSchedulesHandler)

	http.HandleFunc("/getInput", getInputHandler)

	http.HandleFunc("/powerPrices", httpapi.GetPowerPricesConfigHandler(conf, true))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

// getIP returns the IP portion of the remoteAddr string
func parseIP(remoteAddr string) (net.IP, error) {
	ipaddr, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return nil, err
	}
	return net.ParseIP(ipaddr), nil
}

// getIP returns an IP from either the query parameter (if allowed), the
// configuration or sets it to the originating request's IP.
func getIP(req *http.Request, allowInQuery bool) (net.IP, error) {
	query := req.URL.Query()
	in := query.Get("ip")
	ip := net.ParseIP(in)
	// if the input IP is not valid, and we allow the parameter, return error
	if ip == nil && in != "" && allowInQuery {
		return nil, ErrInvalidIP
	}
	if ip == nil || !allowInQuery {
		// If we have a configured IP, use that
		if i := config.GetConf().IP(); i != nil {
			return i, nil
		}
		var err error
		ip, err = parseIP(req.RemoteAddr)
		if err != nil {
			return nil, err
		}
	}
	return ip, nil
}

func enableScheduleHandler(w http.ResponseWriter, req *http.Request) {
	// find the originating IP, where we'll be sending the callbacks
	ctx := contx.ProcessCommon(req)
	ip, err := getIP(req, true)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrInvalidIP) {
			status = http.StatusBadRequest
		}
		setStatusMsg(w, status, err.Error())
		return
	}
	//	1. Get list of all schedules
	schedules, err := shelly.GetSchedules(ctx, ip)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	// 2. Set switch according to schedule
	if err := setSwitchToSchedule(ctx, ip, schedules); err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	//  3. Enable schedules
	var ids = make([]int, 0, len(schedules))
	for _, s := range schedules {
		if !s.HasMethod("switch.set") {
			continue
		}
		ids = append(ids, s.Id)
	}
	if err := shelly.EnableSchedules(ctx, ip, ids...); err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	io.WriteString(w, "Schedule is on\n")
	fmt.Println("schedule on")
}

// setSwitchToSchedule refreshes the on/off state according to the schedule
func setSwitchToSchedule(ctx context.Context, ip fmt.Stringer, schedules shelly.Schedules) error {
	paired, err := schellydule.ScheduleToPaired(schedules)
	if err != nil {
		return err
	}
	// Determine if the schedules currently demand on or off
	now := time.Now()
	var on bool
	for _, p := range paired {
		fmt.Printf("On at %s, off at %s\n", p.On.Format("15:04"), p.Off.Format("15:04"))
		if now.After(p.On) && now.Before(p.Off) {
			fmt.Printf("%s is after %s, but still before %s\n", now.Format("15:04"), p.On.Format("15:04"), p.Off.Format("15:04"))
			on = true
			break
		}
	}
	//  3. Set switch to what the schedules demand
	return shelly.SetSwitch(ctx, ip, shelly.State(on))
}

func disableScheduleHandler(w http.ResponseWriter, req *http.Request) {
	ctx := contx.ProcessCommon(req)
	ip, err := getIP(req, true)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrInvalidIP) {
			status = http.StatusBadRequest
		}
		setStatusMsg(w, status, err.Error())
		return
	}
	// 1. Set switch "on"
	if err := shelly.TurnOn(ctx, ip); err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	// 2. Get list of all schedules
	schedules, err := shelly.GetSchedules(nil, ip)
	if err != nil {
		fmt.Println("getschedules", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	// 3. Disable schedules
	// get IDs
	var ids = make([]int, 0, len(schedules))
	for _, s := range schedules {
		if !s.HasMethod("switch.set") {
			continue
		}
		ids = append(ids, s.Id)
	}
	if err := shelly.DisableSchedules(ctx, ip, ids...); err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	io.WriteString(w, "Schedule is off\n")
	fmt.Println("schedule Off")
}

// renewSchedulesHandler will flush existing schedules and generate a new set.
// Should only be called after between 00:00 and 01:00 local time, and will return 400 if not
// (unless override active)
func renewSchedulesHandler(w http.ResponseWriter, req *http.Request) {
	ctx := contx.ProcessCommon(req)
	ip, err := getIP(req, true)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrInvalidIP) {
			status = http.StatusBadRequest
		}
		setStatusMsg(w, status, err.Error())
		return
	}

	query := req.URL.Query()

	// override allows you to force this endpoint to work at all hours of the day.
	override, _ := strconv.ParseBool(query.Get("override")) // if parse error, just assume false and continue
	now := time.Now()
	if !override && now.Hour() != 0 {
		setStatusMsg(w, http.StatusBadRequest, "come back between 00:00 and 01:00")
		return
	}
	if err := generateAndSetSchedule(ctx, query, ip); err != nil {
		if errors.Is(err, power.ErrEloverblik) {
			go func() {
				var i uint
				var ok bool
				// retry ev. 10 minutes for 23 hours
				max := uint(23 * time.Hour * 6)
				for RetryWait(!ok, &i, max, 10*time.Minute, nil) {
					err = generateAndSetSchedule(ctx, query, ip)
					ok = err == nil
				}
				log.Printf("after %d attempt(s), the result was %s", i, err)
			}()
			log.Print("error contacting eloverblik, retrying")
			setStatusMsg(w, http.StatusAccepted, err)
		} else {
			log.Print("error generating schedule")
			setStatusMsg(w, http.StatusBadGateway, err)
		}
	}
	return
}

// RetryWait will increment counter, sleep for `sleep` and return true if retry should be attempted (`retryif` is true and attempts remaining).
// Returns `false` immediately (not incrementing counter) if `retryif` is false or max retries exceeded.
// Use for limited retries with (possibly zero-length) pause. Does not sleep on initial attempt (assuming counter starts at zero).
// If `f` is non-nil, f will execute on exhaustion of attempts
// Use in a loop, e.g. like:
// func f() error {
//   var i uint
//   var ok bool
//   f := func() {
//  	log.Printf("attempts failed: %s", err)
//   }
//   for RetryWait(!ok, &i, maxAttempts, 0, f) {
//	     err = do_stuff()
//       ok = err == nil
//   }
//   return err
// }
func RetryWait(retryif bool, counter *uint, max uint, sleep time.Duration, f func()) bool {
	if !retryif {
		return false
	}
	if *counter >= max {
		if f != nil {
			f()
		}
		return false
	}
	if *counter != 0 {
		time.Sleep(sleep)
	}
	*counter++
	return true
}

func generateAndSetSchedule(ctx context.Context, query url.Values, ip fmt.Stringer) error {
	hps, err := reqGenerateSchedule(query, false)
	if err != nil {
		return fmt.Errorf("generateSchedule: %w", err)
	}

	enable, err := shelly.GetInputState(ctx, ip)
	if err != nil {
		return err
	}

	s := shelly.ShellySchedule(hps, enable)
	// FXIME: this is a hacky workaround, which is a quick-and-dirty fix for cron
	// being annoying and not care about anything outside the current day (the way
	// it's used here).

	//	if len(hps) != s.Hours() {
	log.Printf("Schedule is %d hours, should be %d", hps.Hours(), s.Hours())
	//}

	// Switch off the shelly. Maybe there's a schedule that's
	// currently running, which was supposed to end at midnight. We can stop that
	// now, unless we're running manually with 'override'. If schedules are disabled, don't do this.
	if enable {
		if err := shelly.TurnOff(ctx, ip); err != nil {
			return err
		}
	}

	//Delete all schedules
	if err := shelly.DeleteAllSchedules(ctx, ip); err != nil {
		return err
	}
	if err := shelly.CreateSchedule(ctx, ip, s); err != nil {
		return err
	}

	// Turn shelly on or off according to schedule, if schedules are enabled. If not, don't touch.
	if enable {
		if err := setSwitchToSchedule(ctx, ip, s.Jobs); err != nil {
			return err
		}
	}

	if err := shelly.CreateScheduleRefresherSchedule(ctx, ip, port); err != nil {
		return err
	}
	return nil
}

// reqGenerateSchedule handle request parameters and generates a schedule. if
// `tomorrow` is true, ignores offset and tries to generate for tomorrow.
func reqGenerateSchedule(query url.Values, tomorrow bool) (schedule.Schedule, error) {
	conf := config.GetConf()
	hours, err := strconv.Atoi(query.Get("hours"))
	if err != nil || hours == 0 {
		hours = conf.Hours()
	}

	// offset is a debugging option, that can be used to adjust how far into the
	// future we're looking for power prices. It should be a multiple of 24 hours,
	// but should be 0 during normal ops. If you're manually running the endpoint at
	// a time when tomorrow's power prices are not yet available, and you want this
	// endpoint to regenerate today's schedule, set offset to zero (or any number
	// less than the number of hours left in the day. Same same).
	offset, err := strconv.Atoi(query.Get("offset"))
	if err != nil {
		fmt.Printf("error parsing %s, using 0", query.Get("offset"))
		offset = 0
	}
	if tomorrow {
		offset = 24
	}
	darkHours, err := strconv.Atoi(query.Get("dark"))
	if err != nil {
		darkHours = conf.DarkHours()
	}
	fmt.Println("PARAMS:", hours, darkHours, offset)
	hp, err := generateSchedule(hours, darkHours, time.Duration(offset)*time.Hour)
	log.Printf("generated schedule is %d hours", len(hp))
	if err != nil {
		return schedule.Schedule{}, fmt.Errorf("generateSchedule: %w", err)
	}
	// handle the special case where the last stop-hour is midnight. This creates
	// confusion, because then we might have ambiguity, if there's also a midnight
	// start time. So set that to 23:59 instead (and minute resolution, not seconds, because Shelly doesn't show seconds).
	hps := hp.Schedule()
	for i, j := range hps {
		if t := j.Stop; t.Hour() == 0 {
			hps[i].Stop = t.Add(-1 * time.Minute)
		}
	}
	log.Printf("hps: %#v", hps)
	return hps, nil
}

// showSchedulesHandler is a GET controller, that returns the currently configured schedule
func showSchedulesHandler(w http.ResponseWriter, req *http.Request) {
	ip, err := getIP(req, true)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrInvalidIP) {
			status = http.StatusBadRequest
		}
		setStatusMsg(w, status, err.Error())
		return
	}
	q := req.URL.Query()
	ws := q.Get("watts")
	watts := 1000.0
	if ws != "" {
		watts, err = strconv.ParseFloat(ws, 64)
		if err != nil {
			setStatusMsg(w, http.StatusBadRequest, err)
			return
		}
	}
	tomorrow, err := strconv.ParseBool(q.Get("tomorrow"))
	if err != nil {
		fmt.Printf("error parsing bool from '%s', assuming false", q.Get("tomorrow"))
	}
	recalc, err := strconv.ParseBool(q.Get("recalc"))
	if err != nil {
		fmt.Printf("error parsing bool from '%s', assuming false", q.Get("recalc"))
	}

	var parsed schedule.Schedule
	if tomorrow || recalc {
		var err error
		parsed, err = reqGenerateSchedule(q, tomorrow)
		if err != nil {
			setStatusMsg(w, http.StatusInternalServerError, err)
			return
		}
	} else {
		schedules, err := shelly.GetSchedules(nil, ip)
		if err != nil {
			setStatusMsg(w, http.StatusBadGateway, err)
			return
		}

		parsed, err = schellydule.Schedule(schedules)
		if err != nil {
			setStatusMsg(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	var out []byte
	if out, err = json.Marshal(parsed.Map(watts)); err != nil {
		setStatusMsg(w, http.StatusInternalServerError, err.Error())
		return
	}
	io.WriteString(w, string(out))
}

func generateSchedule(length, maxDark int, offset time.Duration) (schedule.HourPrices, error) {
	conf := config.GetConf()
	tomorrow := schedule.Hour(time.Now().Add(offset), 0)
	prices, err := power.Prices(tomorrow, tomorrow.Add(24*time.Hour), conf, true)
	if err != nil {
		return nil, err
	}
	list := schedule.FPToHourPrices(prices)

	return list.NCheapest(length, maxDark)
}

func setStatusMsg(w http.ResponseWriter, status int, msg interface{}) {
	var m string
	switch s := msg.(type) {
	case error:
		m = s.Error()
	case string:
		m = s
	case fmt.Stringer:
		m = s.String()
	case []byte:
		m = string(s)
	}
	w.WriteHeader(status)
	w.Write([]byte(m))
}

func getInputHandler(w http.ResponseWriter, req *http.Request) {
	ctx := contx.ProcessCommon(req)
	ip, err := getIP(req, true)
	if err != nil {
		setStatusMsg(w, http.StatusBadRequest, err)
		return
	}
	state, err := shelly.GetInputState(ctx, ip)
	if err != nil {
		setStatusMsg(w, http.StatusBadGateway, err)
		return
	}
	setStatusMsg(w, http.StatusOK, fmt.Sprintf("%t", state))
}

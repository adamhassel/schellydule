package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/adamhassel/power"
	"github.com/adamhassel/power/entities/config"
	"github.com/adamhassel/schedule"
	"github.com/adamhassel/schellydule"
	"github.com/adamhassel/schellydule/shelly"
)

var confFile string
var conf config.Config

func init() {
	flag.StringVar(&confFile, "c", "schedule.conf", "location of configuration file.")
}

func main() {
	flag.Parse()
	var conf config.Config
	conf, err := config.LoadConfig(confFile)
	if err != nil {
		log.Fatalf("error reading conf: %s", err)
	}
	if conf.MID() == "" || conf.Token() == "" {
		log.Fatal("MID or Token invalid")
	}

	http.HandleFunc("/enableSchedule", enableScheduleHandler)
	http.HandleFunc("/disableSchedule", disableScheduleHandler)
	http.HandleFunc("/renewSchedules", renewSchedulesHandler)
	http.HandleFunc("/showSchedules", showSchedulesHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))

	/*
		s := generateSchedule(8, 2)
		p := schellydule.PowerPricesSchedule(s)

		for _, e := range p {
			fmt.Printf("%+v\n", e)
		}

		/*

			s, err := shelly.GetSchedules()
			if err != nil {
				log.Fatal(err)
			}

			paired, err := schellydule.ScheduleToPaired(s)
			if err != nil {
				log.Fatal(err)
			}

			for i, p := range paired {
				fmt.Printf("idx %d : Start %s, Stop %s", i, p.On.Format("15:04"), p.Off.Format("15:04"))
			}
	*/
}

// getIP returns the IP where the request shiould go.
// TODO: allow config override
func getIP(remoteAddr string) (net.IP, error) {
	ipaddr, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return nil, err
	}
	return net.ParseIP(ipaddr), nil
}

func enableScheduleHandler(w http.ResponseWriter, req *http.Request) {
	// find the originating IP, where we'll be sending the callbacks
	ip, err := getIP(req.RemoteAddr)
	if err != nil {
		setStatusMsg(w, http.StatusInternalServerError, err.Error())
		return
	}
	//	1. Get list of all schedules
	schedules, err := shelly.GetSchedules(ip)
	if err != nil {
		log.Fatal(err)
	}

	paired, err := schellydule.ScheduleToPaired(schedules)
	if err != nil {
		log.Fatal(err)
	}
	//  2. Determine if the schedules currently demand on or off
	now := time.Now()
	var on bool
	for _, p := range paired {
		if now.After(p.On) && now.Before(p.Off) {
			on = true
			break
		}
	}
	//  3. Set switch to what the schedules demand
	if err := shelly.SetSwitch(ip, shelly.State(on)); err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	//  4. Enable schedule
	var ids = make([]int, len(schedules))
	for i, s := range schedules {
		ids[i] = s.Id
	}
	if err := shelly.EnableSchedules(ip, ids...); err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	io.WriteString(w, "Schedule is on\n")
	fmt.Println("schedule on")
}
func disableScheduleHandler(w http.ResponseWriter, req *http.Request) {
	ip, err := getIP(req.RemoteAddr)
	if err != nil {
		setStatusMsg(w, http.StatusInternalServerError, err.Error())
		return
	}
	// 1. Get list of all schedules
	schedules, err := shelly.GetSchedules(ip)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	// 2. Disable schedules
	// get IDs
	var ids = make([]int, len(schedules))
	for i, s := range schedules {
		ids[i] = s.Id
	}
	if err := shelly.DisableSchedules(ip, ids...); err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	// 3. Set switch "on"
	if err := shelly.TurnOn(ip); err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	io.WriteString(w, "Schedule is off\n")
	fmt.Println("schedule Off")
}

// renewSchedulesHandler will flush existing schedules and generate a new set.
// Should only be called after 2300 local time, and will return 400 if not
// (unless override active)
func renewSchedulesHandler(w http.ResponseWriter, req *http.Request) {
	// get number of hours and max dark hours from request
	query := req.URL.Query()
	hours, err := strconv.Atoi(query.Get("hours"))
	if err != nil {
		log.Println("no hours, defaulting to 12")
		hours = 12
	}
	darkHours, err := strconv.Atoi(query.Get("dark"))
	if err != nil {
		log.Println("no darkHours, defaulting to 3")
		darkHours = 3
	}
	override, _ := strconv.ParseBool(query.Get("override")) // if parse error, just assume false and continue
	now := time.Now()
	if !override && now.Hour() < 23 {
		setStatusMsg(w, http.StatusBadRequest, "come back after 23:00")
		return
	}

	_ = generateSchedule(hours, darkHours)
}

// showSchedulesHandler is a GET controller, that returns the currently configured schedule
func showSchedulesHandler(w http.ResponseWriter, req *http.Request) {
	ip, err := getIP(req.RemoteAddr)
	ip = net.ParseIP("192.168.0.26")
	if err != nil {
		setStatusMsg(w, http.StatusInternalServerError, err.Error())
		return
	}
	schedules, err := shelly.GetSchedules(ip)
	parsed, err := schellydule.Schedule(schedules)
	if err != nil {
		setStatusMsg(w, http.StatusInternalServerError, err.Error())
		return
	}
	var out []byte
	if out, err = json.Marshal(parsed); err != nil {
		setStatusMsg(w, http.StatusInternalServerError, err.Error())
		return
	}
	io.WriteString(w, string(out))
}

func generateSchedule(length, maxDark int) schedule.HourPrices {
	conf := config.GetConf()
	tomorrow := schedule.Hour(time.Now().Add(24*time.Hour), 0)
	prices, err := power.Prices(tomorrow, tomorrow.Add(24*time.Hour), conf.MID(), conf.Token())
	if err != nil {
		log.Fatal(err)
	}
	list, err := schedule.FPToHourPrices(prices).PruneNightHours(maxDark)

	if err != nil {
		log.Println(err)
	}

	return list.NCheapest(length)
}

func generateScheduleHandler(w http.ResponseWriter, req *http.Request) {

}

func setStatusMsg(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

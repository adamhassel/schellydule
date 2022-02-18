package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/adamhassel/schedule"
	"github.com/adamhassel/schellydule"
	"github.com/adamhassel/schellydule/shelly"
)

func main() {
	/*
		http.HandleFunc("/enableSchedule", enableScheduleHandler)
		http.HandleFunc("/disableSchedule", disableScheduleHandler)
		log.Fatal(http.ListenAndServe(":8080", nil))
	*/

	s := getSchedule()
	p := schellydule.PowerPricesSchedule(s)

	for _, e := range p {
		fmt.Printf("%#v\n", e)
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

func enableScheduleHandler(w http.ResponseWriter, req *http.Request) {

	//	1. Get list of all schedules
	schedules, err := shelly.GetSchedules()
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
	if on {
		if err := shelly.TurnOn(); err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
	}
	//  4. Enable schedule
	var ids = make([]int, len(schedules))
	for i, s := range schedules {
		ids[i] = s.Id
	}
	if err := shelly.EnableSchedules(ids...); err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	io.WriteString(w, "Schedule is on\n")
	fmt.Println("schedule on")
}
func disableScheduleHandler(w http.ResponseWriter, req *http.Request) {

	// 1. Get list of all schedules
	schedules, err := shelly.GetSchedules()
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
	if err := shelly.DisableSchedules(ids...); err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	// 3. Set switch "on"
	if err := shelly.TurnOn(); err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	io.WriteString(w, "Schedule is off\n")
	fmt.Println("schedule Off")
}
func getSchedule() schedule.HourPrices {
	list, err := schedule.Example.PruneNightHours(3)
	if err != nil {
		log.Println(err)
	}
	log.Println("len ", len(schedule.Example))
	c := list.NCheapest(12)
	return c
}
func generateScheduleHandler(w http.ResponseWriter, req *http.Request) {

}

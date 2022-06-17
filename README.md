# Schellydule (for lack of a better name): power schedule for Shelly relays based on power prices


This small web service will act as a place to send webhooks from your shelly to periodically (like, daily) update its power schedule to optimize power usage based on hour-by-hour power prices.

Features (either done or planned out):
* Number of hours to schedule
* Maximum hours allowed to run at night (based on geoIP sunrise/sunset times)
* Endpoint to generate a new schedule, and to toggle if schedules are enabled or not
* Endpoint to return the current schedule, so you can display it on a dashboard or similar
* Autodetection of shelly IP for callbacks
* Support for enabling/disabling schedules via an HTTP call -- I use this with a flip switch to quickly disable schedules if I want to run the appliance manually (yeah, I'll explain how it works eventually).
* Configuration of shelly IP

Feature suggestions:
* Fixed interval to run daily (i.e., run from 8-10 no matter the price)
* Automatic schedule length based on pool size and pump effect (for scheduling pool pump schedules, which is the motivation for the project in the first place)
* Price estimate, given the effect of the appliance connected to the Shelly, possibly with auto estimation for Shelly PM models, which seem to keep power usage stats.


This is still a WIP, so there are probably a bunch of bugs.

Also, if you have a better name suggestion, let me know :D

## Prerequisites

 * Go 1.18+, download at https://go.dev

## Installation

    $ go install github.com/adamhassel/schellydule/cmd/sched@latest

## Setup

 You'll need two things to use this: 

 * A shelly smart switch (I have tested with Shelly Plus 1 and Shelly Plus 1PM).
 * A computer that can run the webservice, preferably 24/7, but at least when you want to refresh the schedule, which happens nightly at 23:30 / 11:30pm.

I am running this on a QNAP Nas, but any Linux based host will do, and probably
any Windows or OSX or other Unix/BSD based host as well, although I haven't tested it. You're probably
fine to use a Raspberry Pi or similar. The only OS dependent thing is some time
zone magic. I'll probably put in something to work around that eventually,
though. Also, maybe at some point, docker the things. For now though, simply
run the executable.

### Connecting the Shelly

I'm using this to run my pool filtration pump. It needs to run for a number of
hours each day to circulate the water through the filter 2 to 3 times. I want
to minimize running at night, because it's during the day the water is
"stressed" the most: swimming, sunlight (which accelerates algae growth) etc.

Sometimes I also want to control my pump manually. Maybe I need to adjust
chlorine or pH outside the pump schedule. So I've built the option to control
that through two switches. Check the wiring diagram for how those are connected
to the Shelly.

![Wiring diagram](wiring.png?raw=true "Wiring")

If you don't want or need to disable/enable the schedules with a switch, you
can omit the "Line" connection into the "SW" port. You can control this with a
web hook manually however you want, but I find it convenient that there's a
physical switch right next to my pool pump, that allows me to override
schedules...

I'll leave getting the Shelly connected to your Wifi (and making sure you have
WiFi coverage in your pump room) as an exercise forthe reader.

### Configuring switching scheduled power on/off (making it possible to manually control the power at will with a switch).

* Open the "Channel Settings" in the Shelly WebUI, and make sure you have "Relay Type" set to "Detached" and "Power on default" to "On".

* Open the "Webhooks" page and configure two webhooks:
	* Webhook 1: Active: 24h, Condition: When input is off, URL: `http://[server:port]/disableSchedules`
	* Webhook 2: Active: 24h, Condition: When input is on, URL: `http://[server:port]/enableSchedules`

`[server:port]` is of course the IP/hostname and port of the computer where you're running this webservice.

If you only want to run the Shelly schedule, and don't want the option to
disable and enable it with a switch, this is purely optional. As mentioned
above, you can call the `enable/disableSchedules` endpoints directly however
you want.

### Initial schedule generation

You'll need two pieces of information in order to obtain the power prices used:
An API Key and a Measuring Point ID. Both of these things you get from
eloverblik.dk. The API token is generated by following the instructions in
[this PDF](https://energinet.dk/-/media/365F242312244CC284EA9EDF0F9F0AAA.pdf),
and the Measurement ID is obtained from your personal data. If you have
multiple measurement points, be sure to pick the one whatever's running your
shelly/appliance gets its power from.

Fill those two pieces of information into the config file, which should be put
in the same directory as you're running the service from. Or, you can supply an
absolute path with the `-c /path/to/config.conf` command line option.

Check the included example config for other options.

Now, start the service:

    $ ./sched

From a random computer, make a call to the service to set up the first schedule:

	$ curl "http://[server:port]/renewSchedule?override=true&offset=0&ip=[shelly_ip]"

Again, `[server:port]` is of course the IP/hostname and port of the computer
where you're running this webservice.

Note that if you have configured the Shelly's IP address in the config file,
the `&ip=...` part in the address is not necessary.

The other options are, for reference:

* `override` overrides the restriction on the endpoint to only run after 23:00. This is to not inadvertently interfere with the current day's schedule.
* `offset=0` ensures that the schedule you're setting up is for today, rather than tomorrow. Because prices are not available for the next day until as late as 14:00, running without this option too early in the day, may yield an empty schedule.

That's it! You're all set! The schedules will automatically regenerate daily at 23:30.

Check the schedule generated in the Shelly webui (the "Schedules" tab), or by checking the webservice:

	$ curl "http://[shelly_ip]/rpc/Schedules.List

or probably a bit more readable with:

	$ curl "http://[server:port]/showSchedules?ip=[shelly_ip]"


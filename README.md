# Schellydule (for lack of a better name): power schedule for Shelly relays based on power prices

This small web service will act as a place to send webhooks from your shelly to periodically (like, daily) update its power schedule to optimize power usage based on hour-by-hour power prices.

Features (either done or planned out):
* Number of hours to schedule
* Maximum hours allowed to run at night (based on geoIP sunrise/sunset times)
* Endpoint to generate a new schedule, and to toggle if schedules are enabled or not
* Endpoint to return the current schedule, so you can display it on a dashboard or similar
* Autodetection of shelly IP for callbacks
* Price estimate, given the effect of the appliance connected to the Shelly

Feature suggestions:
* Fixed interval to run daily (i.e., run from 8-10 no matter the price)
* Configuration of shelly IP
* Automatic schedule length based on pool size and pump effect (for scheduling pool pump schedules, which is the motivation for the project in the first place)

This is still a WIP, so you probably don't want to use it yet ;)

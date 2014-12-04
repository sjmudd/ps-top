// github.com/sjmudd/wait_info - routines for managing when we
// collect information from MySQL.
package wait_info

import (
	"github.com/sjmudd/pstop/lib"
	"time"
)

// over-schedule the next wait by this time _iff__ the last scheduled time is in the past.
const extra_delay = 200 * time.Millisecond

// used to record when we need to collect information from MySQL
type WaitInfo struct {
	last_collected   time.Time
	collect_interval time.Duration
}

// returns the configured wait interval between collecting data.
func (wi *WaitInfo) WaitInterval() time.Duration {
	return wi.collect_interval
}

// record we have just collected data now.
func (wi *WaitInfo) CollectedNow() {
	wi.SetCollected(time.Now())
}

// Change the desired collection interval to a new value
func (wi *WaitInfo) SetWaitInterval(required_interval time.Duration) {
	wi.collect_interval = required_interval
}

// Set the time we last collected information
func (wi *WaitInfo) SetCollected(collect_time time.Time) {
	wi.last_collected = collect_time
	lib.Logger.Println("WaitInfo.SetCollected() last_collected=", wi.last_collected)
}

// Return when we last collection happened
func (wi WaitInfo) LastCollected() time.Time {
	return wi.last_collected
}

// return the amount of time to wait before doing the next collection
func (wi WaitInfo) TimeToWait() time.Duration {
	now := time.Now()
	lib.Logger.Println("WaitInfo.TimeToWait() now: ", now)

	next_time := wi.last_collected.Add(wi.collect_interval)
	lib.Logger.Println("WaitInfo.TimeToWait() next_time: ", next_time)
	if next_time.Before(now) {
		lib.Logger.Println("WaitInfo.TimeToWait() next_time scheduled time in the past, so schedule", extra_delay, "after", now)
		next_time = now
		next_time.Add(extra_delay) // add a deliberate tiny delay
		lib.Logger.Println("WaitInfo.TimeToWait() next_time: ", next_time, "(corrected)")
	}
	wait_time := next_time.Sub(now)
	lib.Logger.Println("WaitInfo.TimeToWait() returning wait_time:", wait_time)

	return wait_time
}

// return a channel which will be written to at the next 'scheduled' time.
func (wi WaitInfo) WaitNextPeriod() <-chan time.Time {
	return time.After(wi.TimeToWait())
}

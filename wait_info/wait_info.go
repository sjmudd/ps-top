package wait_info

import (
	"github.com/sjmudd/pstop/lib"
	"time"
)

const extra_delay = 200 * time.Millisecond

// used to record when we need to wa
type WaitInfo struct {
	last_collected   time.Time
	collect_interval time.Duration
}

// return the configured wait interval
func (wi *WaitInfo) WaitInterval() time.Duration {
	return wi.collect_interval
}

// recognise we've done a collection
func (wi *WaitInfo) CollectedNow() {
	wi.SetCollected(time.Now())
}

// change the configured wait interval
func (wi *WaitInfo) SetWaitInterval(required_interval time.Duration) {
	wi.collect_interval = required_interval
}

// set the time the last collection happened
func (wi *WaitInfo) SetCollected(collect_time time.Time) {
	wi.last_collected = collect_time
	lib.Logger.Println("WaitInfo.SetCollected() last_collected=", wi.last_collected)
}

// return the time the last collection happened
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

// behave like time.After()
func (wi WaitInfo) WaitNextPeriod() <-chan time.Time {
	return time.After(wi.TimeToWait())
}

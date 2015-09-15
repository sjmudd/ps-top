// Package wait_info contains routines for managing when we
// collect information from MySQL.
package wait_info

import (
	"github.com/sjmudd/ps-top/logger"
	"time"
)

// over-schedule the next wait by this time _iff__ the last scheduled time is in the past.
const extraDelay = 200 * time.Millisecond

// WaitInfo is used to record when we need to collect information from MySQL
type WaitInfo struct {
	lastCollected   time.Time
	collectInterval time.Duration
}

// WaitInterval returns the configured wait interval between collecting data.
func (wi *WaitInfo) WaitInterval() time.Duration {
	return wi.collectInterval
}

// CollectedNow records we have just collected data now.
func (wi *WaitInfo) CollectedNow() {
	wi.SetCollected(time.Now())
}

// SetWaitInterval changes the desired collection interval to a new value
func (wi *WaitInfo) SetWaitInterval(requiredInterval time.Duration) {
	wi.collectInterval = requiredInterval
}

// SetCollected sets the time we last collected information
func (wi *WaitInfo) SetCollected(collectTime time.Time) {
	wi.lastCollected = collectTime
	logger.Println("WaitInfo.SetCollected() lastCollected=", wi.lastCollected)
}

// LastCollected returns when the last collection happened
func (wi WaitInfo) LastCollected() time.Time {
	logger.Println("WaitInfo.LastCollected()", wi, ",", wi.lastCollected)
	return wi.lastCollected
}

// TimeToWait returns the amount of time to wait before doing the next collection
func (wi WaitInfo) TimeToWait() time.Duration {
	now := time.Now()
	logger.Println("WaitInfo.TimeToWait() now: ", now)

	nextTime := wi.lastCollected.Add(wi.collectInterval)
	logger.Println("WaitInfo.TimeToWait() nextTime: ", nextTime)
	if nextTime.Before(now) {
		logger.Println("WaitInfo.TimeToWait() nextTime scheduled time in the past, so schedule", extraDelay, "after", now)
		nextTime = now
		nextTime.Add(extraDelay) // add a deliberate tiny delay
		logger.Println("WaitInfo.TimeToWait() nextTime: ", nextTime, "(corrected)")
	}
	waitTime := nextTime.Sub(now)
	logger.Println("WaitInfo.TimeToWait() returning waitTime:", waitTime)

	return waitTime
}

// WaitNextPeriod returns a channel which will be written to at the next 'scheduled' time.
func (wi WaitInfo) WaitNextPeriod() <-chan time.Time {
	return time.After(wi.TimeToWait())
}

// Package wait manages waits between data collections from MySQL.
package wait

import (
	"github.com/sjmudd/ps-top/logger"
	"time"
)

// over-schedule the next wait by this time _iff__ the last scheduled time is in the past.
const extraDelay = 200 * time.Millisecond

// Handler records when information was last collected from MySQL and how often it should be collected
type Handler struct {
	lastCollected   time.Time
	collectInterval time.Duration
}

// WaitInterval returns the configured wait interval between collecting data.
func (wi *Handler) WaitInterval() time.Duration {
	return wi.collectInterval
}

// CollectedNow records we have just collected data now.
func (wi *Handler) CollectedNow() {
	wi.SetCollected(time.Now())
}

// SetWaitInterval changes the desired collection interval to a new value
func (wi *Handler) SetWaitInterval(requiredInterval time.Duration) {
	wi.collectInterval = requiredInterval
}

// SetCollected sets the time we last collected information
func (wi *Handler) SetCollected(collectTime time.Time) {
	wi.lastCollected = collectTime
	logger.Println("Handler.SetCollected() lastCollected=", wi.lastCollected)
}

// LastCollected returns when the last collection happened
func (wi Handler) LastCollected() time.Time {
	logger.Println("Handler.LastCollected()", wi, ",", wi.lastCollected)
	return wi.lastCollected
}

// TimeToWait returns the amount of time to wait before doing the next collection
func (wi Handler) TimeToWait() time.Duration {
	now := time.Now()
	logger.Println("Handler.TimeToWait() now: ", now)

	nextTime := wi.lastCollected.Add(wi.collectInterval)
	logger.Println("Handler.TimeToWait() nextTime: ", nextTime)
	if nextTime.Before(now) {
		logger.Println("Handler.TimeToWait() nextTime scheduled time in the past, so schedule", extraDelay, "after", now)
		nextTime = now
		nextTime.Add(extraDelay) // add a deliberate tiny delay
		logger.Println("Handler.TimeToWait() nextTime: ", nextTime, "(corrected)")
	}
	waitTime := nextTime.Sub(now)
	logger.Println("Handler.TimeToWait() returning waitTime:", waitTime)

	return waitTime
}

// WaitUntilNextPeriod returns a channel which will be written to at the next 'scheduled' time.
func (wi Handler) WaitUntilNextPeriod() <-chan time.Time {
	return time.After(wi.TimeToWait())
}

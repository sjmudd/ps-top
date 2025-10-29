// Package wait manages waits between data collections from MySQL.
package wait

import (
	"log"
	"time"
)

// over-schedule the next wait by this time _iff__ the last scheduled time is in the past.
const extraDelay = 200 * time.Millisecond

// timeNow is a function type that returns the current time
type timeNow func() time.Time

// Handler records when information was last collected from MySQL and how often it should be collected
type Handler struct {
	lastCollected   time.Time
	collectInterval time.Duration
	now             timeNow
}

// NewHandler creates a new Handler with default time.Now function
func NewHandler() *Handler {
	return &Handler{
		now: time.Now,
	}
}

// WaitInterval returns the configured wait interval between collecting data.
func (wi *Handler) WaitInterval() time.Duration {
	return wi.collectInterval
}

// CollectedNow records we have just collected data now.
func (wi *Handler) CollectedNow() {
	wi.SetCollected(wi.now())
}

// SetWaitInterval changes the desired collection interval to a new value
func (wi *Handler) SetWaitInterval(requiredInterval time.Duration) {
	wi.collectInterval = requiredInterval
}

// SetCollected sets the time we last collected information
func (wi *Handler) SetCollected(collectTime time.Time) {
	wi.lastCollected = collectTime
	log.Println("Handler.SetCollected() lastCollected=", wi.lastCollected)
}

// LastCollected returns the time when the last collection happened
func (wi Handler) LastCollected() time.Time {
	log.Println("Handler.LastCollected()", wi, ",", wi.lastCollected)
	return wi.lastCollected
}

// TimeToWait returns the amount of time to wait before doing the next collection
func (wi Handler) TimeToWait() time.Duration {
	now := wi.now()
	log.Println("Handler.TimeToWait() now: ", now)

	nextTime := wi.lastCollected.Add(wi.collectInterval)
	log.Println("Handler.TimeToWait() nextTime: ", nextTime)
	if nextTime.Before(now) {
		log.Println("Handler.TimeToWait() nextTime scheduled time in the past, so schedule", extraDelay, "after", now)
		nextTime = now
		nextTime = nextTime.Add(extraDelay) // add a deliberate tiny delay
		log.Println("Handler.TimeToWait() nextTime: ", nextTime, "(corrected)")
	}
	waitTime := nextTime.Sub(now)
	log.Println("Handler.TimeToWait() returning waitTime:", waitTime)

	return waitTime
}

// WaitUntilNextPeriod returns a channel which will be written to at the next 'scheduled' time.
func (wi Handler) WaitUntilNextPeriod() <-chan time.Time {
	return time.After(wi.TimeToWait())
}

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

// Waiter records when information was last collected from MySQL and how often it should be collected
type Waiter struct {
	lastCollected   time.Time
	collectInterval time.Duration
	now             timeNow
}

// NewWaiter creates a new Waiter with default time.Now function
func NewWaiter() *Waiter {
	return &Waiter{
		now: time.Now,
	}
}

// WaitInterval returns the configured wait interval between collecting data.
func (wi *Waiter) WaitInterval() time.Duration {
	return wi.collectInterval
}

// CollectedNow records we have just collected data now.
func (wi *Waiter) CollectedNow() {
	wi.SetCollected(wi.now())
}

// SetWaitInterval changes the desired collection interval to a new value
func (wi *Waiter) SetWaitInterval(requiredInterval time.Duration) {
	wi.collectInterval = requiredInterval
}

// SetCollected sets the time we last collected information
func (wi *Waiter) SetCollected(collectTime time.Time) {
	wi.lastCollected = collectTime
	log.Println("Waiter.SetCollected() lastCollected=", wi.lastCollected)
}

// LastCollected returns the time when the last collection happened
func (wi Waiter) LastCollected() time.Time {
	log.Println("Waiter.LastCollected()", wi, ",", wi.lastCollected)
	return wi.lastCollected
}

// TimeToWait returns the amount of time to wait before doing the next collection
func (wi Waiter) TimeToWait() time.Duration {
	log.Printf("TimeToWait(): wi=%+v\n", wi)
	now := wi.now()
	log.Println("Waiter.TimeToWait() now: ", now)

	nextTime := wi.lastCollected.Add(wi.collectInterval)
	log.Println("Waiter.TimeToWait() nextTime: ", nextTime)
	if nextTime.Before(now) {
		log.Println("Waiter.TimeToWait() nextTime scheduled time in the past, so schedule", extraDelay, "after", now)
		nextTime = now
		nextTime = nextTime.Add(extraDelay) // add a deliberate tiny delay
		log.Println("Waiter.TimeToWait() nextTime: ", nextTime, "(corrected)")
	}
	waitTime := nextTime.Sub(now)
	log.Println("Waiter.TimeToWait() returning waitTime:", waitTime)

	return waitTime
}

// WaitUntilNextPeriod returns a channel which will be written to at the next 'scheduled' time.
func (wi Waiter) WaitUntilNextPeriod() <-chan time.Time {
	return time.After(wi.TimeToWait())
}

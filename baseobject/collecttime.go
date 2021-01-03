// Package baseobject contains the library
// routines for base stuff of an object
package baseobject

import (
	"time"
)

// CollectTime records the first and last time we have collected data
type CollectTime struct {
	first time.Time // the first collection time (for relative data)
	last  time.Time // the last collection time
}

// LastCollectTime is the time we last collected data
func (ct CollectTime) LastCollectTime() time.Time {
	return ct.last
}

// FirstCollectTime is the time we first collected data, statistics were last reset
func (ct CollectTime) FirstCollectTime() time.Time {
	return ct.first
}

// SetFirstCollectTime records the time we first collected data or statistics were reset
func (ct *CollectTime) SetFirstCollectTime(first time.Time) {
	ct.first = first
}

// SetLastCollectTime records the time we last collected data
func (ct *CollectTime) SetLastCollectTime(last time.Time) {
	ct.last = last
}

// Package baseobject contains the library
// routines for base stuff of an object
package baseobject

import (
	"time"
)

// Row holds a row of data from table_lock_waits_summary_by_table
type BaseObject struct {
	intialCollectTime time.Time // the initial collection time (for relative data)
	lastCollectTime   time.Time // the last collection time
	wantRelativeStats bool
}

func (o BaseObject) LastCollectTime() time.Time {
	return o.lastCollectTime
}

// SetNow records the time the data was collected (now)
func (o *BaseObject) SetLastCollectTimeNow() {
	o.lastCollectTime = time.Now()
}

func (o BaseObject) InitialCollectTime() time.Time {
	return o.intialCollectTime
}

func (o *BaseObject) SetInitialCollectTime(initial time.Time) {
	o.intialCollectTime = initial
}

// SetNow records the time the data was collected (now)
func (o *BaseObject) SetInitialCollectTimeNow() {
	o.intialCollectTime = time.Now()
}

// SetWantRelativeStats records whether we want relative stats or not
func (o *BaseObject) SetWantRelativeStats(wantRelativeStats bool) {
	o.wantRelativeStats = wantRelativeStats
}

// WantRelativeStats returns whether we want relative stats or not
func (o BaseObject) WantRelativeStats() bool {
	return o.wantRelativeStats
}

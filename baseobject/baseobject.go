// Package baseobject contains the library
// routines for base stuff of an object
package baseobject

import (
	"time"
)

// Row holds a row of data from table_lock_waits_summary_by_table
type BaseObject struct {
	last              time.Time
	wantRelativeStats bool
}

func (o BaseObject) Last() time.Time {
	return o.last
}

// SetNow records the time the data was collected (now)
func (t *BaseObject) SetNow() {
	t.last = time.Now()
}

func (o *BaseObject) SetLast(last time.Time) {
	o.last = last
}

// SetWantRelativeStats records whether we want relative stats or not
func (o *BaseObject) SetWantRelativeStats(wantRelativeStats bool) {
	o.wantRelativeStats = wantRelativeStats
}

// WantRelativeStats returns whether we want relative stats or not
func (o BaseObject) WantRelativeStats() bool {
	return o.wantRelativeStats
}

// Package mutexlatency contains the library routines for managing the
// events_waits_summary_global_by_event_Name table
package mutexlatency

import (
	"log"
)

// Row contains a row from performance_schema.events_waits_summary_global_by_event_Name
type Row struct {
	Name         string
	SumTimerWait uint64
	CountStar    uint64
}

// subtract the countable values in one row from another
func (row *Row) subtract(other Row) {
	// check for issues here (we have a bug) and log it
	// - this situation should not happen so there's a logic bug somewhere else
	if row.SumTimerWait >= other.SumTimerWait {
		row.SumTimerWait -= other.SumTimerWait
		row.CountStar -= other.CountStar
	} else {
		log.Println("WARNING: Row.subtract() - subtraction problem! (not subtracting)")
		log.Println("row=", row)
		log.Println("other=", other)
	}
}

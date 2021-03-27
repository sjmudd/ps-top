// Package mutex_latency contains the library routines for managing the
// events_waits_summary_global_by_event_Name table
package mutex_latency

import (
	"github.com/sjmudd/ps-top/logger"
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
		logger.Println("WARNING: Row.subtract() - subtraction problem! (not subtracting)")
		logger.Println("row=", row)
		logger.Println("other=", other)
	}
}

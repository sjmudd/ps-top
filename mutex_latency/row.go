// Package mutex_latency contains the library routines for managing the
// events_waits_summary_global_by_event_name table
package mutex_latency

import (
	"fmt"

	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/logger"
)

// Row contains a row from performance_schema.events_waits_summary_global_by_event_name
// Note: upper case names to match the performance_schema column names.
// This type is _not_ meant to be exported.
type Row struct {
	name         string
	sumTimerWait uint64
	countStar    uint64
}

func (row *Row) headings() string {
	return fmt.Sprintf("%10s %8s %8s|%s", "Latency", "MtxCnt", "%", "Mutex Name")
}

// generate a printable result
func (row *Row) rowContent(totals Row) string {
	name := row.name
	if row.countStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %8s %8s|%s",
		lib.FormatTime(row.sumTimerWait),
		lib.FormatAmount(row.countStar),
		lib.FormatPct(lib.MyDivide(row.sumTimerWait, totals.sumTimerWait)),
		name)
}

func (row *Row) add(other Row) {
	row.sumTimerWait += other.sumTimerWait
	row.countStar += other.countStar
}

// subtract the countable values in one row from another
func (row *Row) subtract(other Row) {
	// check for issues here (we have a bug) and log it
	// - this situation should not happen so there's a logic bug somewhere else
	if row.sumTimerWait >= other.sumTimerWait {
		row.sumTimerWait -= other.sumTimerWait
		row.countStar -= other.countStar
	} else {
		logger.Println("WARNING: Row.subtract() - subtraction problem! (not subtracting)")
		logger.Println("row=", row)
		logger.Println("other=", other)
	}
}

// describe a whole row
func (row Row) String() string {
	return fmt.Sprintf("%s|%10s %6s %6s",
		row.name,
		lib.FormatTime(row.sumTimerWait),
		lib.FormatAmount(row.countStar),
		lib.FormatPct(lib.MyDivide(row.sumTimerWait, row.sumTimerWait)))
}

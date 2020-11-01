package stages_latency

import (
	"fmt"

	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/logger"
)

/**************************************************************************

CREATE TABLE `events_stages_summary_global_by_event_name` (
  `EVENT_NAME` varchar(128) NOT NULL,
  `COUNT_STAR` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WAIT` bigint(20) unsigned NOT NULL, // not used
  `AVG_TIMER_WAIT` bigint(20) unsigned NOT NULL, // not used
  `MAX_TIMER_WAIT` bigint(20) unsigned NOT NULL  // not used
) ENGINE=PERFORMANCE_SCHEMA DEFAULT CHARSET=utf8
1 row in set (0.00 sec)

**************************************************************************/

// Row contains the information in one row
type Row struct {
	name         string
	countStar    uint64
	sumTimerWait uint64
}

// add the values of one row to another one
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

// stage headings
func (row *Row) headings() string {
	return fmt.Sprintf("%10s %6s %8s|%s", "Latency", "%", "Counter", "Stage Name")
}

// generate a printable result
func (row *Row) content(totals Row) string {
	name := row.name
	if row.countStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s %8s|%s",
		lib.FormatTime(row.sumTimerWait),
		lib.FormatPct(lib.Divide(row.sumTimerWait, totals.sumTimerWait)),
		lib.FormatAmount(row.countStar),
		name)
}

// String describes a whole row
func (row Row) String() string {
	return fmt.Sprintf("%10s %10s %s",
		lib.FormatTime(row.sumTimerWait),
		lib.FormatAmount(row.countStar),
		row.name)
}

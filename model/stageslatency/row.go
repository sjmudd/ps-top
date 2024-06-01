package stageslatency

import (
	"log"
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
	Name         string
	CountStar    uint64
	SumTimerWait uint64
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

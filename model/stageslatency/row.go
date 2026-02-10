package stageslatency

import (
	"github.com/sjmudd/ps-top/model/common"
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
	common.SubtractCounts(&row.SumTimerWait, &row.CountStar, other.SumTimerWait, other.CountStar, row, other)
}

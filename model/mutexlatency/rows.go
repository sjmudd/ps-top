// Package mutexlatency contains the library routines for managing the
// events_waits_summary_global_by_event_name table
package mutexlatency

import (
	"github.com/sjmudd/ps-top/log"
	"github.com/sjmudd/ps-top/model"
	"github.com/sjmudd/ps-top/model/common"
)

// Rows contains a slice of Row
type Rows []Row

func totals(rows Rows) Row {
	total := Row{Name: "Totals"}

	for _, row := range rows {
		total.SumTimerWait += row.SumTimerWait
		total.CountStar += row.CountStar
	}

	return total
}

func collect(db model.QueryExecutor) Rows {
	const prefix = "wait/synch/"
	var t Rows

	// Collect all information even if it's mainly empty as we may reference it later
	sql := `
SELECT EVENT_NAME, SUM_TIMER_WAIT, COUNT_STAR
FROM events_waits_summary_global_by_event_name
WHERE SUM_TIMER_WAIT > 0
AND (
	EVENT_NAME LIKE 'wait/synch/mutex/%' OR
	EVENT_NAME LIKE 'wait/synch/sxlock/innodb/%' -- FIX: this view name may need changing as it's not a mutex
)
`

	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal(err)
	}

	t = common.Collect(rows, func() (Row, error) {
		var r Row
		if err := rows.Scan(
			&r.Name,
			&r.SumTimerWait,
			&r.CountStar); err != nil {
			return r, err
		}

		// Trim off the leading prefix characters
		if len(r.Name) >= len(prefix) {
			r.Name = r.Name[len(prefix):]
		}

		// Collect all information even if it's mainly empty as we may reference it later
		return r, nil
	})
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	_ = rows.Close()

	return t
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
//
//nolint:unused
func (rows Rows) needsRefresh(otherRows Rows) bool {
	return common.NeedsRefresh(totals(rows).SumTimerWait, totals(otherRows).SumTimerWait)
}

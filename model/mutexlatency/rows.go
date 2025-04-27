// Package mutexlatency contains the library routines for managing the
// events_waits_summary_global_by_event_name table
package mutexlatency

import (
	"database/sql"

	"github.com/sjmudd/ps-top/log"
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

func collect(db *sql.DB) Rows {
	var t Rows

	// we collect all information even if it's mainly empty as we may reference it later
	sql := "SELECT EVENT_NAME, SUM_TIMER_WAIT, COUNT_STAR FROM events_waits_summary_global_by_event_name WHERE SUM_TIMER_WAIT > 0 AND EVENT_NAME LIKE 'wait/synch/mutex/innodb/%'"

	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var r Row
		if err := rows.Scan(
			&r.Name,
			&r.SumTimerWait,
			&r.CountStar); err != nil {
			log.Fatal(err)
		}

		// trim off the leading 'wait/synch/mutex/innodb/'
		if len(r.Name) >= 24 {
			r.Name = r.Name[24:]
		}

		// we collect all information even if it's mainly empty as we may reference it later
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	_ = rows.Close()

	return t
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
func (rows *Rows) subtract(initial Rows) {
	initialByName := make(map[string]int)

	// iterate over rows by name
	for i := range initial {
		initialByName[initial[i].Name] = i
	}

	for i := range *rows {
		name := (*rows)[i].Name
		if _, ok := initialByName[name]; ok {
			initialIndex := initialByName[name]
			(*rows)[i].subtract(initial[initialIndex])
		}
	}
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (rows Rows) needsRefresh(otherRows Rows) bool {
	return totals(rows).SumTimerWait > totals(otherRows).SumTimerWait
}

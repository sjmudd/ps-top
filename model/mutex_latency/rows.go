// Package mutex_latency contains the library routines for managing the
// events_waits_summary_global_by_event_name table
package mutex_latency

import (
	"database/sql"
	"log"
	"sort"
)

// Rows contains a slice of Row
type Rows []Row

func (rows Rows) totals() Row {
	var totals Row
	totals.Name = "Totals"

	for i := range rows {
		totals.add(rows[i])
	}

	return totals
}

func collect(dbh *sql.DB) Rows {
	var t Rows

	// we collect all information even if it's mainly empty as we may reference it later
	sql := "SELECT EVENT_NAME, SUM_TIMER_WAIT, COUNT_STAR FROM events_waits_summary_global_by_event_name WHERE SUM_TIMER_WAIT > 0 AND EVENT_NAME LIKE 'wait/synch/mutex/innodb/%'"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

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

	return t
}

func (rows Rows) Len() int      { return len(rows) }
func (rows Rows) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }

// sort by value (descending) but also by "name" (ascending) if the values are the same
func (rows Rows) Less(i, j int) bool {
	return rows[i].SumTimerWait > rows[j].SumTimerWait
}

func (rows Rows) sort() {
	sort.Sort(rows)
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
	totals := rows.totals()
	otherTotals := otherRows.totals()

	return totals.SumTimerWait > otherTotals.SumTimerWait
}

// Package table_io_latency contains the routines for managing
// performance_schema.table_io_waits_by_table.
package table_io_latency

import (
	"database/sql"
	"log"
	"sort"
	"strings"

	"github.com/sjmudd/ps-top/lib"
)

// Rows contains a set of rows
type Rows []Row

func (rows Rows) totals() Row {
	var totals Row
	totals.name = "Totals"

	for i := range rows {
		totals.add(rows[i])
	}

	return totals
}

func collect(dbh *sql.DB) Rows {
	var t Rows

	// we collect all information even if it's mainly empty as we may reference it later
	sql := "SELECT OBJECT_SCHEMA, OBJECT_NAME, COUNT_STAR, SUM_TIMER_WAIT, COUNT_READ, SUM_TIMER_READ, COUNT_WRITE, SUM_TIMER_WRITE, COUNT_FETCH, SUM_TIMER_FETCH, COUNT_INSERT, SUM_TIMER_INSERT, COUNT_UPDATE, SUM_TIMER_UPDATE, COUNT_DELETE, SUM_TIMER_DELETE FROM table_io_waits_summary_by_table WHERE SUM_TIMER_WAIT > 0"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table string
		var r Row
		if err := rows.Scan(
			&schema,
			&table,
			&r.countStar,
			&r.sumTimerWait,
			&r.countRead,
			&r.sumTimerRead,
			&r.countWrite,
			&r.sumTimerWrite,
			&r.countFetch,
			&r.sumTimerFetch,
			&r.countInsert,
			&r.sumTimerInsert,
			&r.countUpdate,
			&r.sumTimerUpdate,
			&r.countDelete,
			&r.sumTimerDelete); err != nil {
			log.Fatal(err)
		}
		r.name = lib.TableName(schema, table)

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
	return (rows[i].sumTimerWait > rows[j].sumTimerWait) ||
		((rows[i].sumTimerWait == rows[j].sumTimerWait) &&
			(rows[i].name < rows[j].name))
}

// ByOps is used for sorting by the number of operations
type ByOps Rows

func (rows ByOps) Len() int      { return len(rows) }
func (rows ByOps) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }
func (rows ByOps) Less(i, j int) bool {
	return (rows[i].countStar > rows[j].countStar) ||
		((rows[i].sumTimerWait == rows[j].sumTimerWait) &&
			(rows[i].name < rows[j].name))
}

func (rows Rows) sort(wantLatency bool) {
	if wantLatency {
		sort.Sort(rows)
	} else {
		sort.Sort(ByOps(rows))
	}
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
func (rows *Rows) subtract(initial Rows) {
	initialByName := make(map[string]int)

	// iterate over rows by name
	for i := range initial {
		initialByName[initial[i].name] = i
	}

	for i := range *rows {
		rowName := (*rows)[i].name
		if _, ok := initialByName[rowName]; ok {
			initialIndex := initialByName[rowName]
			(*rows)[i].subtract(initial[initialIndex])
		}
	}
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (rows Rows) needsRefresh(otherRows Rows) bool {
	myTotals := rows.totals()
	otherTotals := otherRows.totals()

	return myTotals.sumTimerWait > otherTotals.sumTimerWait
}

// describe a whole table
func (rows Rows) String() string {
	s := make([]string, len(rows))

	for i := range rows {
		s = append(s, rows[i].String())
	}

	return strings.Join(s, "\n")
}

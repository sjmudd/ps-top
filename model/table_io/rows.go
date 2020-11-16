// Package table_io contains the routines for managing
// performance_schema.table_io_waits_by_table.
package table_io

import (
	"database/sql"
	"log"
	"sort"

	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/logger"
	"github.com/sjmudd/ps-top/model/filter"
)

// Rows contains a set of rows
type Rows []Row

func (rows Rows) totals() Row {
	var totals Row
	totals.Name = "Totals"

	for i := range rows {
		totals.add(rows[i])
	}

	return totals
}

func collect(dbh *sql.DB, databaseFilter *filter.DatabaseFilter) Rows {
	var t Rows

	logger.Printf("collect(?,%q)\n", databaseFilter)

	// we collect all information even if it's mainly empty as we may reference it later
	sql := `SELECT OBJECT_SCHEMA, OBJECT_NAME, COUNT_STAR, SUM_TIMER_WAIT, COUNT_READ, SUM_TIMER_READ, COUNT_WRITE, SUM_TIMER_WRITE, COUNT_FETCH, SUM_TIMER_FETCH, COUNT_INSERT, SUM_TIMER_INSERT, COUNT_UPDATE, SUM_TIMER_UPDATE, COUNT_DELETE, SUM_TIMER_DELETE FROM table_io_waits_summary_by_table WHERE SUM_TIMER_WAIT > 0`
	args := []interface{}{}

	// Apply the filter if provided and seems good.
	if len(databaseFilter.Args()) > 0 {
		sql = sql + databaseFilter.ExtraSQL()
		args = append(args, databaseFilter.Args())
		logger.Printf("apply databaseFilter: sql: %q, args: %+v\n", sql, args)
	}

	rows, err := dbh.Query(sql, args...)
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
			&r.CountStar,
			&r.SumTimerWait,
			&r.CountRead,
			&r.SumTimerRead,
			&r.CountWrite,
			&r.SumTimerWrite,
			&r.CountFetch,
			&r.SumTimerFetch,
			&r.CountInsert,
			&r.SumTimerInsert,
			&r.CountUpdate,
			&r.SumTimerUpdate,
			&r.CountDelete,
			&r.SumTimerDelete); err != nil {
			log.Fatal(err)
		}
		r.Name = lib.TableName(schema, table)

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
	return (rows[i].SumTimerWait > rows[j].SumTimerWait) ||
		((rows[i].SumTimerWait == rows[j].SumTimerWait) &&
			(rows[i].Name < rows[j].Name))
}

// ByOps is used for sorting by the number of operations
type ByOps Rows

func (rows ByOps) Len() int      { return len(rows) }
func (rows ByOps) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }
func (rows ByOps) Less(i, j int) bool {
	return (rows[i].CountStar > rows[j].CountStar) ||
		((rows[i].SumTimerWait == rows[j].SumTimerWait) &&
			(rows[i].Name < rows[j].Name))
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
		initialByName[initial[i].Name] = i
	}

	for i := range *rows {
		rowName := (*rows)[i].Name
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

	return myTotals.SumTimerWait > otherTotals.SumTimerWait
}

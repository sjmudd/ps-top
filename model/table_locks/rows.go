// Package table_locks contains the library
// routines for managing the table_lock_waits_summary_by_table table.
package table_locks

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql" // keep glint happy
	"log"

	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/logger"
	"github.com/sjmudd/ps-top/model/filter"
)

// Rows contains multiple rows
type Rows []Row

// return the total of a slice of rows
func (rows Rows) totals() Row {
	total := Row{Name: "Totals"}

	for _, row := range rows {
		total.add(row)
	}

	return total
}

// Select the raw data from the database into file_summary_by_instance_rows
// - filter out empty values
// - merge rows with the same name into a single row
// - change FILE_NAME into a more descriptive value.
func collect(dbh *sql.DB, databaseFilter *filter.DatabaseFilter) Rows {
	var t Rows

	sql := `
SELECT	OBJECT_SCHEMA,
	OBJECT_NAME,
	SUM_TIMER_WAIT,
	SUM_TIMER_READ,
	SUM_TIMER_WRITE,
	SUM_TIMER_READ_WITH_SHARED_LOCKS,
	SUM_TIMER_READ_HIGH_PRIORITY,
	SUM_TIMER_READ_NO_INSERT,
	SUM_TIMER_READ_NORMAL,
	SUM_TIMER_READ_EXTERNAL,
	SUM_TIMER_WRITE_ALLOW_WRITE,
	SUM_TIMER_WRITE_CONCURRENT_INSERT,
	SUM_TIMER_WRITE_LOW_PRIORITY,
	SUM_TIMER_WRITE_NORMAL,
	SUM_TIMER_WRITE_EXTERNAL
FROM	table_lock_waits_summary_by_table
WHERE	COUNT_STAR > 0`

	args := []interface{}{}

	// Apply the filter if provided and seems good.
	if len(databaseFilter.Args()) > 0 {
		sql = sql + databaseFilter.ExtraSQL()
		for _, v := range databaseFilter.Args() {
			args = append(args, v)
		}
		logger.Printf("apply databaseFilter: sql: %q, args: %+v\n", sql, args)
	}

	rows, err := dbh.Query(sql, args...)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r Row
		var schema, table string

		if err := rows.Scan(
			&schema,
			&table,
			&r.SumTimerWait,
			&r.SumTimerRead,
			&r.SumTimerWrite,
			&r.SumTimerReadWithSharedLocks,
			&r.SumTimerReadHighPriority,
			&r.SumTimerReadNoInsert,
			&r.SumTimerReadNormal,
			&r.SumTimerReadExternal,
			&r.SumTimerWriteAllowWrite,
			&r.SumTimerWriteConcurrentInsert,
			&r.SumTimerWriteLowPriority,
			&r.SumTimerWriteNormal,
			&r.SumTimerWriteExternal); err != nil {
			log.Fatal(err)
		}
		r.Name = lib.TableName(schema, table)
		// we collect all data as we may need it later
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return t
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
func (rows *Rows) subtract(initial Rows) {
	iByName := make(map[string]int)

	// iterate over rows by name
	for i := range initial {
		iByName[initial[i].Name] = i
	}

	for i := range *rows {
		if _, ok := iByName[(*rows)[i].Name]; ok {
			initialI := iByName[(*rows)[i].Name]
			(*rows)[i].subtract(initial[initialI])
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

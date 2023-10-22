// Package tablelocks contains the utilsrary
// routines for managing the table_lock_waits_summary_by_table table.
package tablelocks

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql" // keep glint happy

	"github.com/sjmudd/ps-top/log"
	"github.com/sjmudd/ps-top/model/filter"
	"github.com/sjmudd/ps-top/utils"
)

// Rows contains multiple rows
type Rows []Row

// return the total of a slice of rows
func totals(rows Rows) Row {
	total := Row{Name: "Totals"}

	for _, row := range rows {
		total.SumTimerWait += row.SumTimerWait
		total.SumTimerRead += row.SumTimerRead
		total.SumTimerWrite += row.SumTimerWrite
		total.SumTimerReadWithSharedLocks += row.SumTimerReadWithSharedLocks
		total.SumTimerReadHighPriority += row.SumTimerReadHighPriority
		total.SumTimerReadNoInsert += row.SumTimerReadNoInsert
		total.SumTimerReadNormal += row.SumTimerReadNormal
		total.SumTimerReadExternal += row.SumTimerReadExternal
		total.SumTimerWriteAllowWrite += row.SumTimerWriteAllowWrite
		total.SumTimerWriteConcurrentInsert += row.SumTimerWriteConcurrentInsert
		total.SumTimerWriteLowPriority += row.SumTimerWriteLowPriority
		total.SumTimerWriteNormal += row.SumTimerWriteNormal
		total.SumTimerWriteExternal += row.SumTimerWriteExternal
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
		log.Printf("apply databaseFilter: sql: %q, args: %+v\n", sql, args)
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
		r.Name = utils.QualifiedTableName(schema, table)
		// we collect all data as we may need it later
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return t
}

// remove the initial values from those rows where there's a match
// ignoring rows names that do not match
func (rows *Rows) subtract(initial Rows) {
	initialNameLookup := make(map[string]int)

	// generate lookup map for initial values
	for i, r := range initial {
		initialNameLookup[r.Name] = i
	}

	// subtract initial value for matching rows
	for i, r := range *rows {
		if _, ok := initialNameLookup[r.Name]; ok {
			(*rows)[i].subtract(initial[initialNameLookup[r.Name]])
		}
	}
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (rows Rows) needsRefresh(otherRows Rows) bool {
	return totals(rows).SumTimerWait > totals(otherRows).SumTimerWait
}

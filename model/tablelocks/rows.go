// Package tablelocks contains the utilsrary
// routines for managing the table_lock_waits_summary_by_table table.
package tablelocks

import (
	"database/sql"

	"github.com/sjmudd/ps-top/log"
	"github.com/sjmudd/ps-top/model/filter"
	"github.com/sjmudd/ps-top/utils"

	_ "github.com/go-sql-driver/mysql" // keep glint happy
)

// Rows contains multiple rows
type Rows []Row

// return the total of a slice of rows
func totals(rows []Row) Row {
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
// - change FILE_NAME into a more descriptive value.
func collect(db *sql.DB, filter *filter.DatabaseFilter) []Row {
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
	if len(filter.Args()) > 0 {
		sql += filter.ExtraSQL()
		for _, v := range filter.Args() {
			args = append(args, v)
		}
		log.Printf("apply filter: sql: %q, args: %+v\n", sql, args)
	}

	var rows []Row // to be returned to caller

	sqlrows, err := db.Query(sql, args...)
	if err != nil {
		log.Fatal(err)
	}

	for sqlrows.Next() {
		var row Row
		var schema, table string

		if err := sqlrows.Scan(
			&schema,
			&table,
			&row.SumTimerWait,
			&row.SumTimerRead,
			&row.SumTimerWrite,
			&row.SumTimerReadWithSharedLocks,
			&row.SumTimerReadHighPriority,
			&row.SumTimerReadNoInsert,
			&row.SumTimerReadNormal,
			&row.SumTimerReadExternal,
			&row.SumTimerWriteAllowWrite,
			&row.SumTimerWriteConcurrentInsert,
			&row.SumTimerWriteLowPriority,
			&row.SumTimerWriteNormal,
			&row.SumTimerWriteExternal); err != nil {
			log.Fatal(err)
		}
		row.Name = utils.QualifiedTableName(schema, table)

		rows = append(rows, row)
	}
	if err := sqlrows.Err(); err != nil {
		log.Fatal(err)
	}
	_ = sqlrows.Close()

	return rows
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

// Package table_lock_latency contains the library
// routines for managing the table_lock_waits_summary_by_table table.
package table_lock_latency

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql" // keep glint happy
	"log"
	"sort"
	"strings"

	"github.com/sjmudd/ps-top/lib"
)

// Rows contains multiple rows
type Rows []Row

// return the totals of a slice of rows
func (t Rows) totals() Row {
	var totals Row
	totals.name = "Totals"

	for i := range t {
		totals.add(t[i])
	}

	return totals
}

// Select the raw data from the database into file_summary_by_instance_rows
// - filter out empty values
// - merge rows with the same name into a single row
// - change FILE_NAME into a more descriptive value.
func collect(dbh *sql.DB) Rows {
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

	rows, err := dbh.Query(sql)
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
			&r.sumTimerWait,
			&r.sumTimerRead,
			&r.sumTimerWrite,
			&r.sumTimerReadWithSharedLocks,
			&r.sumTimerReadHighPriority,
			&r.sumTimerReadNoInsert,
			&r.sumTimerReadNormal,
			&r.sumTimerReadExternal,
			&r.sumTimerWriteAllowWrite,
			&r.sumTimerWriteConcurrentInsert,
			&r.sumTimerWriteLowPriority,
			&r.sumTimerWriteNormal,
			&r.sumTimerWriteExternal); err != nil {
			log.Fatal(err)
		}
		r.name = lib.TableName(schema, table)
		// we collect all data as we may need it later
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return t
}

func (t Rows) Len() int      { return len(t) }
func (t Rows) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t Rows) Less(i, j int) bool {
	return (t[i].sumTimerWait > t[j].sumTimerWait) ||
		((t[i].sumTimerWait == t[j].sumTimerWait) &&
			(t[i].name < t[j].name))

}

// sort the data
func (t *Rows) sort() {
	sort.Sort(t)
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
func (t *Rows) subtract(initial Rows) {
	iByName := make(map[string]int)

	// iterate over rows by name
	for i := range initial {
		iByName[initial[i].name] = i
	}

	for i := range *t {
		if _, ok := iByName[(*t)[i].name]; ok {
			initialI := iByName[(*t)[i].name]
			(*t)[i].subtract(initial[initialI])
		}
	}
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (t Rows) needsRefresh(t2 Rows) bool {
	myTotals := t.totals()
	otherTotals := t2.totals()

	return myTotals.sumTimerWait > otherTotals.sumTimerWait
}

// describe a whole table
func (t Rows) String() string {
	s := make([]string, len(t))

	for i := range t {
		s = append(s, t[i].String())
	}

	return strings.Join(s, "\n")
}

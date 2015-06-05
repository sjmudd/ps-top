// Package table_io_latency contains the routines for managing
// performance_schema.table_io_waits_by_table.
package table_io_latency

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/sjmudd/ps-top/lib"
)

// Row contains w from table_io_waits_summary_by_table
type Row struct {
	// Note: upper case names to match the performance_schema column names
	// This type is _not_ exported.

	tableName string // we don't keep the retrieved columns but store the generated table name

	sumTimerWait   uint64
	sumTimerRead   uint64
	sumTimerWrite  uint64
	sumTimerFetch  uint64
	sumTimerInsert uint64
	sumTimerUpdate uint64
	sumTimerDelete uint64

	countStar   uint64
	countRead   uint64
	countWrite  uint64
	countFetch  uint64
	countInsert uint64
	countUpdate uint64
	countDelete uint64
}
// Rows contains a set of rows
type Rows []Row

// name returns the table name
func (row Row) name() string {
	return row.tableName
}

// latencyHeadings returns the latency headings as a string
func (row Row) latencyHeadings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s", "Latency", "%", "Fetch", "Insert", "Update", "Delete", "Table Name")
}

// opsHeadings returns the headings by operations as a string
func (row Row) opsHeadings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s", "Ops", "%", "Fetch", "Insert", "Update", "Delete", "Table Name")
}

// latencyRowContents reutrns the printable result
func (row Row) latencyRowContent(totals Row) string {
	// assume the data is empty so hide it.
	name := row.name()
	if row.countStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s",
		lib.FormatTime(row.sumTimerWait),
		lib.FormatPct(lib.MyDivide(row.sumTimerWait, totals.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(row.sumTimerFetch, row.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(row.sumTimerInsert, row.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(row.sumTimerUpdate, row.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(row.sumTimerDelete, row.sumTimerWait)),
		name)
}

// generate a printable result for ops
func (row Row) opsRowContent(totals Row) string {
	// assume the data is empty so hide it.
	name := row.name()
	if row.countStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s",
		lib.FormatAmount(row.countStar),
		lib.FormatPct(lib.MyDivide(row.countStar, totals.countStar)),
		lib.FormatPct(lib.MyDivide(row.countFetch, row.countStar)),
		lib.FormatPct(lib.MyDivide(row.countInsert, row.countStar)),
		lib.FormatPct(lib.MyDivide(row.countUpdate, row.countStar)),
		lib.FormatPct(lib.MyDivide(row.countDelete, row.countStar)),
		name)
}

func (row *Row) add(other Row) {
	row.sumTimerWait += other.sumTimerWait
	row.sumTimerFetch += other.sumTimerFetch
	row.sumTimerInsert += other.sumTimerInsert
	row.sumTimerUpdate += other.sumTimerUpdate
	row.sumTimerDelete += other.sumTimerDelete
	row.sumTimerRead += other.sumTimerRead
	row.sumTimerWrite += other.sumTimerWrite

	row.countStar += other.countStar
	row.countFetch += other.countFetch
	row.countInsert += other.countInsert
	row.countUpdate += other.countUpdate
	row.countDelete += other.countDelete
	row.countRead += other.countRead
	row.countWrite += other.countWrite
}

// subtract the countable values in one row from another
func (row *Row) subtract(other Row) {
	row.sumTimerWait -= other.sumTimerWait
	row.sumTimerFetch -= other.sumTimerFetch
	row.sumTimerInsert -= other.sumTimerInsert
	row.sumTimerUpdate -= other.sumTimerUpdate
	row.sumTimerDelete -= other.sumTimerDelete
	row.sumTimerRead -= other.sumTimerRead
	row.sumTimerWrite -= other.sumTimerWrite

	row.countStar -= other.countStar
	row.countFetch -= other.countFetch
	row.countInsert -= other.countInsert
	row.countUpdate -= other.countUpdate
	row.countDelete -= other.countDelete
	row.countRead -= other.countRead
	row.countWrite -= other.countWrite
}

func (rows Rows) totals() Row {
	var totals Row
	totals.tableName = "Totals"

	for i := range rows {
		totals.add(rows[i])
	}

	return totals
}

func selectRows(dbh *sql.DB) Rows {
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
		r.tableName = lib.TableName(schema, table)

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
			(rows[i].tableName < rows[j].tableName))
}

// ByOps is used for sorting by the number of operations
type ByOps Rows

func (rows ByOps) Len() int      { return len(rows) }
func (rows ByOps) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }
func (rows ByOps) Less(i, j int) bool {
	return (rows[i].countStar > rows[j].countStar) ||
		((rows[i].sumTimerWait == rows[j].sumTimerWait) &&
			(rows[i].tableName < rows[j].tableName))
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
		initialByName[initial[i].name()] = i
	}

	for i := range *rows {
		rowName := (*rows)[i].name()
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

// describe a whole row
func (row Row) String() string {
	return fmt.Sprintf("%s|%10s %10s %10s %10s %10s|%10s %10s|%10s %10s %10s %10s %10s|%10s %10s",
		row.name(),
		lib.FormatTime(row.sumTimerWait),
		lib.FormatTime(row.sumTimerFetch),
		lib.FormatTime(row.sumTimerInsert),
		lib.FormatTime(row.sumTimerUpdate),
		lib.FormatTime(row.sumTimerDelete),

		lib.FormatTime(row.sumTimerRead),
		lib.FormatTime(row.sumTimerWrite),

		lib.FormatAmount(row.countStar),
		lib.FormatAmount(row.countFetch),
		lib.FormatAmount(row.countInsert),
		lib.FormatAmount(row.countUpdate),
		lib.FormatAmount(row.countDelete),

		lib.FormatAmount(row.countRead),
		lib.FormatAmount(row.countWrite))
}

// describe a whole table
func (rows Rows) String() string {
	s := make([]string, len(rows))

	for i := range rows {
		s = append(s, rows[i].String())
	}

	return strings.Join(s, "\n")
}

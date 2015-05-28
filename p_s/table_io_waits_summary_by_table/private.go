// Package table_io_waits_summary_by_table contains the routines for managing
// performance_schema.table_io_waits_by_table.
package table_io_waits_summary_by_table

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

	SUM_TIMER_WAIT   uint64
	SUM_TIMER_READ   uint64
	SUM_TIMER_WRITE  uint64
	SUM_TIMER_FETCH  uint64
	SUM_TIMER_INSERT uint64
	SUM_TIMER_UPDATE uint64
	SUM_TIMER_DELETE uint64

	COUNT_STAR   uint64
	COUNT_READ   uint64
	COUNT_WRITE  uint64
	COUNT_FETCH  uint64
	COUNT_INSERT uint64
	COUNT_UPDATE uint64
	COUNT_DELETE uint64
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
	if row.COUNT_STAR == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s",
		lib.FormatTime(row.SUM_TIMER_WAIT),
		lib.FormatPct(lib.MyDivide(row.SUM_TIMER_WAIT, totals.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(row.SUM_TIMER_FETCH, row.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(row.SUM_TIMER_INSERT, row.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(row.SUM_TIMER_UPDATE, row.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(row.SUM_TIMER_DELETE, row.SUM_TIMER_WAIT)),
		name)
}

// generate a printable result for ops
func (row Row) opsRowContent(totals Row) string {
	// assume the data is empty so hide it.
	name := row.name()
	if row.COUNT_STAR == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s",
		lib.FormatAmount(row.COUNT_STAR),
		lib.FormatPct(lib.MyDivide(row.COUNT_STAR, totals.COUNT_STAR)),
		lib.FormatPct(lib.MyDivide(row.COUNT_FETCH, row.COUNT_STAR)),
		lib.FormatPct(lib.MyDivide(row.COUNT_INSERT, row.COUNT_STAR)),
		lib.FormatPct(lib.MyDivide(row.COUNT_UPDATE, row.COUNT_STAR)),
		lib.FormatPct(lib.MyDivide(row.COUNT_DELETE, row.COUNT_STAR)),
		name)
}

func (row *Row) add(other Row) {
	row.SUM_TIMER_WAIT += other.SUM_TIMER_WAIT
	row.SUM_TIMER_FETCH += other.SUM_TIMER_FETCH
	row.SUM_TIMER_INSERT += other.SUM_TIMER_INSERT
	row.SUM_TIMER_UPDATE += other.SUM_TIMER_UPDATE
	row.SUM_TIMER_DELETE += other.SUM_TIMER_DELETE
	row.SUM_TIMER_READ += other.SUM_TIMER_READ
	row.SUM_TIMER_WRITE += other.SUM_TIMER_WRITE

	row.COUNT_STAR += other.COUNT_STAR
	row.COUNT_FETCH += other.COUNT_FETCH
	row.COUNT_INSERT += other.COUNT_INSERT
	row.COUNT_UPDATE += other.COUNT_UPDATE
	row.COUNT_DELETE += other.COUNT_DELETE
	row.COUNT_READ += other.COUNT_READ
	row.COUNT_WRITE += other.COUNT_WRITE
}

// subtract the countable values in one row from another
func (row *Row) subtract(other Row) {
	row.SUM_TIMER_WAIT -= other.SUM_TIMER_WAIT
	row.SUM_TIMER_FETCH -= other.SUM_TIMER_FETCH
	row.SUM_TIMER_INSERT -= other.SUM_TIMER_INSERT
	row.SUM_TIMER_UPDATE -= other.SUM_TIMER_UPDATE
	row.SUM_TIMER_DELETE -= other.SUM_TIMER_DELETE
	row.SUM_TIMER_READ -= other.SUM_TIMER_READ
	row.SUM_TIMER_WRITE -= other.SUM_TIMER_WRITE

	row.COUNT_STAR -= other.COUNT_STAR
	row.COUNT_FETCH -= other.COUNT_FETCH
	row.COUNT_INSERT -= other.COUNT_INSERT
	row.COUNT_UPDATE -= other.COUNT_UPDATE
	row.COUNT_DELETE -= other.COUNT_DELETE
	row.COUNT_READ -= other.COUNT_READ
	row.COUNT_WRITE -= other.COUNT_WRITE
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
			&r.COUNT_STAR,
			&r.SUM_TIMER_WAIT,
			&r.COUNT_READ,
			&r.SUM_TIMER_READ,
			&r.COUNT_WRITE,
			&r.SUM_TIMER_WRITE,
			&r.COUNT_FETCH,
			&r.SUM_TIMER_FETCH,
			&r.COUNT_INSERT,
			&r.SUM_TIMER_INSERT,
			&r.COUNT_UPDATE,
			&r.SUM_TIMER_UPDATE,
			&r.COUNT_DELETE,
			&r.SUM_TIMER_DELETE); err != nil {
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
	return (rows[i].SUM_TIMER_WAIT > rows[j].SUM_TIMER_WAIT) ||
		((rows[i].SUM_TIMER_WAIT == rows[j].SUM_TIMER_WAIT) &&
			(rows[i].tableName < rows[j].tableName))
}

// ByOps is used for sorting by the number of operations
type ByOps Rows

func (rows ByOps) Len() int      { return len(rows) }
func (rows ByOps) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }
func (rows ByOps) Less(i, j int) bool {
	return (rows[i].COUNT_STAR > rows[j].COUNT_STAR) ||
		((rows[i].SUM_TIMER_WAIT == rows[j].SUM_TIMER_WAIT) &&
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

	return myTotals.SUM_TIMER_WAIT > otherTotals.SUM_TIMER_WAIT
}

// describe a whole row
func (row Row) String() string {
	return fmt.Sprintf("%s|%10s %10s %10s %10s %10s|%10s %10s|%10s %10s %10s %10s %10s|%10s %10s",
		row.name(),
		lib.FormatTime(row.SUM_TIMER_WAIT),
		lib.FormatTime(row.SUM_TIMER_FETCH),
		lib.FormatTime(row.SUM_TIMER_INSERT),
		lib.FormatTime(row.SUM_TIMER_UPDATE),
		lib.FormatTime(row.SUM_TIMER_DELETE),

		lib.FormatTime(row.SUM_TIMER_READ),
		lib.FormatTime(row.SUM_TIMER_WRITE),

		lib.FormatAmount(row.COUNT_STAR),
		lib.FormatAmount(row.COUNT_FETCH),
		lib.FormatAmount(row.COUNT_INSERT),
		lib.FormatAmount(row.COUNT_UPDATE),
		lib.FormatAmount(row.COUNT_DELETE),

		lib.FormatAmount(row.COUNT_READ),
		lib.FormatAmount(row.COUNT_WRITE))
}

// describe a whole table
func (rows Rows) String() string {
	s := make([]string, len(rows))

	for i := range rows {
		s = append(s, rows[i].String())
	}

	return strings.Join(s, "\n")
}

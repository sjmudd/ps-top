// This file contains the library routines for managing
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

// a row from performance_schema.table_io_waits_summary_by_table
type tableRow struct {
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
type tableRows []tableRow

// name returns the table name
func (row tableRow) name() string {
	return row.tableName
}

// latencyHeadings returns the latency headings as a string
func (row tableRow) latencyHeadings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s", "Latency", "%", "Fetch", "Insert", "Update", "Delete", "Table Name")
}

// opsHeadings returns the headings by operations as a string
func (row tableRow) opsHeadings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s", "Ops", "%", "Fetch", "Insert", "Update", "Delete", "Table Name")
}

// latencyRowContents reutrns the printable result
func (row tableRow) latencyRowContent(totals tableRow) string {
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
func (row tableRow) opsRowContent(totals tableRow) string {
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

func (row *tableRow) add(other tableRow) {
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
func (row *tableRow) subtract(other tableRow) {
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

func (rows tableRows) totals() tableRow {
	var totals tableRow
	totals.tableName = "Totals"

	for i := range rows {
		totals.add(rows[i])
	}

	return totals
}

func selectRows(dbh *sql.DB) tableRows {
	var t tableRows

	// we collect all information even if it's mainly empty as we may reference it later
	sql := "SELECT OBJECT_SCHEMA, OBJECT_NAME, COUNT_STAR, SUM_TIMER_WAIT, COUNT_READ, SUM_TIMER_READ, COUNT_WRITE, SUM_TIMER_WRITE, COUNT_FETCH, SUM_TIMER_FETCH, COUNT_INSERT, SUM_TIMER_INSERT, COUNT_UPDATE, SUM_TIMER_UPDATE, COUNT_DELETE, SUM_TIMER_DELETE FROM table_io_waits_summary_by_table WHERE SUM_TIMER_WAIT > 0"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table string
		var r tableRow
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

func (rows tableRows) Len() int      { return len(rows) }
func (rows tableRows) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }

// sort by value (descending) but also by "name" (ascending) if the values are the same
func (rows tableRows) Less(i, j int) bool {
	return (rows[i].SUM_TIMER_WAIT > rows[j].SUM_TIMER_WAIT) ||
		((rows[i].SUM_TIMER_WAIT == rows[j].SUM_TIMER_WAIT) &&
			(rows[i].tableName < rows[j].tableName))
}

// for sorting
type ByOps tableRows

func (rows ByOps) Len() int      { return len(rows) }
func (rows ByOps) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }
func (rows ByOps) Less(i, j int) bool {
	return (rows[i].COUNT_STAR > rows[j].COUNT_STAR) ||
		((rows[i].SUM_TIMER_WAIT == rows[j].SUM_TIMER_WAIT) &&
			(rows[i].tableName < rows[j].tableName))
}

func (rows tableRows) Sort(wantLatency bool) {
	if wantLatency {
		sort.Sort(rows)
	} else {
		sort.Sort(ByOps(rows))
	}
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
func (rows *tableRows) subtract(initial tableRows) {
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
func (t tableRows) needsRefresh(t2 tableRows) bool {
	myTotals := t.totals()
	otherTotals := t2.totals()

	return myTotals.SUM_TIMER_WAIT > otherTotals.SUM_TIMER_WAIT
}

// describe a whole row
func (r tableRow) String() string {
	return fmt.Sprintf("%s|%10s %10s %10s %10s %10s|%10s %10s|%10s %10s %10s %10s %10s|%10s %10s",
		r.name(),
		lib.FormatTime(r.SUM_TIMER_WAIT),
		lib.FormatTime(r.SUM_TIMER_FETCH),
		lib.FormatTime(r.SUM_TIMER_INSERT),
		lib.FormatTime(r.SUM_TIMER_UPDATE),
		lib.FormatTime(r.SUM_TIMER_DELETE),

		lib.FormatTime(r.SUM_TIMER_READ),
		lib.FormatTime(r.SUM_TIMER_WRITE),

		lib.FormatAmount(r.COUNT_STAR),
		lib.FormatAmount(r.COUNT_FETCH),
		lib.FormatAmount(r.COUNT_INSERT),
		lib.FormatAmount(r.COUNT_UPDATE),
		lib.FormatAmount(r.COUNT_DELETE),

		lib.FormatAmount(r.COUNT_READ),
		lib.FormatAmount(r.COUNT_WRITE))
}

// describe a whole table
func (t tableRows) String() string {
	s := make([]string, len(t))

	for i := range t {
		s = append(s, t[i].String())
	}

	return strings.Join(s, "\n")
}

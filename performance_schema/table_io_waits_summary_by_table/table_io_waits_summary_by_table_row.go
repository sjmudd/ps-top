// This file contains the library routines for managing the
// table_io_waits_by_table table.
package table_io_waits_summary_by_table

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/sjmudd/pstop/lib"
)

// a row from performance_schema.table_io_waits_summary_by_table
type table_io_waits_summary_by_table_row struct {
	// Note: upper case names to match the performance_schema column names
	// This type is _not_ exported.

	OBJECT_TYPE   string // in theory redundant but keep anyway
	OBJECT_SCHEMA string // in theory redundant but keep anyway
	OBJECT_NAME   string // in theory redundant but keep anyway

	SUM_TIMER_WAIT   int
	SUM_TIMER_READ   int
	SUM_TIMER_WRITE  int
	SUM_TIMER_FETCH  int
	SUM_TIMER_INSERT int
	SUM_TIMER_UPDATE int
	SUM_TIMER_DELETE int

	COUNT_STAR   int
	COUNT_READ   int
	COUNT_WRITE  int
	COUNT_FETCH  int
	COUNT_INSERT int
	COUNT_UPDATE int
	COUNT_DELETE int
}
type table_io_waits_summary_by_table_rows []table_io_waits_summary_by_table_row

// // return the table name from the columns as '<schema>.<table>'
func (r *table_io_waits_summary_by_table_row) name() string {
	var n string
	if len(r.OBJECT_SCHEMA) > 0 {
		n += r.OBJECT_SCHEMA
	}
	if len(n) > 0 {
		if len(r.OBJECT_NAME) > 0 {
			n += "." + r.OBJECT_NAME
		}
	} else {
		if len(r.OBJECT_NAME) > 0 {
			n += r.OBJECT_NAME
		}
	}
	return n
}

func (r *table_io_waits_summary_by_table_row) pretty_name() string {
	s := r.name()
	if len(s) > 30 {
		s = s[:29]
	}
	return fmt.Sprintf("%-30s", s)
}

func (r *table_io_waits_summary_by_table_row) latency_headings() string {
	return fmt.Sprintf("%-30s %10s %6s|%6s %6s %6s %6s", "Table Name", "Latency", "%", "Fetch", "Insert", "Update", "Delete")
}
func (r *table_io_waits_summary_by_table_row) ops_headings() string {
	return fmt.Sprintf("%-30s %10s %6s|%6s %6s %6s %6s", "Table Name", "Ops", "%", "Fetch", "Insert", "Update", "Delete")
}

// generate a printable result
func (r *table_io_waits_summary_by_table_row) latency_row_content(totals table_io_waits_summary_by_table_row) string {
	// assume the data is empty so hide it.
	name := r.pretty_name()
	if r.COUNT_STAR == 0 {
		name = ""
	}

	return fmt.Sprintf("%-30s %10s %6s|%6s %6s %6s %6s",
		name,
		lib.FormatTime(r.SUM_TIMER_WAIT),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_WAIT, totals.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_FETCH, r.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_INSERT, r.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_UPDATE, r.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_DELETE, r.SUM_TIMER_WAIT)))
}

// generate a printable result for ops
func (r *table_io_waits_summary_by_table_row) ops_row_content(totals table_io_waits_summary_by_table_row) string {
	// assume the data is empty so hide it.
	name := r.pretty_name()
	if r.COUNT_STAR == 0 {
		name = ""
	}

	return fmt.Sprintf("%-30s %10s %6s|%6s %6s %6s %6s",
		name,
		lib.FormatAmount(r.COUNT_STAR),
		lib.FormatPct(lib.MyDivide(r.COUNT_STAR, totals.COUNT_STAR)),
		lib.FormatPct(lib.MyDivide(r.COUNT_FETCH, r.COUNT_STAR)),
		lib.FormatPct(lib.MyDivide(r.COUNT_INSERT, r.COUNT_STAR)),
		lib.FormatPct(lib.MyDivide(r.COUNT_UPDATE, r.COUNT_STAR)),
		lib.FormatPct(lib.MyDivide(r.COUNT_DELETE, r.COUNT_STAR)))
}

func (this *table_io_waits_summary_by_table_row) add(other table_io_waits_summary_by_table_row) {
	this.SUM_TIMER_WAIT += other.SUM_TIMER_WAIT
	this.SUM_TIMER_FETCH += other.SUM_TIMER_FETCH
	this.SUM_TIMER_INSERT += other.SUM_TIMER_INSERT
	this.SUM_TIMER_UPDATE += other.SUM_TIMER_UPDATE
	this.SUM_TIMER_DELETE += other.SUM_TIMER_DELETE
	this.SUM_TIMER_READ += other.SUM_TIMER_READ
	this.SUM_TIMER_WRITE += other.SUM_TIMER_WRITE

	this.COUNT_STAR += other.COUNT_STAR
	this.COUNT_FETCH += other.COUNT_FETCH
	this.COUNT_INSERT += other.COUNT_INSERT
	this.COUNT_UPDATE += other.COUNT_UPDATE
	this.COUNT_DELETE += other.COUNT_DELETE
	this.COUNT_READ += other.COUNT_READ
	this.COUNT_WRITE += other.COUNT_WRITE
}

func (this *table_io_waits_summary_by_table_row) subtract(other table_io_waits_summary_by_table_row) {
	this.SUM_TIMER_WAIT -= other.SUM_TIMER_WAIT
	this.SUM_TIMER_FETCH -= other.SUM_TIMER_FETCH
	this.SUM_TIMER_INSERT -= other.SUM_TIMER_INSERT
	this.SUM_TIMER_UPDATE -= other.SUM_TIMER_UPDATE
	this.SUM_TIMER_DELETE -= other.SUM_TIMER_DELETE
	this.SUM_TIMER_READ -= other.SUM_TIMER_READ
	this.SUM_TIMER_WRITE -= other.SUM_TIMER_WRITE

	this.COUNT_STAR -= other.COUNT_STAR
	this.COUNT_FETCH -= other.COUNT_FETCH
	this.COUNT_INSERT -= other.COUNT_INSERT
	this.COUNT_UPDATE -= other.COUNT_UPDATE
	this.COUNT_DELETE -= other.COUNT_DELETE
	this.COUNT_READ -= other.COUNT_READ
	this.COUNT_WRITE -= other.COUNT_WRITE
}

func (t table_io_waits_summary_by_table_rows) totals() table_io_waits_summary_by_table_row {
	var totals table_io_waits_summary_by_table_row
	totals.OBJECT_SCHEMA = "TOTALS"

	for i := range t {
		totals.add(t[i])
	}

	return totals
}

func select_tiwsbt_rows(dbh *sql.DB) table_io_waits_summary_by_table_rows {
	var t table_io_waits_summary_by_table_rows

	// we collect all information even if it's mainly empty as we may reference it later
	sql := "SELECT OBJECT_TYPE, OBJECT_SCHEMA, OBJECT_NAME, COUNT_STAR, SUM_TIMER_WAIT, COUNT_READ, SUM_TIMER_READ, COUNT_WRITE, SUM_TIMER_WRITE, COUNT_FETCH, SUM_TIMER_FETCH, COUNT_INSERT, SUM_TIMER_INSERT, COUNT_UPDATE, SUM_TIMER_UPDATE, COUNT_DELETE, SUM_TIMER_DELETE FROM table_io_waits_summary_by_table WHERE SUM_TIMER_WAIT > 0"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r table_io_waits_summary_by_table_row
		if err := rows.Scan(
			&r.OBJECT_TYPE,
			&r.OBJECT_SCHEMA,
			&r.OBJECT_NAME,
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
		// we collect all information even if it's mainly empty as we may reference it later
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return t
}

func (t table_io_waits_summary_by_table_rows) Len() int      { return len(t) }
func (t table_io_waits_summary_by_table_rows) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

// sort by value (descending) but also by "name" (ascending) if the values are the same
func (t table_io_waits_summary_by_table_rows) Less(i, j int) bool {
	return (t[i].SUM_TIMER_WAIT > t[j].SUM_TIMER_WAIT) ||
		((t[i].SUM_TIMER_WAIT == t[j].SUM_TIMER_WAIT) &&
			(t[i].OBJECT_SCHEMA < t[j].OBJECT_SCHEMA) &&
			(t[i].OBJECT_NAME < t[j].OBJECT_NAME))
}

// for sorting
type ByOps table_io_waits_summary_by_table_rows

func (t ByOps) Len() int      { return len(t) }
func (t ByOps) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t ByOps) Less(i, j int) bool {
	return (t[i].COUNT_STAR > t[j].COUNT_STAR) ||
		((t[i].SUM_TIMER_WAIT == t[j].SUM_TIMER_WAIT) &&
			(t[i].OBJECT_SCHEMA < t[j].OBJECT_SCHEMA) &&
			(t[i].OBJECT_NAME < t[j].OBJECT_NAME))
}

func (t table_io_waits_summary_by_table_rows) Sort(want_latency bool) {
	if want_latency {
		sort.Sort(t)
	} else {
		sort.Sort(ByOps(t))
	}
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
func (this *table_io_waits_summary_by_table_rows) subtract(initial table_io_waits_summary_by_table_rows) {
	i_by_name := make(map[string]int)

	// iterate over rows by name
	for i := range initial {
		i_by_name[initial[i].name()] = i
	}

	for i := range *this {
		if _, ok := i_by_name[(*this)[i].name()]; ok {
			initial_i := i_by_name[(*this)[i].name()]
			(*this)[i].subtract(initial[initial_i])
		}
	}
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (t table_io_waits_summary_by_table_rows) needs_refresh(t2 table_io_waits_summary_by_table_rows) bool {
	my_totals := t.totals()
	t2_totals := t2.totals()

	return my_totals.SUM_TIMER_WAIT > t2_totals.SUM_TIMER_WAIT
}

// describe a whole row
func (r table_io_waits_summary_by_table_row) String() string {
	return fmt.Sprintf("%-30s|%10s %10s %10s %10s %10s|%10s %10s|%10s %10s %10s %10s %10s|%10s %10s",
		r.pretty_name(),
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
func (t table_io_waits_summary_by_table_rows) String() string {
	s := make([]string, len(t))

	for i := range t {
		s = append(s, t[i].String())
	}

	return strings.Join(s, "\n")
}

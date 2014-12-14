// This file contains the library routines for managing the
// table_lock_waits_summary_by_table table.
package table_lock_waits_summary_by_table

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"sort"
	"strings"

	"github.com/sjmudd/pstop/lib"
)

/*

From 5.7.5

*************************** 1. row ***************************
       Table: table_lock_waits_summary_by_table
Create Table: CREATE TABLE `table_lock_waits_summary_by_table` (
  `OBJECT_TYPE` varchar(64) DEFAULT NULL,
  `OBJECT_SCHEMA` varchar(64) DEFAULT NULL,
  `OBJECT_NAME` varchar(64) DEFAULT NULL,
  `COUNT_STAR` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `COUNT_READ` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ` bigint(20) unsigned NOT NULL,
  `COUNT_WRITE` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `COUNT_READ_NORMAL` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ_NORMAL` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ_NORMAL` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ_NORMAL` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ_NORMAL` bigint(20) unsigned NOT NULL,
  `COUNT_READ_WITH_SHARED_LOCKS` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ_WITH_SHARED_LOCKS` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ_WITH_SHARED_LOCKS` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ_WITH_SHARED_LOCKS` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ_WITH_SHARED_LOCKS` bigint(20) unsigned NOT NULL,
  `COUNT_READ_HIGH_PRIORITY` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ_HIGH_PRIORITY` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ_HIGH_PRIORITY` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ_HIGH_PRIORITY` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ_HIGH_PRIORITY` bigint(20) unsigned NOT NULL,
  `COUNT_READ_NO_INSERT` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ_NO_INSERT` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ_NO_INSERT` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ_NO_INSERT` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ_NO_INSERT` bigint(20) unsigned NOT NULL,
  `COUNT_READ_EXTERNAL` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ_EXTERNAL` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ_EXTERNAL` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ_EXTERNAL` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ_EXTERNAL` bigint(20) unsigned NOT NULL,
  `COUNT_WRITE_ALLOW_WRITE` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE_ALLOW_WRITE` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE_ALLOW_WRITE` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE_ALLOW_WRITE` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE_ALLOW_WRITE` bigint(20) unsigned NOT NULL,
  `COUNT_WRITE_CONCURRENT_INSERT` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE_CONCURRENT_INSERT` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE_CONCURRENT_INSERT` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE_CONCURRENT_INSERT` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE_CONCURRENT_INSERT` bigint(20) unsigned NOT NULL,
  `COUNT_WRITE_LOW_PRIORITY` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE_LOW_PRIORITY` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE_LOW_PRIORITY` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE_LOW_PRIORITY` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE_LOW_PRIORITY` bigint(20) unsigned NOT NULL,
  `COUNT_WRITE_NORMAL` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE_NORMAL` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE_NORMAL` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE_NORMAL` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE_NORMAL` bigint(20) unsigned NOT NULL,
  `COUNT_WRITE_EXTERNAL` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE_EXTERNAL` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE_EXTERNAL` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE_EXTERNAL` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE_EXTERNAL` bigint(20) unsigned NOT NULL
) ENGINE=PERFORMANCE_SCHEMA DEFAULT CHARSET=utf8

*/

type table_lock_waits_summary_by_table_row struct {
	OBJECT_TYPE   string // in theory redundant but keep anyway
	OBJECT_SCHEMA string // in theory redundant but keep anyway
	OBJECT_NAME   string // in theory redundant but keep anyway
	COUNT_STAR    int

	SUM_TIMER_WAIT  uint64
	SUM_TIMER_READ  uint64
	SUM_TIMER_WRITE uint64

	SUM_TIMER_READ_WITH_SHARED_LOCKS uint64
	SUM_TIMER_READ_HIGH_PRIORITY     uint64
	SUM_TIMER_READ_NO_INSERT         uint64
	SUM_TIMER_READ_NORMAL            uint64
	SUM_TIMER_READ_EXTERNAL          uint64

	SUM_TIMER_WRITE_ALLOW_WRITE       uint64
	SUM_TIMER_WRITE_CONCURRENT_INSERT uint64
	SUM_TIMER_WRITE_LOW_PRIORITY      uint64
	SUM_TIMER_WRITE_NORMAL            uint64
	SUM_TIMER_WRITE_EXTERNAL          uint64
}

type table_lock_waits_summary_by_table_rows []table_lock_waits_summary_by_table_row

// return the table name from the columns as '<schema>.<table>'
func (r *table_lock_waits_summary_by_table_row) name() string {
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

func (r *table_lock_waits_summary_by_table_row) pretty_name() string {
	s := r.name()
	if len(s) > 30 {
		s = s[:29]
	}
	return s
}

// Table Name                        Latency      %|  Read  Write|S.Lock   High  NoIns Normal Extrnl|AlloWr CncIns WrtDly    Low Normal Extrnl|
// xxxxxxxxxxxxxxxxxxxxxxxxxxxxx  1234567890 100.0%|xxxxx% xxxxx%|xxxxx% xxxxx% xxxxx% xxxxx% xxxxx%|xxxxx% xxxxx% xxxxx% xxxxx% xxxxx% xxxxx%|
func (r *table_lock_waits_summary_by_table_row) headings() string {
	return fmt.Sprintf("%-30s %10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s",
		"Table Name", "Latency", "%",
		"Read", "Write",
		"S.Lock", "High", "NoIns", "Normal", "Extrnl",
		"AlloWr", "CncIns", "Low", "Normal", "Extrnl")
}

// generate a printable result
func (r *table_lock_waits_summary_by_table_row) row_content(totals table_lock_waits_summary_by_table_row) string {

	// assume the data is empty so hide it.
	name := r.pretty_name()
	if r.COUNT_STAR == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%-30s %10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s",
		name,
		lib.FormatTime(r.SUM_TIMER_WAIT),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_WAIT, totals.SUM_TIMER_WAIT)),

		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_READ, r.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_WRITE, r.SUM_TIMER_WAIT)),

		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_READ_WITH_SHARED_LOCKS, r.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_READ_HIGH_PRIORITY, r.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_READ_NO_INSERT, r.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_READ_NORMAL, r.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_READ_EXTERNAL, r.SUM_TIMER_WAIT)),

		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_WRITE_ALLOW_WRITE, r.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_WRITE_CONCURRENT_INSERT, r.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_WRITE_LOW_PRIORITY, r.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_WRITE_NORMAL, r.SUM_TIMER_WAIT)),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_WRITE_EXTERNAL, r.SUM_TIMER_WAIT)))
}

func (this *table_lock_waits_summary_by_table_row) add(other table_lock_waits_summary_by_table_row) {
	this.COUNT_STAR += other.COUNT_STAR
	this.SUM_TIMER_WAIT += other.SUM_TIMER_WAIT
	this.SUM_TIMER_READ += other.SUM_TIMER_READ
	this.SUM_TIMER_WRITE += other.SUM_TIMER_WRITE
	this.SUM_TIMER_READ_WITH_SHARED_LOCKS += other.SUM_TIMER_READ_WITH_SHARED_LOCKS
	this.SUM_TIMER_READ_HIGH_PRIORITY += other.SUM_TIMER_READ_HIGH_PRIORITY
	this.SUM_TIMER_READ_NO_INSERT += other.SUM_TIMER_READ_NO_INSERT
	this.SUM_TIMER_READ_NORMAL += other.SUM_TIMER_READ_NORMAL
	this.SUM_TIMER_READ_EXTERNAL += other.SUM_TIMER_READ_EXTERNAL
	this.SUM_TIMER_WRITE_CONCURRENT_INSERT += other.SUM_TIMER_WRITE_CONCURRENT_INSERT
	this.SUM_TIMER_WRITE_LOW_PRIORITY += other.SUM_TIMER_WRITE_LOW_PRIORITY
	this.SUM_TIMER_WRITE_NORMAL += other.SUM_TIMER_WRITE_NORMAL
	this.SUM_TIMER_WRITE_EXTERNAL += other.SUM_TIMER_WRITE_EXTERNAL
}

func (this *table_lock_waits_summary_by_table_row) subtract(other table_lock_waits_summary_by_table_row) {
	this.COUNT_STAR -= other.COUNT_STAR
	this.SUM_TIMER_WAIT -= other.SUM_TIMER_WAIT
	this.SUM_TIMER_READ -= other.SUM_TIMER_READ
	this.SUM_TIMER_WRITE -= other.SUM_TIMER_WRITE
	this.SUM_TIMER_READ_WITH_SHARED_LOCKS -= other.SUM_TIMER_READ_WITH_SHARED_LOCKS
	this.SUM_TIMER_READ_HIGH_PRIORITY -= other.SUM_TIMER_READ_HIGH_PRIORITY
	this.SUM_TIMER_READ_NO_INSERT -= other.SUM_TIMER_READ_NO_INSERT
	this.SUM_TIMER_READ_NORMAL -= other.SUM_TIMER_READ_NORMAL
	this.SUM_TIMER_READ_EXTERNAL -= other.SUM_TIMER_READ_EXTERNAL
	this.SUM_TIMER_WRITE_CONCURRENT_INSERT -= other.SUM_TIMER_WRITE_CONCURRENT_INSERT
	this.SUM_TIMER_WRITE_LOW_PRIORITY -= other.SUM_TIMER_WRITE_LOW_PRIORITY
	this.SUM_TIMER_WRITE_NORMAL -= other.SUM_TIMER_WRITE_NORMAL
	this.SUM_TIMER_WRITE_EXTERNAL -= other.SUM_TIMER_WRITE_EXTERNAL
}

// return the totals of a slice of rows
func (t table_lock_waits_summary_by_table_rows) totals() table_lock_waits_summary_by_table_row {
	var totals table_lock_waits_summary_by_table_row
	totals.OBJECT_SCHEMA = "Totals"

	for i := range t {
		totals.add(t[i])
	}

	return totals
}

// Select the raw data from the database into file_summary_by_instance_rows
// - filter out empty values
// - merge rows with the same name into a single row
// - change FILE_NAME into a more descriptive value.
func select_tlwsbt_rows(dbh *sql.DB) table_lock_waits_summary_by_table_rows {
	var t table_lock_waits_summary_by_table_rows

	sql := "SELECT OBJECT_TYPE, OBJECT_SCHEMA, OBJECT_NAME, COUNT_STAR, SUM_TIMER_WAIT, SUM_TIMER_READ, SUM_TIMER_WRITE, SUM_TIMER_READ_WITH_SHARED_LOCKS, SUM_TIMER_READ_HIGH_PRIORITY, SUM_TIMER_READ_NO_INSERT, SUM_TIMER_READ_NORMAL, SUM_TIMER_READ_EXTERNAL, SUM_TIMER_WRITE_ALLOW_WRITE, SUM_TIMER_WRITE_CONCURRENT_INSERT, SUM_TIMER_WRITE_LOW_PRIORITY, SUM_TIMER_WRITE_NORMAL, SUM_TIMER_WRITE_EXTERNAL FROM table_lock_waits_summary_by_table WHERE COUNT_STAR > 0"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r table_lock_waits_summary_by_table_row
		if err := rows.Scan(
			&r.OBJECT_TYPE,
			&r.OBJECT_SCHEMA,
			&r.OBJECT_NAME,
			&r.COUNT_STAR,
			&r.SUM_TIMER_WAIT,
			&r.SUM_TIMER_READ,
			&r.SUM_TIMER_WRITE,
			&r.SUM_TIMER_READ_WITH_SHARED_LOCKS,
			&r.SUM_TIMER_READ_HIGH_PRIORITY,
			&r.SUM_TIMER_READ_NO_INSERT,
			&r.SUM_TIMER_READ_NORMAL,
			&r.SUM_TIMER_READ_EXTERNAL,
			&r.SUM_TIMER_WRITE_ALLOW_WRITE,
			&r.SUM_TIMER_WRITE_CONCURRENT_INSERT,
			&r.SUM_TIMER_WRITE_LOW_PRIORITY,
			&r.SUM_TIMER_WRITE_NORMAL,
			&r.SUM_TIMER_WRITE_EXTERNAL); err != nil {
			log.Fatal(err)
		}
		// we collect all data as we may need it later
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return t
}

func (t table_lock_waits_summary_by_table_rows) Len() int      { return len(t) }
func (t table_lock_waits_summary_by_table_rows) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t table_lock_waits_summary_by_table_rows) Less(i, j int) bool {
	return (t[i].SUM_TIMER_WAIT > t[j].SUM_TIMER_WAIT) ||
		((t[i].SUM_TIMER_WAIT == t[j].SUM_TIMER_WAIT) &&
			(t[i].OBJECT_SCHEMA < t[j].OBJECT_SCHEMA) &&
			(t[i].OBJECT_NAME < t[j].OBJECT_NAME))

}

// sort the data
func (t *table_lock_waits_summary_by_table_rows) sort() {
	sort.Sort(t)
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
func (this *table_lock_waits_summary_by_table_rows) subtract(initial table_lock_waits_summary_by_table_rows) {
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
func (t table_lock_waits_summary_by_table_rows) needs_refresh(t2 table_lock_waits_summary_by_table_rows) bool {
	my_totals := t.totals()
	t2_totals := t2.totals()

	return my_totals.SUM_TIMER_WAIT > t2_totals.SUM_TIMER_WAIT
}

// describe a whole row
func (r table_lock_waits_summary_by_table_row) String() string {
	return fmt.Sprintf("%-30s|%10s %10s %10s|%10s %10s %10s %10s %10s|%10s %10s %10s %10s %10s",
		r.pretty_name(),
		lib.FormatTime(r.SUM_TIMER_WAIT),
		lib.FormatTime(r.SUM_TIMER_READ),
		lib.FormatTime(r.SUM_TIMER_WRITE),

		lib.FormatTime(r.SUM_TIMER_READ_WITH_SHARED_LOCKS),
		lib.FormatTime(r.SUM_TIMER_READ_HIGH_PRIORITY),
		lib.FormatTime(r.SUM_TIMER_READ_NO_INSERT),
		lib.FormatTime(r.SUM_TIMER_READ_NORMAL),
		lib.FormatTime(r.SUM_TIMER_READ_EXTERNAL),

		lib.FormatTime(r.SUM_TIMER_WRITE_ALLOW_WRITE),
		lib.FormatTime(r.SUM_TIMER_WRITE_CONCURRENT_INSERT),
		lib.FormatTime(r.SUM_TIMER_WRITE_LOW_PRIORITY),
		lib.FormatTime(r.SUM_TIMER_WRITE_NORMAL),
		lib.FormatTime(r.SUM_TIMER_WRITE_EXTERNAL))
}

// describe a whole table
func (t table_lock_waits_summary_by_table_rows) String() string {
	s := make([]string, len(t))

	for i := range t {
		s = append(s, t[i].String())
	}

	return strings.Join(s, "\n")
}

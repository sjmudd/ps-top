// This file contains the library routines for managing the
// table_io_waits_by_table table.
package events_waits_summary_global_by_event_name

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/sjmudd/ps-top/lib"
)

// a row from performance_schema.events_waits_summary_global_by_event_name
// Note: upper case names to match the performance_schema column names.
// This type is _not_ meant to be exported.
type table_row struct {
	EVENT_NAME     string
	SUM_TIMER_WAIT uint64
	COUNT_STAR     uint64
}
type table_rows []table_row

// trim off the leading 'wait/synch/mutex/innodb/'
func (r *table_row) name() string {
	var n string
	if r.EVENT_NAME == "Totals" {
		n += r.EVENT_NAME
	} else if len(r.EVENT_NAME) >= 24 {
		n += r.EVENT_NAME[24:]
	}
	return n
}

func (r *table_row) headings() string {
	return fmt.Sprintf("%10s %6s %6s %s", "Latency", "MtxCnt", "%", "Mutex Name")
}

// generate a printable result
func (r *table_row) row_content(totals table_row) string {
	name := r.name()
	if r.COUNT_STAR == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s %6s|%s",
		lib.FormatTime(r.SUM_TIMER_WAIT),
		lib.FormatAmount(r.COUNT_STAR),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_WAIT, totals.SUM_TIMER_WAIT)),
		name)
}

func (this *table_row) add(other table_row) {
	this.SUM_TIMER_WAIT += other.SUM_TIMER_WAIT
	this.COUNT_STAR += other.COUNT_STAR
}

// subtract the countable values in one row from another
func (this *table_row) subtract(other table_row) {
	// check for issues here (we have a bug) and log it
	// - this situation should not happen so there's a logic bug somewhere else
	if this.SUM_TIMER_WAIT >= other.SUM_TIMER_WAIT {
		this.SUM_TIMER_WAIT -= other.SUM_TIMER_WAIT
		this.COUNT_STAR -= other.COUNT_STAR
	} else {
		lib.Logger.Println("WARNING: table_row.subtract() - subtraction problem! (not subtracting)")
		lib.Logger.Println("this=", this)
		lib.Logger.Println("other=", other)
	}
}

func (t table_rows) totals() table_row {
	var totals table_row
	totals.EVENT_NAME = "Totals"

	for i := range t {
		totals.add(t[i])
	}

	return totals
}

func select_rows(dbh *sql.DB) table_rows {
	var t table_rows

	// we collect all information even if it's mainly empty as we may reference it later
	sql := "SELECT EVENT_NAME, SUM_TIMER_WAIT, COUNT_STAR FROM events_waits_summary_global_by_event_name WHERE SUM_TIMER_WAIT > 0 AND EVENT_NAME LIKE 'wait/synch/mutex/innodb/%'"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r table_row
		if err := rows.Scan(
			&r.EVENT_NAME,
			&r.SUM_TIMER_WAIT,
			&r.COUNT_STAR); err != nil {
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

func (t table_rows) Len() int      { return len(t) }
func (t table_rows) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

// sort by value (descending) but also by "name" (ascending) if the values are the same
func (t table_rows) Less(i, j int) bool {
	return t[i].SUM_TIMER_WAIT > t[j].SUM_TIMER_WAIT
}

func (t table_rows) Sort() {
	sort.Sort(t)
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
func (this *table_rows) subtract(initial table_rows) {
	initial_by_name := make(map[string]int)

	// iterate over rows by name
	for i := range initial {
		initial_by_name[initial[i].name()] = i
	}

	for i := range *this {
		this_name := (*this)[i].name()
		if _, ok := initial_by_name[this_name]; ok {
			initial_index := initial_by_name[this_name]
			(*this)[i].subtract(initial[initial_index])
		}
	}
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (t table_rows) needs_refresh(t2 table_rows) bool {
	my_totals := t.totals()
	t2_totals := t2.totals()

	return my_totals.SUM_TIMER_WAIT > t2_totals.SUM_TIMER_WAIT
}

// describe a whole row
func (r table_row) String() string {
	return fmt.Sprintf("%10s %10s %10s %10s %10s|%10s %10s|%10s %10s %10s %10s %10s|%10s %10s|%s",
		lib.FormatTime(r.SUM_TIMER_WAIT),
		lib.FormatAmount(r.COUNT_STAR),
		r.name())
}

// describe a whole table
func (t table_rows) String() string {
	s := make([]string, len(t))

	for i := range t {
		s = append(s, t[i].String())
	}

	return strings.Join(s, "\n")
}

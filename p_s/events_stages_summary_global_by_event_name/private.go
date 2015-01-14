package events_stages_summary_global_by_event_name

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/sjmudd/pstop/lib"
)

/**************************************************************************

root@localhost [performance_schema]> show create table  events_stages_summary_global_by_event_name\G
*************************** 1. row ***************************
       Table: events_stages_summary_global_by_event_name
Create Table: CREATE TABLE `events_stages_summary_global_by_event_name` (
  `EVENT_NAME` varchar(128) NOT NULL,
  `COUNT_STAR` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WAIT` bigint(20) unsigned NOT NULL, // not used
  `AVG_TIMER_WAIT` bigint(20) unsigned NOT NULL, // not used
  `MAX_TIMER_WAIT` bigint(20) unsigned NOT NULL  // not used
) ENGINE=PERFORMANCE_SCHEMA DEFAULT CHARSET=utf8
1 row in set (0.00 sec)

**************************************************************************/

// one row of data
type table_row struct {
	EVENT_NAME     string
	COUNT_STAR     uint64
	SUM_TIMER_WAIT uint64
}

// a table of rows
type table_rows []table_row

// select the rows into table
func select_rows(dbh *sql.DB) table_rows {
	var t table_rows

	lib.Logger.Println("events_stages_summary_global_by_event_name.select_rows()")
	sql := "SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT FROM events_stages_summary_global_by_event_name WHERE SUM_TIMER_WAIT > 0"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r table_row
		if err := rows.Scan(
			&r.EVENT_NAME,
			&r.COUNT_STAR,
			&r.SUM_TIMER_WAIT); err != nil {
			log.Fatal(err)
		}
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	lib.Logger.Println("recovered", len(t), "row(s):")
	lib.Logger.Println(t)

	return t
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (t table_rows) needs_refresh(t2 table_rows) bool {
	my_totals := t.totals()
	t2_totals := t2.totals()

	return my_totals.SUM_TIMER_WAIT > t2_totals.SUM_TIMER_WAIT
}

// generate the totals of a table
func (t table_rows) totals() table_row {
	var totals table_row
	totals.EVENT_NAME = "Totals"

	for i := range t {
		totals.add(t[i])
	}

	return totals
}

// return the stage name, removing any leading stage/sql/
func (r *table_row) name() string {
	if len(r.EVENT_NAME) > 10 && r.EVENT_NAME[0:10] == "stage/sql/" {
		return r.EVENT_NAME[10:]
	} else {
		return r.EVENT_NAME
	}
}

// stage name limited to 40 characters
func (r *table_row) pretty_name() string {
	s := r.name()
	if len(s) > 40 {
		s = s[:39]
	}
	return s
}

// add the values of one row to another one
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

func (t table_rows) Len() int      { return len(t) }
func (t table_rows) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

// sort by value (descending) but also by "name" (ascending) if the values are the same
func (t table_rows) Less(i, j int) bool {
	return (t[i].SUM_TIMER_WAIT > t[j].SUM_TIMER_WAIT) ||
		((t[i].SUM_TIMER_WAIT == t[j].SUM_TIMER_WAIT) && (t[i].EVENT_NAME < t[j].EVENT_NAME))
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

// stage headings
func (r *table_row) headings() string {
	return fmt.Sprintf("%-40s|%10s %6s %8s|", "Stage Name", "Latency", "%", "Counter")
}

// generate a printable result
func (r *table_row) row_content(totals table_row) string {
	name := r.pretty_name()
	if r.COUNT_STAR == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%-40s|%10s %6s %8s|",
		name,
		lib.FormatTime(r.SUM_TIMER_WAIT),
		lib.FormatPct(lib.MyDivide(r.SUM_TIMER_WAIT, totals.SUM_TIMER_WAIT)),
		lib.FormatAmount(r.COUNT_STAR))
}

// describe a whole row
func (r table_row) String() string {
	return fmt.Sprintf("%-40s %10s %10s",
		r.pretty_name(),
		lib.FormatTime(r.SUM_TIMER_WAIT),
		lib.FormatAmount(r.COUNT_STAR))
}

// describe a whole table
func (t table_rows) String() string {
	s := make([]string, len(t))

	for i := range t {
		s = append(s, t[i].String())
	}

	return strings.Join(s, "\n")
}

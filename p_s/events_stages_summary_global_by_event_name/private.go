package events_stages_summary_global_by_event_name

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/sjmudd/ps-top/lib"
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
type tableRow struct {
	EVENT_NAME     string
	COUNT_STAR     uint64
	SUM_TIMER_WAIT uint64
}

// a table of rows
type tableRows []tableRow

// select the rows into table
func selectRows(dbh *sql.DB) tableRows {
	var t tableRows

	lib.Logger.Println("events_stages_summary_global_by_event_name.selectRows()")
	sql := "SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT FROM events_stages_summary_global_by_event_name WHERE SUM_TIMER_WAIT > 0"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r tableRow
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
func (rows tableRows) needsRefresh(otherRows tableRows) bool {
	myTotals := rows.totals()
	otherTotals := otherRows.totals()

	return myTotals.SUM_TIMER_WAIT > otherTotals.SUM_TIMER_WAIT
}

// generate the totals of a table
func (rows tableRows) totals() tableRow {
	var totals tableRow
	totals.EVENT_NAME = "Totals"

	for i := range rows {
		totals.add(rows[i])
	}

	return totals
}

// return the stage name, removing any leading stage/sql/
func (row *tableRow) name() string {
	if len(row.EVENT_NAME) > 10 && row.EVENT_NAME[0:10] == "stage/sql/" {
		return row.EVENT_NAME[10:]
	}
	return row.EVENT_NAME
}

// add the values of one row to another one
func (row *tableRow) add(other tableRow) {
	row.SUM_TIMER_WAIT += other.SUM_TIMER_WAIT
	row.COUNT_STAR += other.COUNT_STAR
}

// subtract the countable values in one row from another
func (row *tableRow) subtract(other tableRow) {
	// check for issues here (we have a bug) and log it
	// - this situation should not happen so there's a logic bug somewhere else
	if row.SUM_TIMER_WAIT >= other.SUM_TIMER_WAIT {
		row.SUM_TIMER_WAIT -= other.SUM_TIMER_WAIT
		row.COUNT_STAR -= other.COUNT_STAR
	} else {
		lib.Logger.Println("WARNING: tableRow.subtract() - subtraction problem! (not subtracting)")
		lib.Logger.Println("row=", row)
		lib.Logger.Println("other=", other)
	}
}

func (rows tableRows) Len() int      { return len(rows) }
func (rows tableRows) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }

// sort by value (descending) but also by "name" (ascending) if the values are the same
func (rows tableRows) Less(i, j int) bool {
	return (rows[i].SUM_TIMER_WAIT > rows[j].SUM_TIMER_WAIT) ||
		((rows[i].SUM_TIMER_WAIT == rows[j].SUM_TIMER_WAIT) && (rows[i].EVENT_NAME < rows[j].EVENT_NAME))
}

func (rows tableRows) Sort() {
	sort.Sort(rows)
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
		name := (*rows)[i].name()
		if _, ok := initialByName[name]; ok {
			initialIndex := initialByName[name]
			(*rows)[i].subtract(initial[initialIndex])
		}
	}
}

// stage headings
func (row *tableRow) headings() string {
	return fmt.Sprintf("%10s %6s %8s|%s", "Latency", "%", "Counter", "Stage Name")
}

// generate a printable result
func (row *tableRow) rowContent(totals tableRow) string {
	name := row.name()
	if row.COUNT_STAR == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s %8s|%s",
		lib.FormatTime(row.SUM_TIMER_WAIT),
		lib.FormatPct(lib.MyDivide(row.SUM_TIMER_WAIT, totals.SUM_TIMER_WAIT)),
		lib.FormatAmount(row.COUNT_STAR),
		name)
}

// String describes a whole row
func (row tableRow) String() string {
	return fmt.Sprintf("%10s %10s %s",
		lib.FormatTime(row.SUM_TIMER_WAIT),
		lib.FormatAmount(row.COUNT_STAR),
		row.name())
}

// String describes a whole table
func (rows tableRows) String() string {
	s := make([]string, len(rows))

	for i := range rows {
		s = append(s, rows[i].String())
	}

	return strings.Join(s, "\n")
}

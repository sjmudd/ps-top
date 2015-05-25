// Package events_waits_summary_global_by_event_name contains the library routines for managing the
// events_waits_summary_global_by_event_name table
package events_waits_summary_global_by_event_name

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/sjmudd/ps-top/lib"
)

// tableRow contains a row from performance_schema.events_waits_summary_global_by_event_name
// Note: upper case names to match the performance_schema column names.
// This type is _not_ meant to be exported.
type tableRow struct {
	EVENT_NAME     string
	SUM_TIMER_WAIT uint64
	COUNT_STAR     uint64
}
type tableRows []tableRow

// trim off the leading 'wait/synch/mutex/innodb/'
func (row *tableRow) name() string {
	var n string
	if row.EVENT_NAME == "Totals" {
		n += row.EVENT_NAME
	} else if len(row.EVENT_NAME) >= 24 {
		n += row.EVENT_NAME[24:]
	}
	return n
}

func (row *tableRow) headings() string {
	return fmt.Sprintf("%10s %6s %6s %s", "Latency", "MtxCnt", "%", "Mutex Name")
}

// generate a printable result
func (row *tableRow) rowContent(totals tableRow) string {
	name := row.name()
	if row.COUNT_STAR == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s %6s|%s",
		lib.FormatTime(row.SUM_TIMER_WAIT),
		lib.FormatAmount(row.COUNT_STAR),
		lib.FormatPct(lib.MyDivide(row.SUM_TIMER_WAIT, totals.SUM_TIMER_WAIT)),
		name)
}

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

func (rows tableRows) totals() tableRow {
	var totals tableRow
	totals.EVENT_NAME = "Totals"

	for i := range rows {
		totals.add(rows[i])
	}

	return totals
}

func selectRows(dbh *sql.DB) tableRows {
	var t tableRows

	// we collect all information even if it's mainly empty as we may reference it later
	sql := "SELECT EVENT_NAME, SUM_TIMER_WAIT, COUNT_STAR FROM events_waits_summary_global_by_event_name WHERE SUM_TIMER_WAIT > 0 AND EVENT_NAME LIKE 'wait/synch/mutex/innodb/%'"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r tableRow
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

func (rows tableRows) Len() int      { return len(rows) }
func (rows tableRows) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }

// sort by value (descending) but also by "name" (ascending) if the values are the same
func (rows tableRows) Less(i, j int) bool {
	return rows[i].SUM_TIMER_WAIT > rows[j].SUM_TIMER_WAIT
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

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (rows tableRows) needsRefresh(otherRows tableRows) bool {
	totals := rows.totals()
	otherTotals := otherRows.totals()

	return totals.SUM_TIMER_WAIT > otherTotals.SUM_TIMER_WAIT
}

// describe a whole row
func (row tableRow) String() string {
	return fmt.Sprintf("%s|%10s %6s %6s",
		row.name(),
		lib.FormatTime(row.SUM_TIMER_WAIT),
		lib.FormatAmount(row.COUNT_STAR),
		lib.FormatPct(lib.MyDivide(row.SUM_TIMER_WAIT, row.SUM_TIMER_WAIT)))
}

// describe a whole table
func (rows tableRows) String() string {
	s := make([]string, len(rows))

	for i := range rows {
		s = append(s, rows[i].String())
	}

	return strings.Join(s, "\n")
}

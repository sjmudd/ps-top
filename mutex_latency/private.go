// Package mutex_latency contains the library routines for managing the
// events_waits_summary_global_by_event_name table
package mutex_latency

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/logger"
)

// Row contains a row from performance_schema.events_waits_summary_global_by_event_name
// Note: upper case names to match the performance_schema column names.
// This type is _not_ meant to be exported.
type Row struct {
	eventName    string
	sumTimerWait uint64
	countStar    uint64
}

// Rows contains a slice of Row
type Rows []Row

// trim off the leading 'wait/synch/mutex/innodb/'
func (row *Row) name() string {
	var n string
	if row.eventName == "Totals" {
		n += row.eventName
	} else if len(row.eventName) >= 24 {
		n += row.eventName[24:]
	}
	return n
}

func (row *Row) headings() string {
	return fmt.Sprintf("%10s %6s %6s %s", "Latency", "MtxCnt", "%", "Mutex Name")
}

// generate a printable result
func (row *Row) rowContent(totals Row) string {
	name := row.name()
	if row.countStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s %6s|%s",
		lib.FormatTime(row.sumTimerWait),
		lib.FormatAmount(row.countStar),
		lib.FormatPct(lib.MyDivide(row.sumTimerWait, totals.sumTimerWait)),
		name)
}

func (row *Row) add(other Row) {
	row.sumTimerWait += other.sumTimerWait
	row.countStar += other.countStar
}

// subtract the countable values in one row from another
func (row *Row) subtract(other Row) {
	// check for issues here (we have a bug) and log it
	// - this situation should not happen so there's a logic bug somewhere else
	if row.sumTimerWait >= other.sumTimerWait {
		row.sumTimerWait -= other.sumTimerWait
		row.countStar -= other.countStar
	} else {
		logger.Println("WARNING: Row.subtract() - subtraction problem! (not subtracting)")
		logger.Println("row=", row)
		logger.Println("other=", other)
	}
}

func (rows Rows) totals() Row {
	var totals Row
	totals.eventName = "Totals"

	for i := range rows {
		totals.add(rows[i])
	}

	return totals
}

func selectRows(dbh *sql.DB) Rows {
	var t Rows

	// we collect all information even if it's mainly empty as we may reference it later
	sql := "SELECT EVENT_NAME, SUM_TIMER_WAIT, COUNT_STAR FROM events_waits_summary_global_by_event_name WHERE SUM_TIMER_WAIT > 0 AND EVENT_NAME LIKE 'wait/synch/mutex/innodb/%'"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r Row
		if err := rows.Scan(
			&r.eventName,
			&r.sumTimerWait,
			&r.countStar); err != nil {
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

func (rows Rows) Len() int      { return len(rows) }
func (rows Rows) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }

// sort by value (descending) but also by "name" (ascending) if the values are the same
func (rows Rows) Less(i, j int) bool {
	return rows[i].sumTimerWait > rows[j].sumTimerWait
}

func (rows Rows) sort() {
	sort.Sort(rows)
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
		name := (*rows)[i].name()
		if _, ok := initialByName[name]; ok {
			initialIndex := initialByName[name]
			(*rows)[i].subtract(initial[initialIndex])
		}
	}
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (rows Rows) needsRefresh(otherRows Rows) bool {
	totals := rows.totals()
	otherTotals := otherRows.totals()

	return totals.sumTimerWait > otherTotals.sumTimerWait
}

// describe a whole row
func (row Row) String() string {
	return fmt.Sprintf("%s|%10s %6s %6s",
		row.name(),
		lib.FormatTime(row.sumTimerWait),
		lib.FormatAmount(row.countStar),
		lib.FormatPct(lib.MyDivide(row.sumTimerWait, row.sumTimerWait)))
}

// describe a whole table
func (rows Rows) String() string {
	s := make([]string, len(rows))

	for i := range rows {
		s = append(s, rows[i].String())
	}

	return strings.Join(s, "\n")
}

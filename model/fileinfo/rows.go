// Package fileinfo contains the routines for
// managing the file_summary_by_instance table.
package fileinfo

import (
	"database/sql"
	"log"
	"time"

	"github.com/sjmudd/ps-top/mylog"
)

// Config provides an interface for getting a configuration value from a key/value store
type Config interface {
	Get(setting string) string
}

// Rows represents a slice of Row
type Rows []Row

func (rows Rows) log() {
	for i := range rows {
		log.Println(i, rows[i])
	}
}

// return the totals of a slice of rows
func totals(rows Rows) Row {
	total := Row{Name: "Totals"}

	for _, row := range rows {
		total = add(total, row)
	}

	return total
}

// Valid determines if each individual row is valid
func (rows Rows) Valid() bool {
	valid := true
	for i := range rows {
		if rows[i].Valid(false) {
			valid = false
		}
	}
	return valid
}

// Select the raw data from the database into Rows
func collect(dbh *sql.DB) Rows {
	log.Println("collect() starts")
	var t Rows
	start := time.Now()

	sql := `
SELECT	FILE_NAME,
	SUM_TIMER_WAIT,
	SUM_TIMER_READ,
	SUM_TIMER_WRITE,
	SUM_NUMBER_OF_BYTES_READ,
	SUM_NUMBER_OF_BYTES_WRITE,
	SUM_TIMER_MISC,
	COUNT_STAR,
	COUNT_READ,
	COUNT_WRITE,
	COUNT_MISC
FROM	file_summary_by_instance
WHERE	SUM_TIMER_WAIT > 0
`

	rows, err := dbh.Query(sql)
	if err != nil {
		mylog.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r Row

		if err := rows.Scan(
			&r.Name, // raw filename
			&r.SumTimerWait,
			&r.SumTimerRead,
			&r.SumTimerWrite,
			&r.SumNumberOfBytesRead,
			&r.SumNumberOfBytesWrite,
			&r.SumTimerMisc,
			&r.CountStar,
			&r.CountRead,
			&r.CountWrite,
			&r.CountMisc); err != nil {
			mylog.Fatal(err)
		}
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		mylog.Fatal(err)
	}
	if !t.Valid() {
		log.Println("WARNING: collect(): t is invalid")
	}
	log.Println("collect() took:", time.Duration(time.Since(start)).String(), "and returned", len(t), "rows")

	return t
}

// subtract compares 2 slices of rows by name and removes the initial values
// - if we find a row we can not match we leave the row untouched
func (rows *Rows) subtract(initial Rows) {
	// make temporary copy for debugging.
	tempRows := make(Rows, len(*rows))
	copy(tempRows, *rows)

	if !rows.Valid() {
		log.Println("WARNING: Rows.subtract(): rows is invalid (pre)")
	}
	if !initial.Valid() {
		log.Println("WARNING: Rows.subtract(): initial is invalid (pre)")
	}

	// check that initial is "earlier"
	rowsT := totals(*rows)
	initialT := totals(initial)
	if rowsT.SumTimerWait < initialT.SumTimerWait {
		log.Println("BUG: (rows *Rows) subtract(initial): rows < initial")
		log.Println("sum(rows):  ", rowsT)
		log.Println("sum(initial):", initialT)
	}

	iByName := make(map[string]int)

	// iterate over rows by name
	for i := range initial {
		iByName[initial[i].Name] = i
	}

	for i := range *rows {
		if _, ok := iByName[(*rows)[i].Name]; ok {
			initialI := iByName[(*rows)[i].Name]
			(*rows)[i] = subtract((*rows)[i], initial[initialI])
		}
	}
	if !rows.Valid() {
		log.Println("WARNING: Rows.subtract(): rows is invalid (post)")
		log.Println("WARNING: tempRows:")
		tempRows.log()
		log.Println("WARNING: initial:")
		initial.log()
		log.Println("WARNING: END")
	}
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (rows Rows) needsRefresh(otherRows Rows) bool {
	return totals(rows).SumTimerWait > totals(otherRows).SumTimerWait
}

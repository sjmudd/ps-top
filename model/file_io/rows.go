// Package file_io contains the routines for
// managing the file_summary_by_instance table.
package file_io

import (
	"database/sql"
	"log"
	"time"

	"github.com/sjmudd/ps-top/global"
	"github.com/sjmudd/ps-top/logger"
)

// Rows represents a slice of Row
type Rows []Row

func (rows Rows) logger() {
	for i := range rows {
		logger.Println(i, rows[i])
	}
}

// return the totals of a slice of rows
func (rows Rows) totals() Row {
	var totals Row
	totals.Name = "Totals"

	for i := range rows {
		totals = add(totals, rows[i])
	}

	return totals
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

// Convert the imported rows to a merged one with merged data.
// - Combine all entries with the same "name" by adding their values.
func (rows Rows) mergeByName(globalVariables *global.Variables) Rows {
	start := time.Now()
	rowsByName := make(map[string]Row)

	var newName string
	for i := range rows {
		var newRow Row

		if rows[i].SumTimerWait > 0 {
			newName = simplify(rows[i].Name, globalVariables)

			// check if we have an entry in the map
			if _, found := rowsByName[newName]; found {
				newRow = rowsByName[newName]
			} else {
				newRow = Row{Name: newName} // empty row with new name
			}
			newRow = add(newRow, rows[i])
			rowsByName[newName] = newRow // update the map with the new summed value
		}
	}

	// add the map contents back into the table
	var mergedRows Rows
	for _, row := range rowsByName {
		mergedRows = append(mergedRows, row)
	}
	if !mergedRows.Valid() {
		logger.Println("WARNING: mergeByName(): mergedRows is invalid")
	}

	logger.Println("mergeByName() took:", time.Duration(time.Since(start)).String(), "and returned", len(rowsByName), "rows")
	return mergedRows
}

// Select the raw data from the database into Rows
func collect(dbh *sql.DB) Rows {
	logger.Println("collect() starts")
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
		log.Fatal(err)
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
			log.Fatal(err)
		}
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	if !t.Valid() {
		logger.Println("WARNING: collect(): t is invalid")
	}
	logger.Println("collect() took:", time.Duration(time.Since(start)).String(), "and returned", len(t), "rows")

	return t
}

// subtract compares 2 slices of rows by name and removes the initial values
// - if we find a row we can not match we leave the row untouched
func (rows *Rows) subtract(initial Rows) {
	// make temporary copy for debugging.
	tempRows := make(Rows, len(*rows))
	copy(tempRows, *rows)

	if !rows.Valid() {
		logger.Println("WARNING: Rows.subtract(): rows is invalid (pre)")
	}
	if !initial.Valid() {
		logger.Println("WARNING: Rows.subtract(): initial is invalid (pre)")
	}

	// check that initial is "earlier"
	rowsT := rows.totals()
	initialT := initial.totals()
	if rowsT.SumTimerWait < initialT.SumTimerWait {
		logger.Println("BUG: (rows *Rows) subtract(initial): rows < initial")
		logger.Println("sum(rows):  ", rowsT)
		logger.Println("sum(intial):", initialT)
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
		logger.Println("WARNING: Rows.subtract(): rows is invalid (post)")
		logger.Println("WARNING: tempRows:")
		tempRows.logger()
		logger.Println("WARNING: initial:")
		initial.logger()
		logger.Println("WARNING: END")
	}
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (rows Rows) needsRefresh(t2 Rows) bool {
	myTotals := rows.totals()
	otherTotals := t2.totals()

	return (myTotals.SumTimerWait > otherTotals.SumTimerWait) || (myTotals.CountStar > otherTotals.CountStar)
}

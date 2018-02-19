// Package file_io_latency contains the routines for
// managing the file_summary_by_instance table.
package file_io_latency

import (
	"database/sql"
	"log"
	"regexp"
	"sort"
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
	totals.name = "Totals"

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

		if rows[i].sumTimerWait > 0 {
			newName = rows[i].simplifyName(globalVariables)

			// check if we have an entry in the map
			if _, found := rowsByName[newName]; found {
				newRow = rowsByName[newName]
			} else {
				newRow = Row{name: newName} // empty row with new name
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

// used for testing
// usage: match(r.name, "demodb.table")
func match(text string, searchFor string) bool {
	re := regexp.MustCompile(searchFor)

	result := re.MatchString(text)

	logger.Println("match(", text, ",", searchFor, ")", result)

	return result
}

// Select the raw data from the database into Rows
// - filter out empty values
// - merge rows with the same name into a single row
// - change name into a more descriptive value.
func selectRows(dbh *sql.DB) Rows {
	alwaysAdd := true // false for testing

	logger.Println("selectRows() starts")
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
			&r.name, // raw filename
			&r.sumTimerWait,
			&r.sumTimerRead,
			&r.sumTimerWrite,
			&r.sumNumberOfBytesRead,
			&r.sumNumberOfBytesWrite,
			&r.sumTimerMisc,
			&r.countStar,
			&r.countRead,
			&r.countWrite,
			&r.countMisc); err != nil {
			log.Fatal(err)
		}

		if alwaysAdd || match(r.name, "demodb.table") {
			t = append(t, r)
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	if !t.Valid() {
		logger.Println("WARNING: selectRows(): t is invalid")
	}
	logger.Println("selectRows() took:", time.Duration(time.Since(start)).String(), "and returned", len(t), "rows")

	return t
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
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
	if rowsT.sumTimerWait < initialT.sumTimerWait {
		logger.Println("BUG: (rows *Rows) subtract(initial): rows < initial")
		logger.Println("sum(rows):  ", rowsT)
		logger.Println("sum(intial):", initialT)
	}

	iByName := make(map[string]int)

	// iterate over rows by name
	for i := range initial {
		iByName[initial[i].name] = i
	}

	for i := range *rows {
		if _, ok := iByName[(*rows)[i].name]; ok {
			initialI := iByName[(*rows)[i].name]
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

func (rows Rows) Len() int      { return len(rows) }
func (rows Rows) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }
func (rows Rows) Less(i, j int) bool {
	return (rows[i].sumTimerWait > rows[j].sumTimerWait) ||
		((rows[i].sumTimerWait == rows[j].sumTimerWait) && (rows[i].name < rows[j].name))
}

func (rows *Rows) sort() {
	sort.Sort(rows)
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (rows Rows) needsRefresh(t2 Rows) bool {
	myTotals := rows.totals()
	otherTotals := t2.totals()

	return (myTotals.sumTimerWait > otherTotals.sumTimerWait) || (myTotals.countStar > otherTotals.countStar)
}

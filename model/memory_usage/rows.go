// Package memory_usage contains the library
// routines for managing memory_summary_global_by_event_name table.
package memory_usage

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql" // keep glint happy
	"log"
	"sort"

	"github.com/sjmudd/ps-top/logger"
)

// Rows contains multiple rows
type Rows []Row

// return the totals of a slice of rows
func (t Rows) totals() Row {
	var totals Row
	totals.Name = "Totals"

	for i := range t {
		totals.add(t[i])
	}

	return totals
}

// catch a SELECT error - specifically this one.
// Error 1146: Table 'performance_schema.memory_summary_global_by_event_name' doesn't exist
func sqlErrorHandler(err error) bool {
	var ignore bool

	logger.Println("- SELECT gave an error:", err.Error())
	if err.Error()[0:11] != "Error 1146:" {
		fmt.Println(fmt.Sprintf("XXX'%s'XXX", err.Error()[0:11]))
		log.Fatal("Unexpected error", fmt.Sprintf("XXX'%s'XXX", err.Error()[0:11]))
		// log.Fatal("Unexpected error:", err.Error())
	} else {
		logger.Println("- expected error, so ignoring")
		ignore = true
	}

	return ignore
}

// Select the raw data from the database
func collect(dbh *sql.DB) Rows {
	var t Rows
	var skip bool

	sql := `-- memory_usage
SELECT	EVENT_NAME                                           AS eventName,
	CURRENT_COUNT_USED                                   AS currentCountUsed,
	HIGH_COUNT_USED                                      AS highCountUsed,
	CURRENT_NUMBER_OF_BYTES_USED                         AS currentBytesUsed,
	HIGH_NUMBER_OF_BYTES_USED                            AS highBytesUsed,
	COUNT_ALLOC + COUNT_FREE                             AS totalMemoryOps,
	SUM_NUMBER_OF_BYTES_ALLOC + SUM_NUMBER_OF_BYTES_FREE AS totalBytesManaged
FROM	memory_summary_global_by_event_name
WHERE	HIGH_COUNT_USED > 0`

	logger.Println("Querying db:", sql)
	rows, err := dbh.Query(sql)
	if err != nil {
		// FIXME - This should be caught by the validateViews() upstream but isn't for initial
		// FIXME   table collection. I'm waiting to clean up by splitting views and models but
		// FIXME   that has not been done yet so for now work aruond the initial app.CollectAll()
		// FIXME   by simply ignoring a request if the table does not exist.
		skip = sqlErrorHandler(err) // temporarily catch a SELECT error. // should not be necessary now
	}

	if !skip {
		defer rows.Close()

		for rows.Next() {
			var r Row
			if err := rows.Scan(
				&r.Name,
				&r.CurrentCountUsed,
				&r.HighCountUsed,
				&r.CurrentBytesUsed,
				&r.HighBytesUsed,
				&r.TotalMemoryOps,
				&r.TotalBytesManaged); err != nil {
				log.Fatal(err)
			}
			t = append(t, r)
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}
	}

	return t
}

func (t Rows) Len() int      { return len(t) }
func (t Rows) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t Rows) Less(i, j int) bool {
	return (t[i].CurrentBytesUsed > t[j].CurrentBytesUsed) ||
		((t[i].CurrentBytesUsed == t[j].CurrentBytesUsed) &&
			(t[i].Name < t[j].Name))

}

// sort the data
func (t *Rows) sort() {
	sort.Sort(t)
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
func (t *Rows) subtract(initial Rows) {
	iByName := make(map[string]int)

	// iterate over rows by name
	for i := range initial {
		iByName[initial[i].Name] = i
	}

	for i := range *t {
		if _, ok := iByName[(*t)[i].Name]; ok {
			initialI := iByName[(*t)[i].Name]
			(*t)[i].subtract(initial[initialI])
		}
	}
}

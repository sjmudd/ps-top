// Package memoryusage contains the library
// routines for managing memory_summary_global_by_event_name table.
package memoryusage

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql" // keep glint happy

	"github.com/sjmudd/ps-top/log"
)

// Rows contains multiple rows
type Rows []Row

// return the totals of a slice of rows
func totals(rows Rows) Row {
	total := Row{Name: "Totals"}

	for _, row := range rows {
		total.CurrentBytesUsed += row.CurrentBytesUsed
		total.TotalMemoryOps += row.TotalMemoryOps
		total.CurrentCountUsed += row.CurrentCountUsed
	}

	return total
}

// catch a SELECT error - specifically this one.
// Error 1146: Table 'performance_schema.memory_summary_global_by_event_name' doesn't exist
func sqlErrorHandler(err error) bool {
	var ignore bool

	log.Println("- SELECT gave an error:", err.Error())
	if err.Error()[0:11] != "Error 1146:" {
		fmt.Printf("XXX'%s'XXX\n", err.Error()[0:11])
		log.Fatal("Unexpected error", fmt.Sprintf("XXX'%s'XXX", err.Error()[0:11]))
	} else {
		log.Println("- expected error, so ignoring")
		ignore = true
	}

	return ignore
}

// Select the raw data from the database
func collect(dbh *sql.DB) Rows {
	var t Rows
	var skip bool

	sql := `-- memoryusage
SELECT	EVENT_NAME                                           AS eventName,
	CURRENT_COUNT_USED                                   AS currentCountUsed,
	HIGH_COUNT_USED                                      AS highCountUsed,
	CURRENT_NUMBER_OF_BYTES_USED                         AS currentBytesUsed,
	HIGH_NUMBER_OF_BYTES_USED                            AS highBytesUsed,
	COUNT_ALLOC + COUNT_FREE                             AS totalMemoryOps,
	SUM_NUMBER_OF_BYTES_ALLOC + SUM_NUMBER_OF_BYTES_FREE AS totalBytesManaged
FROM	memory_summary_global_by_event_name
WHERE	HIGH_COUNT_USED > 0`

	log.Println("Querying db:", sql)
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
				log.Fatalf("collect: rows.Scan() failed: %+v", err)
			}
			t = append(t, r)
		}
		if err := rows.Err(); err != nil {
			log.Fatalf("collect: rows.Err() returned: %v", err)
		}
	}

	return t
}

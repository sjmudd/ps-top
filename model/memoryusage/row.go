// Package memoryusage contains the library
// routines for managing memory_summary_global_by_event_name table.
package memoryusage

import (
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql"

	"github.com/sjmudd/ps-top/log"
)

/* This table does not exist in MySQL 5.6

// MySQL 5.7+
CREATE TABLE `memory_summary_global_by_event_name` (
  `EVENT_NAME` varchar(128) NOT NULL,
  `COUNT_ALLOC` bigint(20) unsigned NOT NULL,
  `COUNT_FREE` bigint(20) unsigned NOT NULL,
  `SUM_NUMBER_OF_BYTES_ALLOC` bigint(20) unsigned NOT NULL,
  `SUM_NUMBER_OF_BYTES_FREE` bigint(20) unsigned NOT NULL,
  `LOW_COUNT_USED` bigint(20) NOT NULL,
  `CURRENT_COUNT_USED` bigint(20) NOT NULL,
  `HIGH_COUNT_USED` bigint(20) NOT NULL,
  `LOW_NUMBER_OF_BYTES_USED` bigint(20) NOT NULL,
  `CURRENT_NUMBER_OF_BYTES_USED` bigint(20) NOT NULL,
  `HIGH_NUMBER_OF_BYTES_USED` bigint(20) NOT NULL
) ENGINE=PERFORMANCE_SCHEMA DEFAULT CHARSET=utf8

// MySQL 8.4
CREATE TABLE `memory_summary_global_by_event_name` (
  `EVENT_NAME` varchar(128) NOT NULL,
  `COUNT_ALLOC` bigint unsigned NOT NULL,
  `COUNT_FREE` bigint unsigned NOT NULL,
  `SUM_NUMBER_OF_BYTES_ALLOC` bigint unsigned NOT NULL,
  `SUM_NUMBER_OF_BYTES_FREE` bigint unsigned NOT NULL,
  `LOW_COUNT_USED` bigint NOT NULL,
  `CURRENT_COUNT_USED` bigint NOT NULL,
  `HIGH_COUNT_USED` bigint NOT NULL,
  `LOW_NUMBER_OF_BYTES_USED` bigint NOT NULL,
  `CURRENT_NUMBER_OF_BYTES_USED` bigint NOT NULL,
  `HIGH_NUMBER_OF_BYTES_USED` bigint NOT NULL,
  PRIMARY KEY (`EVENT_NAME`)
) ENGINE=PERFORMANCE_SCHEMA DEFAULT CHARSET=utf8mb3

*/

// Row holds a row of data from memory_summary_global_by_event_name
type Row struct {
	Name              string
	CurrentCountUsed  int64
	HighCountUsed     int64
	TotalMemoryOps    int64
	CurrentBytesUsed  int64
	HighBytesUsed     int64
	TotalBytesManaged uint64
}

// HasData returns true if there is valid data in the row
func (r *Row) HasData() bool {
	return r != nil && r.Name != "" && r.CurrentCountUsed != 0 && r.TotalMemoryOps != 0
}

// return the totals of a slice of rows
func totals(rows []Row) Row {
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

	// try to convert error to a MySQL error for better handling. The go-mysql-driver now reports differently.
	if mysqlError, ok := err.(*mysql.MySQLError); ok {
		log.Println("- SELECT gave error:", mysqlError.Number, mysqlError.SQLState, mysqlError.Message)
		if mysqlError.Number == 1146 {
			ignore = true
			log.Println("- expected error, so ignoring")
		} else {
			log.Println("- unexpected error, aborting")
		}
	} else {
		log.Println("- SELECT gave error:", err.Error())
		if err.Error()[0:11] != "Error 1146:" {
			ignore = true
			log.Println("- expected error, so ignoring")
		} else {
			log.Fatal(fmt.Sprintf("Unexpected error: got: '%s', expecting '%s', full error: '%s', aborting", err.Error()[0:11], "Error 1146:", err.Error()))
		}
	}

	return ignore
}

// Select the raw data from the database
func collect(db *sql.DB) []Row {
	var t []Row
	var skip bool

	sql := `SELECT EVENT_NAME, CURRENT_COUNT_USED, HIGH_COUNT_USED, CURRENT_NUMBER_OF_BYTES_USED, HIGH_NUMBER_OF_BYTES_USED, COUNT_ALLOC + COUNT_FREE, SUM_NUMBER_OF_BYTES_ALLOC + SUM_NUMBER_OF_BYTES_FREE FROM memory_summary_global_by_event_name WHERE HIGH_COUNT_USED > 0`

	log.Println("Querying db:", sql)
	rows, err := db.Query(sql)
	if err != nil {
		// FIXME - This should be caught by the validateViews() upstream but isn't for initial
		// FIXME   table collection. I'm waiting to clean up by splitting views and models but
		// FIXME   that has not been done yet so for now work around the initial app.CollectAll()
		// FIXME   by simply ignoring a request if the table does not exist.
		skip = sqlErrorHandler(err) // temporarily catch a SELECT error. // should not be necessary now
	}

	if !skip {
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
	_ = rows.Close()

	return t
}

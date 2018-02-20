// Package memory_usage contains the library
// routines for managing memory_summary_global_by_event_name table.
package memory_usage

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql" // keep glint happy
	"log"
	"sort"

	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/logger"
)

/* This table exists in MySQL 5.7 but not 5.6

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

*/

// Row holds a row of data from memory_summary_global_by_event_name
type Row struct {
	name              string
	currentCountUsed  int64
	highCountUsed     int64
	totalMemoryOps    int64
	currentBytesUsed  int64
	highBytesUsed     int64
	totalBytesManaged uint64
}

// Rows contains multiple rows
type Rows []Row

func (r *Row) headings() string {
	return fmt.Sprint("CurBytes         %  High Bytes|MemOps          %|CurAlloc       %  HiAlloc|Memory Area")
	//                         1234567890  100.0%  1234567890|123456789  100.0%|12345678  100.0%  12345678|Some memory name
}

// generate a printable result
func (r *Row) rowContent(totals Row) string {

	// assume the data is empty so hide it.
	name := r.name
	if r.totalMemoryOps == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s  %6s  %10s|%10s %6s|%8s  %6s  %8s|%s",
		lib.SignedFormatAmount(r.currentBytesUsed),
		lib.FormatPct(lib.SignedMyDivide(r.currentBytesUsed, totals.currentBytesUsed)),
		lib.SignedFormatAmount(r.highBytesUsed),
		lib.SignedFormatAmount(r.totalMemoryOps),
		lib.FormatPct(lib.SignedMyDivide(r.totalMemoryOps, totals.totalMemoryOps)),
		lib.SignedFormatAmount(r.currentCountUsed),
		lib.FormatPct(lib.SignedMyDivide(r.currentCountUsed, totals.currentCountUsed)),
		lib.SignedFormatAmount(r.highCountUsed),
		name)
}

func (r *Row) add(other Row) {
	r.currentBytesUsed += other.currentBytesUsed
	r.totalMemoryOps += other.totalMemoryOps
	r.currentCountUsed += other.currentCountUsed
}

func (r *Row) subtract(other Row) {
}

// return the totals of a slice of rows
func (t Rows) totals() Row {
	var totals Row
	totals.name = "Totals"

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
func selectRows(dbh *sql.DB) Rows {
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
				&r.name,
				&r.currentCountUsed,
				&r.highCountUsed,
				&r.currentBytesUsed,
				&r.highBytesUsed,
				&r.totalMemoryOps,
				&r.totalBytesManaged); err != nil {
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
	return (t[i].currentBytesUsed > t[j].currentBytesUsed) ||
		((t[i].currentBytesUsed == t[j].currentBytesUsed) &&
			//	return (t[i].totalMemoryOps > t[j].totalMemoryOps) ||
			//		((t[i].totalMemoryOps == t[j].totalMemoryOps) &&
			(t[i].name < t[j].name))

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
		iByName[initial[i].name] = i
	}

	for i := range *t {
		if _, ok := iByName[(*t)[i].name]; ok {
			initialI := iByName[(*t)[i].name]
			(*t)[i].subtract(initial[initialI])
		}
	}
}

func (t *Object) makeResults() {
	t.results = make(Rows, len(t.current))
	copy(t.results, t.current)
	t.results.sort()
	t.totals = t.results.totals()
}

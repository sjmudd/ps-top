// Package table_lock_latency contains the library
// routines for managing the table_lock_waits_summary_by_table table.
package table_lock_latency

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql" // keep glint happy
	"log"
	"sort"
	"strings"

	"github.com/sjmudd/ps-top/lib"
)

/*

From 5.7.5

*************************** 1. row ***************************
       Table: table_lock_waits_summary_by_table
Create Table: CREATE TABLE `table_lock_waits_summary_by_table` (
  `OBJECT_TYPE` varchar(64) DEFAULT NULL,
  `OBJECT_SCHEMA` varchar(64) DEFAULT NULL,
  `OBJECT_NAME` varchar(64) DEFAULT NULL,
  `COUNT_STAR` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WAIT` bigint(20) unsigned NOT NULL,
  `COUNT_READ` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ` bigint(20) unsigned NOT NULL,
  `COUNT_WRITE` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `COUNT_READ_NORMAL` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ_NORMAL` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ_NORMAL` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ_NORMAL` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ_NORMAL` bigint(20) unsigned NOT NULL,
  `COUNT_READ_WITH_SHARED_LOCKS` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ_WITH_SHARED_LOCKS` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ_WITH_SHARED_LOCKS` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ_WITH_SHARED_LOCKS` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ_WITH_SHARED_LOCKS` bigint(20) unsigned NOT NULL,
  `COUNT_READ_HIGH_PRIORITY` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ_HIGH_PRIORITY` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ_HIGH_PRIORITY` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ_HIGH_PRIORITY` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ_HIGH_PRIORITY` bigint(20) unsigned NOT NULL,
  `COUNT_READ_NO_INSERT` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ_NO_INSERT` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ_NO_INSERT` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ_NO_INSERT` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ_NO_INSERT` bigint(20) unsigned NOT NULL,
  `COUNT_READ_EXTERNAL` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_READ_EXTERNAL` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_READ_EXTERNAL` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_READ_EXTERNAL` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_READ_EXTERNAL` bigint(20) unsigned NOT NULL,
  `COUNT_WRITE_ALLOW_WRITE` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE_ALLOW_WRITE` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE_ALLOW_WRITE` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE_ALLOW_WRITE` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE_ALLOW_WRITE` bigint(20) unsigned NOT NULL,
  `COUNT_WRITE_CONCURRENT_INSERT` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE_CONCURRENT_INSERT` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE_CONCURRENT_INSERT` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE_CONCURRENT_INSERT` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE_CONCURRENT_INSERT` bigint(20) unsigned NOT NULL,
  `COUNT_WRITE_LOW_PRIORITY` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE_LOW_PRIORITY` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE_LOW_PRIORITY` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE_LOW_PRIORITY` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE_LOW_PRIORITY` bigint(20) unsigned NOT NULL,
  `COUNT_WRITE_NORMAL` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE_NORMAL` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE_NORMAL` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE_NORMAL` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE_NORMAL` bigint(20) unsigned NOT NULL,
  `COUNT_WRITE_EXTERNAL` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE_EXTERNAL` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE_EXTERNAL` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE_EXTERNAL` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE_EXTERNAL` bigint(20) unsigned NOT NULL
) ENGINE=PERFORMANCE_SCHEMA DEFAULT CHARSET=utf8

*/

// Row holds a row of data from table_lock_waits_summary_by_table
type Row struct {
	tableName                      string // combination of <schema>.<table>
	countStar                      int
	sumTimerWait                   uint64
	sumTimerRead                   uint64
	sumTimerWrite                  uint64
	sumTimerReadWithSharedLocks    uint64
	sumTimerReadHighPriority       uint64
	sumTimerReadNoInsert           uint64
	sumTimerReadNormal             uint64
	sumTimerReadExternal           uint64
	sumTimerWriteAllowWrite        uint64
	sumTimerWriteConcurrencyInsert uint64
	sumTimerWriteLowPriority       uint64
	sumTimerWriteNormal            uint64
	sumTimerWriteExternal          uint64
}

// Rows contains multiple rows
type Rows []Row

// return the table name from the columns as '<schema>.<table>'
func (r *Row) name() string {
	return r.tableName
}

// Latency      %|  Read  Write|S.Lock   High  NoIns Normal Extrnl|AlloWr CncIns WrtDly    Low Normal Extrnl|
// 1234567 100.0%|xxxxx% xxxxx%|xxxxx% xxxxx% xxxxx% xxxxx% xxxxx%|xxxxx% xxxxx% xxxxx% xxxxx% xxxxx% xxxxx%|xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
func (r *Row) headings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s|%-30s",
		"Latency", "%",
		"Read", "Write",
		"S.Lock", "High", "NoIns", "Normal", "Extrnl",
		"AlloWr", "CncIns", "Low", "Normal", "Extrnl",
		"Table Name")
}

// generate a printable result
func (r *Row) rowContent(totals Row) string {

	// assume the data is empty so hide it.
	name := r.name()
	if r.countStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s|%s",
		lib.FormatTime(r.sumTimerWait),
		lib.FormatPct(lib.MyDivide(r.sumTimerWait, totals.sumTimerWait)),

		lib.FormatPct(lib.MyDivide(r.sumTimerRead, r.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(r.sumTimerWrite, r.sumTimerWait)),

		lib.FormatPct(lib.MyDivide(r.sumTimerReadWithSharedLocks, r.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(r.sumTimerReadHighPriority, r.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(r.sumTimerReadNoInsert, r.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(r.sumTimerReadNormal, r.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(r.sumTimerReadExternal, r.sumTimerWait)),

		lib.FormatPct(lib.MyDivide(r.sumTimerWriteAllowWrite, r.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(r.sumTimerWriteConcurrencyInsert, r.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(r.sumTimerWriteLowPriority, r.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(r.sumTimerWriteNormal, r.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(r.sumTimerWriteExternal, r.sumTimerWait)),
		name)
}

func (r *Row) add(other Row) {
	r.countStar += other.countStar
	r.sumTimerWait += other.sumTimerWait
	r.sumTimerRead += other.sumTimerRead
	r.sumTimerWrite += other.sumTimerWrite
	r.sumTimerReadWithSharedLocks += other.sumTimerReadWithSharedLocks
	r.sumTimerReadHighPriority += other.sumTimerReadHighPriority
	r.sumTimerReadNoInsert += other.sumTimerReadNoInsert
	r.sumTimerReadNormal += other.sumTimerReadNormal
	r.sumTimerReadExternal += other.sumTimerReadExternal
	r.sumTimerWriteConcurrencyInsert += other.sumTimerWriteConcurrencyInsert
	r.sumTimerWriteLowPriority += other.sumTimerWriteLowPriority
	r.sumTimerWriteNormal += other.sumTimerWriteNormal
	r.sumTimerWriteExternal += other.sumTimerWriteExternal
}

func (r *Row) subtract(other Row) {
	r.countStar -= other.countStar
	r.sumTimerWait -= other.sumTimerWait
	r.sumTimerRead -= other.sumTimerRead
	r.sumTimerWrite -= other.sumTimerWrite
	r.sumTimerReadWithSharedLocks -= other.sumTimerReadWithSharedLocks
	r.sumTimerReadHighPriority -= other.sumTimerReadHighPriority
	r.sumTimerReadNoInsert -= other.sumTimerReadNoInsert
	r.sumTimerReadNormal -= other.sumTimerReadNormal
	r.sumTimerReadExternal -= other.sumTimerReadExternal
	r.sumTimerWriteConcurrencyInsert -= other.sumTimerWriteConcurrencyInsert
	r.sumTimerWriteLowPriority -= other.sumTimerWriteLowPriority
	r.sumTimerWriteNormal -= other.sumTimerWriteNormal
	r.sumTimerWriteExternal -= other.sumTimerWriteExternal
}

// return the totals of a slice of rows
func (t Rows) totals() Row {
	var totals Row
	totals.tableName = "Totals"

	for i := range t {
		totals.add(t[i])
	}

	return totals
}

// Select the raw data from the database into file_summary_by_instance_rows
// - filter out empty values
// - merge rows with the same name into a single row
// - change FILE_NAME into a more descriptive value.
func selectRows(dbh *sql.DB) Rows {
	var t Rows

	sql := "SELECT OBJECT_SCHEMA, OBJECT_NAME, COUNT_STAR, SUM_TIMER_WAIT, SUM_TIMER_READ, SUM_TIMER_WRITE, SUM_TIMER_READ_WITH_SHARED_LOCKS, SUM_TIMER_READ_HIGH_PRIORITY, SUM_TIMER_READ_NO_INSERT, SUM_TIMER_READ_NORMAL, SUM_TIMER_READ_EXTERNAL, SUM_TIMER_WRITE_ALLOW_WRITE, SUM_TIMER_WRITE_CONCURRENT_INSERT, SUM_TIMER_WRITE_LOW_PRIORITY, SUM_TIMER_WRITE_NORMAL, SUM_TIMER_WRITE_EXTERNAL FROM table_lock_waits_summary_by_table WHERE COUNT_STAR > 0"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r Row
		var schema, table string

		if err := rows.Scan(
			&schema,
			&table,
			&r.countStar,
			&r.sumTimerWait,
			&r.sumTimerRead,
			&r.sumTimerWrite,
			&r.sumTimerReadWithSharedLocks,
			&r.sumTimerReadHighPriority,
			&r.sumTimerReadNoInsert,
			&r.sumTimerReadNormal,
			&r.sumTimerReadExternal,
			&r.sumTimerWriteAllowWrite,
			&r.sumTimerWriteConcurrencyInsert,
			&r.sumTimerWriteLowPriority,
			&r.sumTimerWriteNormal,
			&r.sumTimerWriteExternal); err != nil {
			log.Fatal(err)
		}
		r.tableName = lib.TableName(schema, table)
		// we collect all data as we may need it later
		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return t
}

func (t Rows) Len() int      { return len(t) }
func (t Rows) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t Rows) Less(i, j int) bool {
	return (t[i].sumTimerWait > t[j].sumTimerWait) ||
		((t[i].sumTimerWait == t[j].sumTimerWait) &&
			(t[i].tableName < t[j].tableName))

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
		iByName[initial[i].name()] = i
	}

	for i := range *t {
		if _, ok := iByName[(*t)[i].name()]; ok {
			initialI := iByName[(*t)[i].name()]
			(*t)[i].subtract(initial[initialI])
		}
	}
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (t Rows) needsRefresh(t2 Rows) bool {
	myTotals := t.totals()
	otherTotals := t2.totals()

	return myTotals.sumTimerWait > otherTotals.sumTimerWait
}

// describe a whole row
func (r Row) String() string {
	return fmt.Sprintf("%10s %10s %10s|%10s %10s %10s %10s %10s|%10s %10s %10s %10s %10s|%s",
		lib.FormatTime(r.sumTimerWait),
		lib.FormatTime(r.sumTimerRead),
		lib.FormatTime(r.sumTimerWrite),

		lib.FormatTime(r.sumTimerReadWithSharedLocks),
		lib.FormatTime(r.sumTimerReadHighPriority),
		lib.FormatTime(r.sumTimerReadNoInsert),
		lib.FormatTime(r.sumTimerReadNormal),
		lib.FormatTime(r.sumTimerReadExternal),

		lib.FormatTime(r.sumTimerWriteAllowWrite),
		lib.FormatTime(r.sumTimerWriteConcurrencyInsert),
		lib.FormatTime(r.sumTimerWriteLowPriority),
		lib.FormatTime(r.sumTimerWriteNormal),
		lib.FormatTime(r.sumTimerWriteExternal),
		r.name())
}

// describe a whole table
func (t Rows) String() string {
	s := make([]string, len(t))

	for i := range t {
		s = append(s, t[i].String())
	}

	return strings.Join(s, "\n")
}

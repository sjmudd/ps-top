// Package table_lock_latency contains the library
// routines for managing the table_lock_waits_summary_by_table table.
package table_lock_latency

import (
	"fmt"

	"github.com/sjmudd/ps-top/lib"
)

/*
From 5.7.5:

CREATE TABLE `table_lock_waits_summary_by_table` (
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
	name                          string // combination of <schema>.<table>
	sumTimerWait                  uint64
	sumTimerRead                  uint64
	sumTimerWrite                 uint64
	sumTimerReadWithSharedLocks   uint64
	sumTimerReadHighPriority      uint64
	sumTimerReadNoInsert          uint64
	sumTimerReadNormal            uint64
	sumTimerReadExternal          uint64
	sumTimerWriteAllowWrite       uint64
	sumTimerWriteConcurrentInsert uint64
	sumTimerWriteLowPriority      uint64
	sumTimerWriteNormal           uint64
	sumTimerWriteExternal         uint64
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
func (r *Row) content(totals Row) string {

	// assume the data is empty so hide it.
	name := r.name
	if r.sumTimerWait == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s|%s",
		lib.FormatTime(r.sumTimerWait),
		lib.FormatPct(lib.Divide(r.sumTimerWait, totals.sumTimerWait)),

		lib.FormatPct(lib.Divide(r.sumTimerRead, r.sumTimerWait)),
		lib.FormatPct(lib.Divide(r.sumTimerWrite, r.sumTimerWait)),

		lib.FormatPct(lib.Divide(r.sumTimerReadWithSharedLocks, r.sumTimerWait)),
		lib.FormatPct(lib.Divide(r.sumTimerReadHighPriority, r.sumTimerWait)),
		lib.FormatPct(lib.Divide(r.sumTimerReadNoInsert, r.sumTimerWait)),
		lib.FormatPct(lib.Divide(r.sumTimerReadNormal, r.sumTimerWait)),
		lib.FormatPct(lib.Divide(r.sumTimerReadExternal, r.sumTimerWait)),

		lib.FormatPct(lib.Divide(r.sumTimerWriteAllowWrite, r.sumTimerWait)),
		lib.FormatPct(lib.Divide(r.sumTimerWriteConcurrentInsert, r.sumTimerWait)),
		lib.FormatPct(lib.Divide(r.sumTimerWriteLowPriority, r.sumTimerWait)),
		lib.FormatPct(lib.Divide(r.sumTimerWriteNormal, r.sumTimerWait)),
		lib.FormatPct(lib.Divide(r.sumTimerWriteExternal, r.sumTimerWait)),
		name)
}

func (r *Row) add(other Row) {
	r.sumTimerWait += other.sumTimerWait
	r.sumTimerRead += other.sumTimerRead
	r.sumTimerWrite += other.sumTimerWrite
	r.sumTimerReadWithSharedLocks += other.sumTimerReadWithSharedLocks
	r.sumTimerReadHighPriority += other.sumTimerReadHighPriority
	r.sumTimerReadNoInsert += other.sumTimerReadNoInsert
	r.sumTimerReadNormal += other.sumTimerReadNormal
	r.sumTimerReadExternal += other.sumTimerReadExternal
	r.sumTimerWriteAllowWrite += other.sumTimerWriteAllowWrite
	r.sumTimerWriteConcurrentInsert += other.sumTimerWriteConcurrentInsert
	r.sumTimerWriteLowPriority += other.sumTimerWriteLowPriority
	r.sumTimerWriteNormal += other.sumTimerWriteNormal
	r.sumTimerWriteExternal += other.sumTimerWriteExternal
}

func (r *Row) subtract(other Row) {
	r.sumTimerWait -= other.sumTimerWait
	r.sumTimerRead -= other.sumTimerRead
	r.sumTimerWrite -= other.sumTimerWrite
	r.sumTimerReadWithSharedLocks -= other.sumTimerReadWithSharedLocks
	r.sumTimerReadHighPriority -= other.sumTimerReadHighPriority
	r.sumTimerReadNoInsert -= other.sumTimerReadNoInsert
	r.sumTimerReadNormal -= other.sumTimerReadNormal
	r.sumTimerReadExternal -= other.sumTimerReadExternal
	r.sumTimerWriteAllowWrite -= other.sumTimerWriteAllowWrite
	r.sumTimerWriteConcurrentInsert -= other.sumTimerWriteConcurrentInsert
	r.sumTimerWriteLowPriority -= other.sumTimerWriteLowPriority
	r.sumTimerWriteNormal -= other.sumTimerWriteNormal
	r.sumTimerWriteExternal -= other.sumTimerWriteExternal
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
		lib.FormatTime(r.sumTimerWriteConcurrentInsert),
		lib.FormatTime(r.sumTimerWriteLowPriority),
		lib.FormatTime(r.sumTimerWriteNormal),
		lib.FormatTime(r.sumTimerWriteExternal),
		r.name)
}

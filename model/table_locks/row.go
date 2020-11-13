// Package table_locks contains the library
// routines for managing the table_lock_waits_summary_by_table table.
package table_locks

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
	Name                          string // combination of <schema>.<table>
	SumTimerWait                  uint64
	SumTimerRead                  uint64
	SumTimerWrite                 uint64
	SumTimerReadWithSharedLocks   uint64
	SumTimerReadHighPriority      uint64
	SumTimerReadNoInsert          uint64
	SumTimerReadNormal            uint64
	SumTimerReadExternal          uint64
	SumTimerWriteAllowWrite       uint64
	SumTimerWriteConcurrentInsert uint64
	SumTimerWriteLowPriority      uint64
	SumTimerWriteNormal           uint64
	SumTimerWriteExternal         uint64
}

func (r *Row) add(other Row) {
	r.SumTimerWait += other.SumTimerWait
	r.SumTimerRead += other.SumTimerRead
	r.SumTimerWrite += other.SumTimerWrite
	r.SumTimerReadWithSharedLocks += other.SumTimerReadWithSharedLocks
	r.SumTimerReadHighPriority += other.SumTimerReadHighPriority
	r.SumTimerReadNoInsert += other.SumTimerReadNoInsert
	r.SumTimerReadNormal += other.SumTimerReadNormal
	r.SumTimerReadExternal += other.SumTimerReadExternal
	r.SumTimerWriteAllowWrite += other.SumTimerWriteAllowWrite
	r.SumTimerWriteConcurrentInsert += other.SumTimerWriteConcurrentInsert
	r.SumTimerWriteLowPriority += other.SumTimerWriteLowPriority
	r.SumTimerWriteNormal += other.SumTimerWriteNormal
	r.SumTimerWriteExternal += other.SumTimerWriteExternal
}

func (r *Row) subtract(other Row) {
	r.SumTimerWait -= other.SumTimerWait
	r.SumTimerRead -= other.SumTimerRead
	r.SumTimerWrite -= other.SumTimerWrite
	r.SumTimerReadWithSharedLocks -= other.SumTimerReadWithSharedLocks
	r.SumTimerReadHighPriority -= other.SumTimerReadHighPriority
	r.SumTimerReadNoInsert -= other.SumTimerReadNoInsert
	r.SumTimerReadNormal -= other.SumTimerReadNormal
	r.SumTimerReadExternal -= other.SumTimerReadExternal
	r.SumTimerWriteAllowWrite -= other.SumTimerWriteAllowWrite
	r.SumTimerWriteConcurrentInsert -= other.SumTimerWriteConcurrentInsert
	r.SumTimerWriteLowPriority -= other.SumTimerWriteLowPriority
	r.SumTimerWriteNormal -= other.SumTimerWriteNormal
	r.SumTimerWriteExternal -= other.SumTimerWriteExternal
}

func (r *Row) HasData() bool {
	return r != nil && r.SumTimerWait > 0
}

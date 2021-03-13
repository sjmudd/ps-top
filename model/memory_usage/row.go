// Package memory_usage contains the library
// routines for managing memory_summary_global_by_event_name table.
package memory_usage

import (
	_ "github.com/go-sql-driver/mysql" // keep glint happy
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
	Name              string
	CurrentCountUsed  int64
	HighCountUsed     int64
	TotalMemoryOps    int64
	CurrentBytesUsed  int64
	HighBytesUsed     int64
	TotalBytesManaged uint64
}

func (r *Row) add(other Row) {
	r.CurrentBytesUsed += other.CurrentBytesUsed
	r.TotalMemoryOps += other.TotalMemoryOps
	r.CurrentCountUsed += other.CurrentCountUsed
}

// HasData returns true if there is valid data in the row
func (r *Row) HasData() bool {
	return r != nil && r.Name != "" && r.CurrentCountUsed != 0 && r.TotalMemoryOps != 0
}

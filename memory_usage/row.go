// Package memory_usage contains the library
// routines for managing memory_summary_global_by_event_name table.
package memory_usage

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql" // keep glint happy

	"github.com/sjmudd/ps-top/lib"
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

func (r *Row) headings() string {
	return fmt.Sprint("CurBytes         %  High Bytes|MemOps          %|CurAlloc       %  HiAlloc|Memory Area")
	//                         1234567890  100.0%  1234567890|123456789  100.0%|12345678  100.0%  12345678|Some memory name
}

// generate a printable result
func (r *Row) content(totals Row) string {

	// assume the data is empty so hide it.
	name := r.name
	if r.totalMemoryOps == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s  %6s  %10s|%10s %6s|%8s  %6s  %8s|%s",
		lib.SignedFormatAmount(r.currentBytesUsed),
		lib.FormatPct(lib.SignedDivide(r.currentBytesUsed, totals.currentBytesUsed)),
		lib.SignedFormatAmount(r.highBytesUsed),
		lib.SignedFormatAmount(r.totalMemoryOps),
		lib.FormatPct(lib.SignedDivide(r.totalMemoryOps, totals.totalMemoryOps)),
		lib.SignedFormatAmount(r.currentCountUsed),
		lib.FormatPct(lib.SignedDivide(r.currentCountUsed, totals.currentCountUsed)),
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

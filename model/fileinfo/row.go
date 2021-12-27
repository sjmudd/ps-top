// Package fileinfo contains the routines for
// managing the file_summary_by_instance table.
package fileinfo

import (
	"log"
)

/*
CREATE TABLE `file_summary_by_instance` (
  `FILE_NAME` varchar(512) NOT NULL,
  `EVENT_NAME` varchar(128) NOT NULL,				// not collected
  `OBJECT_INSTANCE_BEGIN` bigint(20) unsigned NOT NULL,		// not collected
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
  `SUM_NUMBER_OF_BYTES_READ` bigint(20) NOT NULL,
  `COUNT_WRITE` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_WRITE` bigint(20) unsigned NOT NULL,
  `SUM_NUMBER_OF_BYTES_WRITE` bigint(20) NOT NULL,
  `COUNT_MISC` bigint(20) unsigned NOT NULL,
  `SUM_TIMER_MISC` bigint(20) unsigned NOT NULL,
  `MIN_TIMER_MISC` bigint(20) unsigned NOT NULL,
  `AVG_TIMER_MISC` bigint(20) unsigned NOT NULL,
  `MAX_TIMER_MISC` bigint(20) unsigned NOT NULL
) ENGINE=PERFORMANCE_SCHEMA DEFAULT CHARSET=utf8
1 row in set (0.00 sec)
*/

// Row contains a row from file_summary_by_instance
type Row struct {
	Name                  string
	CountStar             uint64
	CountRead             uint64
	CountWrite            uint64
	CountMisc             uint64
	SumTimerWait          uint64
	SumTimerRead          uint64
	SumTimerWrite         uint64
	SumTimerMisc          uint64
	SumNumberOfBytesRead  uint64
	SumNumberOfBytesWrite uint64
}

// duplicateSlice copies the full slice
func duplicateSlice(slice []Row) []Row {
	return append(make([]Row, len(slice)), slice...)
}

// Valid checks if the row is valid and if asked to do so logs the problem
func (row Row) Valid(logProblem bool) bool {
	var problem bool
	if (row.CountStar < row.CountRead) ||
		(row.CountStar < row.CountWrite) ||
		(row.CountStar < row.CountMisc) {
		problem = true
		if logProblem {
			log.Println("Row.Valid() FAILED (count)", row)
		}
	}
	if (row.SumTimerWait < row.SumTimerRead) ||
		(row.SumTimerWait < row.SumTimerWrite) ||
		(row.SumTimerWait < row.SumTimerMisc) {
		problem = true
		if logProblem {
			log.Println("Row.Valid() FAILED (sumTimer)", row)
		}
	}
	return problem
}

// Add rows together, keeping the name of first row
func add(row, other Row) Row {
	return Row{
		Name:       row.Name,
		CountStar:  row.CountStar + other.CountStar,
		CountRead:  row.CountRead + other.CountRead,
		CountWrite: row.CountWrite + other.CountWrite,
		CountMisc:  row.CountMisc + other.CountMisc,

		SumTimerWait:  row.SumTimerWait + other.SumTimerWait,
		SumTimerRead:  row.SumTimerRead + other.SumTimerRead,
		SumTimerWrite: row.SumTimerWrite + other.SumTimerWrite,
		SumTimerMisc:  row.SumTimerMisc + other.SumTimerMisc,

		SumNumberOfBytesRead:  row.SumNumberOfBytesRead + other.SumNumberOfBytesRead,
		SumNumberOfBytesWrite: row.SumNumberOfBytesWrite + other.SumNumberOfBytesWrite,
	}
}

// sometimes the values can drop and catch us out. This routines hides that by returning 0
func validSubtract(this, that uint64) uint64 {
	if this > that {
		return this - that
	}
	return 0
}

// subtract one set of values from another one keeping the original row name
// - use validSubtract() to catch negative jumps which do happen from time to time.
func subtract(row, other Row) Row {
	newRow := row

	newRow.CountStar = validSubtract(row.CountStar, other.CountStar)
	newRow.CountRead = validSubtract(row.CountRead, other.CountRead)
	newRow.CountWrite = validSubtract(row.CountWrite, other.CountWrite)
	newRow.CountMisc = validSubtract(row.CountMisc, other.CountMisc)

	newRow.SumTimerWait = validSubtract(row.SumTimerWait, other.SumTimerWait)
	newRow.SumTimerRead = validSubtract(row.SumTimerRead, other.SumTimerRead)
	newRow.SumTimerWrite = validSubtract(row.SumTimerWrite, other.SumTimerWrite)
	newRow.SumTimerMisc = validSubtract(row.SumTimerMisc, other.SumTimerMisc)

	newRow.SumNumberOfBytesRead = validSubtract(row.SumNumberOfBytesRead, other.SumNumberOfBytesRead)
	newRow.SumNumberOfBytesWrite = validSubtract(row.SumNumberOfBytesWrite, other.SumNumberOfBytesWrite)

	return newRow
}

// HasData indicates if there is data in the row (for counting valid rows)
func (row *Row) HasData() bool {
	return row != nil && row.SumTimerWait > 0
}

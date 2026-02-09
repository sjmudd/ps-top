// Package tablelocklatency holds the routines which manage the file_summary_by_instance table.
package tablelocklatency

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/tablelocks"
	"github.com/sjmudd/ps-top/wrapper"
)

// Wrapper wraps a TableLockLatency struct
type Wrapper struct {
	tl *tablelocks.TableLocks
}

// NewTableLockLatency creates a wrapper around TableLockLatency
func NewTableLockLatency(cfg *config.Config, db *sql.DB) *Wrapper {
	return &Wrapper{
		tl: tablelocks.NewTableLocks(cfg, db),
	}
}

// ResetStatistics resets the statistics to last values
func (tlw *Wrapper) ResetStatistics() {
	tlw.tl.ResetStatistics()
}

// Collect data from the db, then merge it in.
func (tlw *Wrapper) Collect() {
	tlw.tl.Collect()
	sort.Slice(tlw.tl.Results, func(i, j int) bool {
		return (tlw.tl.Results[i].SumTimerWait > tlw.tl.Results[j].SumTimerWait) ||
			((tlw.tl.Results[i].SumTimerWait == tlw.tl.Results[j].SumTimerWait) &&
				(tlw.tl.Results[i].Name < tlw.tl.Results[j].Name))
	})
}

// Headings returns the headings for a table
func (tlw Wrapper) Headings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s|%-30s",
		"Latency", "%",
		"Read", "Write",
		"S.Lock", "High", "NoIns", "Normal", "Extrnl",
		"AlloWr", "CncIns", "Low", "Normal", "Extrnl",
		"Table Name")
}

// RowContent returns the rows we need for displaying
func (tlw Wrapper) RowContent() []string {
	n := len(tlw.tl.Results)
	return wrapper.RowsFromGetter(n, func(i int) string {
		return tlw.content(tlw.tl.Results[i], tlw.tl.Totals)
	})
}

// TotalRowContent returns all the totals
func (tlw Wrapper) TotalRowContent() string {
	return wrapper.TotalRowContent(tlw.tl.Totals, tlw.content)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (tlw Wrapper) EmptyRowContent() string {
	return wrapper.EmptyRowContent(tlw.content)
}

// Description returns a description of the table
func (tlw Wrapper) Description() string {
	count := len(tlw.tl.Results)

	return fmt.Sprintf("Locks by Table Name (table_lock_waits_summary_by_table) %d rows", count)
}

// HaveRelativeStats is true for this object
func (tlw Wrapper) HaveRelativeStats() bool {
	return tlw.tl.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (tlw Wrapper) FirstCollectTime() time.Time {
	return tlw.tl.FirstCollected
}

// LastCollectTime returns the time the last value was collected
func (tlw Wrapper) LastCollectTime() time.Time {
	return tlw.tl.LastCollected
}

// WantRelativeStats indicates if we want relative statistics
func (tlw Wrapper) WantRelativeStats() bool {
	return tlw.tl.WantRelativeStats()
}

// content generate a printable result for a row, given the totals
func (tlw Wrapper) content(row, totals tablelocks.Row) string {
	// assume the data is empty so hide it.
	name := row.Name
	if row.SumTimerWait == 0 && name != "Totals" {
		name = ""
	}
	timeStr, pctStr := wrapper.TimePct(row.SumTimerWait, totals.SumTimerWait)
	pct := wrapper.PctStrings(row.SumTimerWait,
		row.SumTimerRead,
		row.SumTimerWrite,
		row.SumTimerReadWithSharedLocks,
		row.SumTimerReadHighPriority,
		row.SumTimerReadNoInsert,
		row.SumTimerReadNormal,
		row.SumTimerReadExternal,
		row.SumTimerWriteAllowWrite,
		row.SumTimerWriteConcurrentInsert,
		row.SumTimerWriteLowPriority,
		row.SumTimerWriteNormal,
		row.SumTimerWriteExternal)

	return fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s|%s",
		timeStr,
		pctStr,

		pct[0],
		pct[1],

		pct[2],
		pct[3],
		pct[4],
		pct[5],
		pct[6],

		pct[7],
		pct[8],
		pct[9],
		pct[10],
		pct[11],
		name)
}

// sorting handled inline with sort.Slice to avoid repeated boilerplate types

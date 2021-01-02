// Package table_lock_latency holds the routines which manage the file_summary_by_instance table.
package table_lock_latency

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/model/table_locks"
)

// Wrapper wraps a TableLockLatency struct
type Wrapper struct {
	tl *table_locks.TableLocks
}

// NewTableLockLatency creates a wrapper around TableLockLatency
func NewTableLockLatency(ctx *context.Context, db *sql.DB) *Wrapper {
	return &Wrapper{
		tl: table_locks.NewTableLocks(ctx, db),
	}
}

// SetFirstFromLast resets the statistics to last values
func (tlw *Wrapper) SetFirstFromLast() {
	tlw.tl.SetFirstFromLast()
}

// Collect data from the db, then merge it in.
func (tlw *Wrapper) Collect() {
	tlw.tl.Collect()
	sort.Sort(byLatency(tlw.tl.Results))
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
	rows := make([]string, 0, len(tlw.tl.Results))

	for i := range tlw.tl.Results {
		rows = append(rows, tlw.content(tlw.tl.Results[i], tlw.tl.Totals))
	}

	return rows
}

// TotalRowContent returns all the totals
func (tlw Wrapper) TotalRowContent() string {
	return tlw.content(tlw.tl.Totals, tlw.tl.Totals)
}

// Len return the length of the result set
func (tlw Wrapper) Len() int {
	return len(tlw.tl.Results)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (tlw Wrapper) EmptyRowContent() string {
	var empty table_locks.Row

	return tlw.content(empty, empty)
}

// Description returns a description of the table
func (tlw Wrapper) Description() string {
	return "Locks by Table Name (table_lock_waits_summary_by_table)"
}

// HaveRelativeStats is true for this object
func (tlw Wrapper) HaveRelativeStats() bool {
	return tlw.tl.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (tlw Wrapper) FirstCollectTime() time.Time {
	return tlw.tl.FirstCollectTime()
}

// LastCollectTime returns the time the last value was collected
func (tlw Wrapper) LastCollectTime() time.Time {
	return tlw.tl.LastCollectTime()
}

// WantRelativeStats indiates if we want relative statistics
func (tlw Wrapper) WantRelativeStats() bool {
	return tlw.tl.WantRelativeStats()
}

// content generate a printable result for a row, given the totals
func (tlw Wrapper) content(row, totals table_locks.Row) string {
	// assume the data is empty so hide it.
	name := row.Name
	if row.SumTimerWait == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s|%s",
		lib.FormatTime(row.SumTimerWait),
		lib.FormatPct(lib.Divide(row.SumTimerWait, totals.SumTimerWait)),

		lib.FormatPct(lib.Divide(row.SumTimerRead, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerWrite, row.SumTimerWait)),

		lib.FormatPct(lib.Divide(row.SumTimerReadWithSharedLocks, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerReadHighPriority, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerReadNoInsert, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerReadNormal, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerReadExternal, row.SumTimerWait)),

		lib.FormatPct(lib.Divide(row.SumTimerWriteAllowWrite, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerWriteConcurrentInsert, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerWriteLowPriority, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerWriteNormal, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerWriteExternal, row.SumTimerWait)),
		name)
}

type byLatency table_locks.Rows

func (t byLatency) Len() int      { return len(t) }
func (t byLatency) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t byLatency) Less(i, j int) bool {
	return (t[i].SumTimerWait > t[j].SumTimerWait) ||
		((t[i].SumTimerWait == t[j].SumTimerWait) &&
			(t[i].Name < t[j].Name))

}

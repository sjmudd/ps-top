// Package memory_usage holds the routines which manage the file_summary_by_instance table.
package memory_usage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/model/memory_usage"
)

// Wrapper wraps a FileIoLatency struct  representing the contents of the data collected from file_summary_by_instance, but adding formatting for presentation in the terminal
type Wrapper struct {
	mu *memory_usage.MemoryUsage
}

// NewMemoryUsage creates a wrapper around MemoryUsage
func NewMemoryUsage(ctx *context.Context, db *sql.DB) *Wrapper {
	return &Wrapper{
		mu: memory_usage.NewMemoryUsage(ctx, db),
	}
}

// SetFirstFromLast resets the statistics to last values
func (muw *Wrapper) SetFirstFromLast() {
	muw.mu.SetFirstFromLast()
}

// Collect data from the db, then merge it in.
func (muw *Wrapper) Collect() {
	muw.mu.Collect()
}

// Headings returns the headings for a table
func (muw Wrapper) Headings() string {
	return fmt.Sprint("CurBytes         %  High Bytes|MemOps          %|CurAlloc       %  HiAlloc|Memory Area")
	//                         1234567890  100.0%  1234567890|123456789  100.0%|12345678  100.0%  12345678|Some memory name
}

// RowContent returns the rows we need for displaying
func (muw Wrapper) RowContent() []string {
	rows := make([]string, 0, len(muw.mu.Results))

	for i := range muw.mu.Results {
		rows = append(rows, muw.content(muw.mu.Results[i], muw.mu.Totals))
	}

	return rows
}

// TotalRowContent returns all the totals
func (muw Wrapper) TotalRowContent() string {
	return muw.content(muw.mu.Totals, muw.mu.Totals)
}

// Len return the length of the result set
func (muw Wrapper) Len() int {
	return len(muw.mu.Results)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (muw Wrapper) EmptyRowContent() string {
	var empty memory_usage.Row

	return muw.content(empty, empty)
}

// Description returns a description of the table
func (muw Wrapper) Description() string {
	var count int

	for row := range muw.mu.Results {
		if muw.mu.Results[row].HasData() {
			count++
		}
	}

	return fmt.Sprintf("File I/O Latency (file_summary_by_instance) %4d row(s)    ", count)
}

// HaveRelativeStats is true for this object
func (muw Wrapper) HaveRelativeStats() bool {
	return muw.mu.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (muw Wrapper) FirstCollectTime() time.Time {
	return muw.mu.FirstCollectTime()
}

// LastCollectTime returns the time the last value was collected
func (muw Wrapper) LastCollectTime() time.Time {
	return muw.mu.LastCollectTime()
}

// WantRelativeStats indiates if we want relative statistics
func (muw Wrapper) WantRelativeStats() bool {
	return muw.mu.WantRelativeStats()
}

// content generate a printable result for a row, given the totals
func (muw Wrapper) content(row, totals memory_usage.Row) string {
	// assume the data is empty so hide it.
	name := row.Name
	if row.TotalMemoryOps == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s  %6s  %10s|%10s %6s|%8s  %6s  %8s|%s",
		lib.SignedFormatAmount(row.CurrentBytesUsed),
		lib.FormatPct(lib.SignedDivide(row.CurrentBytesUsed, totals.CurrentBytesUsed)),
		lib.SignedFormatAmount(row.HighBytesUsed),
		lib.SignedFormatAmount(row.TotalMemoryOps),
		lib.FormatPct(lib.SignedDivide(row.TotalMemoryOps, totals.TotalMemoryOps)),
		lib.SignedFormatAmount(row.CurrentCountUsed),
		lib.FormatPct(lib.SignedDivide(row.CurrentCountUsed, totals.CurrentCountUsed)),
		lib.SignedFormatAmount(row.HighCountUsed),
		name)
}

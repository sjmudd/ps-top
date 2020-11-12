// Package file_io_latency holds the routines which manage the file_summary_by_instance table.
package file_io_latency

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/file_io_latency"
	"github.com/sjmudd/ps-top/lib"
)

// Wrapper wraps a FileIoLatency struct  representing the contents of the data collected from file_summary_by_instance, but adding formatting for presentation in the terminal
type Wrapper struct {
	fiol *file_io_latency.FileIoLatency
}

// NewFileSummaryByInstance creates a wrapper around FileIoLatency
func NewFileSummaryByInstance(ctx *context.Context, db *sql.DB) *Wrapper {
	return &Wrapper{
		fiol: file_io_latency.NewFileSummaryByInstance(ctx, db),
	}
}

// SetFirstFromLast resets the statistics to last values
func (fiolw *Wrapper) SetFirstFromLast() {
	fiolw.fiol.SetFirstFromLast()
}

// Collect data from the db, then merge it in.
func (fiolw *Wrapper) Collect() {
	fiolw.fiol.Collect()
}

// Headings returns the headings for a table
func (fiolw Wrapper) Headings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s %6s|%8s %8s|%8s %6s %6s %6s|%s",
		"Latency",
		"%",
		"Read",
		"Write",
		"Misc",
		"Rd bytes",
		"Wr bytes",
		"Ops",
		"R Ops",
		"W Ops",
		"M Ops",
		"Table Name")
}

// RowContent returns the rows we need for displaying
func (fiolw Wrapper) RowContent() []string {
	rows := make([]string, 0, len(fiolw.fiol.Results))

	for i := range fiolw.fiol.Results {
		rows = append(rows, fiolw.content(fiolw.fiol.Results[i], fiolw.fiol.Totals))
	}

	return rows
}

// TotalRowContent returns all the totals
func (fiolw Wrapper) TotalRowContent() string {
	return fiolw.content(fiolw.fiol.Totals, fiolw.fiol.Totals)
}

// Len return the length of the result set
func (fiolw Wrapper) Len() int {
	return len(fiolw.fiol.Results)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (fiolw Wrapper) EmptyRowContent() string {
	var empty file_io_latency.Row

	return fiolw.content(empty, empty)
}

// Description returns a description of the table
func (fiolw Wrapper) Description() string {
	var count int

	for row := range fiolw.fiol.Results {
		if fiolw.fiol.Results[row].HasData() {
			count++
		}
	}

	return fmt.Sprintf("File I/O Latency (file_summary_by_instance) %4d row(s)    ", count)
}

// HaveRelativeStats is true for this object
func (fiolw Wrapper) HaveRelativeStats() bool {
	return fiolw.fiol.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (fiolw Wrapper) FirstCollectTime() time.Time {
	return fiolw.fiol.FirstCollectTime()
}

// LastCollectTime returns the time the last value was collected
func (fiolw Wrapper) LastCollectTime() time.Time {
	return fiolw.fiol.LastCollectTime()
}

// WantRelativeStats indiates if we want relative statistics
func (fiolw Wrapper) WantRelativeStats() bool {
	return fiolw.fiol.WantRelativeStats()
}

// content generate a printable result for a row, given the totals
func (fiolw Wrapper) content(row, totals file_io_latency.Row) string {
	var name = row.Name

	// We assume that if CountStar = 0 then there's no data at all...
	// when we have no data we really don't want to show the name either.
	if (row.SumTimerWait == 0 && row.CountStar == 0 && row.SumNumberOfBytesRead == 0 && row.SumNumberOfBytesWrite == 0) && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s %6s|%8s %8s|%8s %6s %6s %6s|%s",
		lib.FormatTime(row.SumTimerWait),
		lib.FormatPct(lib.Divide(row.SumTimerWait, totals.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerRead, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerWrite, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerMisc, row.SumTimerWait)),
		lib.FormatAmount(row.SumNumberOfBytesRead),
		lib.FormatAmount(row.SumNumberOfBytesWrite),
		lib.FormatAmount(row.CountStar),
		lib.FormatPct(lib.Divide(row.CountRead, row.CountStar)),
		lib.FormatPct(lib.Divide(row.CountWrite, row.CountStar)),
		lib.FormatPct(lib.Divide(row.CountMisc, row.CountStar)),
		name)
}

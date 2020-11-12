// Package file_io_latency holds the routines which manage the file_summary_by_instance table.
package file_io_latency

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/table_io_latency"
)

// FileIoLatency represents the contents of the data collected from file_summary_by_instance
type WrapperLatency struct {
	tiol *table_io_latency.TableIoLatency
}

// NewFileSummaryByInstance creates a wrapper around FileIoLatency
func NewTableIoLatency(ctx *context.Context, db *sql.DB) *WrapperLatency {
	return &WrapperLatency{
		tiol: table_io_latency.NewTableIoLatency(ctx, db),
	}
}

// SetFirstFromLast resets the statistics to last values
func (tiolw *WrapperLatency) SetFirstFromLast() {
	tiolw.tiol.SetFirstFromLast()
}

// Collect data from the db, then merge it in.
func (tiolw *WrapperLatency) Collect() {
	tiolw.tiol.Collect()
}

// Headings returns the latency headings as a string
func (tiolw WrapperLatency) Headings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s",
		"Latency",
		"%",
		"Fetch",
		"Insert",
		"Update",
		"Delete",
		"Table Name")
}

// RowContent returns the rows we need for displaying
func (tiolw WrapperLatency) RowContent() []string {
	rows := make([]string, 0, len(tiolw.tiol.Results))

	for i := range tiolw.tiol.Results {
		rows = append(rows, tiolw.content(tiolw.tiol.Results[i], tiolw.tiol.Totals))
	}

	return rows
}

// Len return the length of the result set
func (tiolw WrapperLatency) Len() int {
	return len(tiolw.tiol.Results)
}

// TotalRowContent returns all the totals
func (tiolw WrapperLatency) TotalRowContent() string {
	return tiolw.content(tiolw.tiol.Totals, tiolw.tiol.Totals)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (tiolw WrapperLatency) EmptyRowContent() string {
	var empty table_io_latency.Row

	return tiolw.content(empty, empty)
}

// Description returns a description of the table
func (tiolw WrapperLatency) Description() string {
	var count int
	for row := range tiolw.tiol.Results {
		if tiolw.tiol.Results[row].HasData() {
			count++
		}
	}

	return fmt.Sprintf("Table Latency (table_io_waits_summary_by_table) %d rows", count)
}

// HaveRelativeStats is true for this object
func (tiolw WrapperLatency) HaveRelativeStats() bool {
	return tiolw.tiol.HaveRelativeStats()
}

// FirstCollectTime
func (tiolw WrapperLatency) FirstCollectTime() time.Time {
	return tiolw.tiol.FirstCollectTime()
}

// LastCollectTime
func (tiolw WrapperLatency) LastCollectTime() time.Time {
	return tiolw.tiol.LastCollectTime()
}

func (tiolw WrapperLatency) WantRelativeStats() bool {
	return tiolw.tiol.WantRelativeStats()
}

// get rid of me as I should not be ncessary
func (tiolw WrapperLatency) SetWantsLatency(wants bool) {
	tiolw.tiol.SetWantsLatency(wants)
}

// latencyRowContents reutrns the printable result
func (tiolw WrapperLatency) content(row, totals table_io_latency.Row) string {
	// assume the data is empty so hide it.
	name := row.Name
	if row.CountStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s",
		lib.FormatTime(row.SumTimerWait),
		lib.FormatPct(lib.Divide(row.SumTimerWait, totals.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerFetch, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerInsert, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerUpdate, row.SumTimerWait)),
		lib.FormatPct(lib.Divide(row.SumTimerDelete, row.SumTimerWait)),
		name)
}

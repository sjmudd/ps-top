// Package table_io_latency holds the routines which manage the table_io statisticss.
package table_io_latency

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/model/table_io"
)

// FileIoLatency represents the contents of the data collected from file_summary_by_instance
type Wrapper struct {
	tiol *table_io.TableIo
}

// NewFileSummaryByInstance creates a wrapper around FileIoLatency
func NewTableIoLatency(ctx *context.Context, db *sql.DB) *Wrapper {
	return &Wrapper{
		tiol: table_io.NewTableIo(ctx, db),
	}
}

func (tiolw *Wrapper) Tiol() *table_io.TableIo {
	return tiolw.tiol
}

// SetFirstFromLast resets the statistics to last values
func (tiolw *Wrapper) SetFirstFromLast() {
	tiolw.tiol.SetFirstFromLast()
}

// Collect data from the db, then merge it in.
func (tiolw *Wrapper) Collect() {
	tiolw.tiol.Collect()
}

// Headings returns the latency headings as a string
func (tiolw Wrapper) Headings() string {
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
func (tiolw Wrapper) RowContent() []string {
	rows := make([]string, 0, len(tiolw.tiol.Results))

	for i := range tiolw.tiol.Results {
		rows = append(rows, tiolw.content(tiolw.tiol.Results[i], tiolw.tiol.Totals))
	}

	return rows
}

// Len return the length of the result set
func (tiolw Wrapper) Len() int {
	return len(tiolw.tiol.Results)
}

// TotalRowContent returns all the totals
func (tiolw Wrapper) TotalRowContent() string {
	return tiolw.content(tiolw.tiol.Totals, tiolw.tiol.Totals)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (tiolw Wrapper) EmptyRowContent() string {
	var empty table_io.Row

	return tiolw.content(empty, empty)
}

// Description returns a description of the table
func (tiolw Wrapper) Description() string {
	var count int
	for row := range tiolw.tiol.Results {
		if tiolw.tiol.Results[row].HasData() {
			count++
		}
	}

	return fmt.Sprintf("Table Latency (table_io_waits_summary_by_table) %d rows", count)
}

// HaveRelativeStats is true for this object
func (tiolw Wrapper) HaveRelativeStats() bool {
	return tiolw.tiol.HaveRelativeStats()
}

// FirstCollectTime
func (tiolw Wrapper) FirstCollectTime() time.Time {
	return tiolw.tiol.FirstCollectTime()
}

// LastCollectTime
func (tiolw Wrapper) LastCollectTime() time.Time {
	return tiolw.tiol.LastCollectTime()
}

func (tiolw Wrapper) WantRelativeStats() bool {
	return tiolw.tiol.WantRelativeStats()
}

// get rid of me as I should not be ncessary
func (tiolw Wrapper) SetWantsLatency(wants bool) {
	tiolw.tiol.SetWantsLatency(wants)
}

// latencyRowContents reutrns the printable result
func (tiolw Wrapper) content(row, totals table_io.Row) string {
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

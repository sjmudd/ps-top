// Package tableiolatency holds the routines which manage the tableio statisticss.
package tableiolatency

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/model/tableio"
)

// Wrapper represents the contents of the data collected related to tableio statistics
type Wrapper struct {
	tiol *tableio.TableIo
}

// NewTableIoLatency creates a wrapper around tableio statistics
func NewTableIoLatency(ctx *context.Context, db *sql.DB) *Wrapper {
	return &Wrapper{
		tiol: tableio.NewTableIo(ctx, db),
	}
}

// Tiol returns the a TableIo value
func (tiolw *Wrapper) Tiol() *tableio.TableIo {
	return tiolw.tiol
}

// ResetStatistics resets the statistics to last values
func (tiolw *Wrapper) ResetStatistics() {
	tiolw.tiol.ResetStatistics()
}

// Collect data from the db, then merge it in.
func (tiolw *Wrapper) Collect() {
	tiolw.tiol.Collect()

	// sort the results by latency (might be needed in other places)
	sort.Sort(byLatency(tiolw.tiol.Results))
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
	var empty tableio.Row

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

// FirstCollectTime returns the time of the first collection
func (tiolw Wrapper) FirstCollectTime() time.Time {
	return tiolw.tiol.FirstCollectTime()
}

// LastCollectTime returns the time of the last collection
func (tiolw Wrapper) LastCollectTime() time.Time {
	return tiolw.tiol.LastCollectTime()
}

// WantRelativeStats returns if we want to see relative stats
func (tiolw Wrapper) WantRelativeStats() bool {
	return tiolw.tiol.WantRelativeStats()
}

// latencyRowContents reutrns the printable result
func (tiolw Wrapper) content(row, totals tableio.Row) string {
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

// for sorting
type byLatency tableio.Rows

// sort the tableio.Rows by latency
func (rows byLatency) Len() int      { return len(rows) }
func (rows byLatency) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }

// sort by value (descending) but also by "name" (ascending) if the values are the same
func (rows byLatency) Less(i, j int) bool {
	return (rows[i].SumTimerWait > rows[j].SumTimerWait) ||
		((rows[i].SumTimerWait == rows[j].SumTimerWait) &&
			(rows[i].Name < rows[j].Name))
}

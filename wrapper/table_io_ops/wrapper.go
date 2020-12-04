// Package table_io_ops holds the routines which manage the table ops
package table_io_ops

import (
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/model/table_io"
	"github.com/sjmudd/ps-top/wrapper/table_io_latency"
)

// FileIoLatency represents a wrapper around table_io
type Wrapper struct {
	tiol *table_io.TableIo
}

// NewTableIoOps creates a wrapper around TableIo, sharing the same connection with the table_io_latency wrapper
func NewTableIoOps(latency *table_io_latency.Wrapper) *Wrapper {
	return &Wrapper{
		tiol: latency.Tiol(),
	}
}

// SetFirstFromLast resets the statistics to last values
func (tiolw *Wrapper) SetFirstFromLast() {
	tiolw.tiol.SetFirstFromLast()
}

// Collect data from the db, then merge it in.
func (tiolw *Wrapper) Collect() {
	tiolw.tiol.Collect()

	// sort the results by ops
	sort.Sort(ByOps(tiolw.tiol.Results))
}

// Headings returns the headings by operations as a string
func (tiolw Wrapper) Headings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s",
		"Ops",
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

	return fmt.Sprintf("Table Ops (table_io_waits_summary_by_table) %d rows", count)
}

// HaveRelativeStats is true for this object
func (tiolw Wrapper) HaveRelativeStats() bool {
	return true
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

// generate a printable result for ops
func (tiolw Wrapper) content(row, totals table_io.Row) string {
	// assume the data is empty so hide it.
	name := row.Name
	if row.CountStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s",
		lib.FormatAmount(row.CountStar),
		lib.FormatPct(lib.Divide(row.CountStar, totals.CountStar)),
		lib.FormatPct(lib.Divide(row.CountFetch, row.CountStar)),
		lib.FormatPct(lib.Divide(row.CountInsert, row.CountStar)),
		lib.FormatPct(lib.Divide(row.CountUpdate, row.CountStar)),
		lib.FormatPct(lib.Divide(row.CountDelete, row.CountStar)),
		name)
}

// ByOps is used for sorting by the number of operations
type ByOps table_io.Rows

func (rows ByOps) Len() int      { return len(rows) }
func (rows ByOps) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }
func (rows ByOps) Less(i, j int) bool {
	return (rows[i].CountStar > rows[j].CountStar) ||
		((rows[i].SumTimerWait == rows[j].SumTimerWait) &&
			(rows[i].Name < rows[j].Name))
}

// Package tableioops holds the routines which manage the table ops
package tableioops

import (
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/model/tableio"
	"github.com/sjmudd/ps-top/wrapper/tableiolatency"
)

// Wrapper represents a wrapper around tableiolatency
type Wrapper struct {
	tiol *tableio.TableIo
}

// NewTableIoOps creates a wrapper around TableIo, sharing the same connection with the tableiolatency wrapper
func NewTableIoOps(latency *tableiolatency.Wrapper) *Wrapper {
	return &Wrapper{
		tiol: latency.Tiol(),
	}
}

// ResetStatistics resets the statistics to last values
func (tiolw *Wrapper) ResetStatistics() {
	tiolw.tiol.ResetStatistics()
}

// Collect data from the db, then merge it in.
func (tiolw *Wrapper) Collect() {
	tiolw.tiol.Collect()

	// sort the results by ops
	sort.Sort(byOperations(tiolw.tiol.Results))
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

	return fmt.Sprintf("Table Ops (table_io_waits_summary_by_table) %d rows", count)
}

// HaveRelativeStats is true for this object
func (tiolw Wrapper) HaveRelativeStats() bool {
	return true
}

// FirstCollectTime returns the time of the first collection of information
func (tiolw Wrapper) FirstCollectTime() time.Time {
	return tiolw.tiol.FirstCollected
}

// LastCollectTime returns the last time data was collected
func (tiolw Wrapper) LastCollectTime() time.Time {
	return tiolw.tiol.LastCollected
}

// WantRelativeStats returns whether we want to see relative or absolute stats
func (tiolw Wrapper) WantRelativeStats() bool {
	return tiolw.tiol.WantRelativeStats()
}

// generate a printable result for ops
func (tiolw Wrapper) content(row, totals tableio.Row) string {
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

// byOperations is used for sorting by the number of operations
type byOperations tableio.Rows

func (rows byOperations) Len() int      { return len(rows) }
func (rows byOperations) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }
func (rows byOperations) Less(i, j int) bool {
	return (rows[i].CountStar > rows[j].CountStar) ||
		((rows[i].SumTimerWait == rows[j].SumTimerWait) &&
			(rows[i].Name < rows[j].Name))
}

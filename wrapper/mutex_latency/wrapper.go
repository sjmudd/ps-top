// Package mutex_latency holds the routines which manage the server mutexs
package mutex_latency

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/model/mutex_latency"
)

// Wrapper wraps a MutexLatency struct
type Wrapper struct {
	ml *mutex_latency.MutexLatency
}

// NewMutexLatency creates a wrapper around mutex_latency.MutexLatency
func NewMutexLatency(ctx *context.Context, db *sql.DB) *Wrapper {
	return &Wrapper{
		ml: mutex_latency.NewMutexLatency(ctx, db),
	}
}

// SetFirstFromLast resets the statistics to last values
func (mlw *Wrapper) SetFirstFromLast() {
	mlw.ml.SetFirstFromLast()
}

// Collect data from the db, then merge it in.
func (mlw *Wrapper) Collect() {
	mlw.ml.Collect()
	sort.Sort(ByValue(mlw.ml.Results))
}

// RowContent returns the rows we need for displaying
func (mlw Wrapper) RowContent() []string {
	rows := make([]string, 0, len(mlw.ml.Results))

	for i := range mlw.ml.Results {
		rows = append(rows, mlw.content(mlw.ml.Results[i], mlw.ml.Totals))
	}

	return rows
}

// TotalRowContent returns all the totals
func (mlw Wrapper) TotalRowContent() string {
	return mlw.content(mlw.ml.Totals, mlw.ml.Totals)
}

// Len return the length of the result set
func (mlw Wrapper) Len() int {
	return len(mlw.ml.Results)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (mlw Wrapper) EmptyRowContent() string {
	var empty mutex_latency.Row

	return mlw.content(empty, empty)
}

// HaveRelativeStats is true for this object
func (mlw Wrapper) HaveRelativeStats() bool {
	return mlw.ml.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (mlw Wrapper) FirstCollectTime() time.Time {
	return mlw.ml.FirstCollectTime()
}

// LastCollectTime returns the time the last value was collected
func (mlw Wrapper) LastCollectTime() time.Time {
	return mlw.ml.LastCollectTime()
}

// WantRelativeStats indiates if we want relative statistics
func (mlw Wrapper) WantRelativeStats() bool {
	return mlw.ml.WantRelativeStats()
}

// Description returns a description of the table
func (mlw Wrapper) Description() string {
	var count int
	for row := range mlw.ml.Results {
		if mlw.ml.Results[row].SumTimerWait > 0 {
			count++
		}
	}
	return fmt.Sprintf("Mutex Latency (events_waits_summary_global_by_event_name) %d rows", count)
}

// Headings returns the headings for a table
func (mlw Wrapper) Headings() string {
	return fmt.Sprintf("%10s %8s %8s|%s", "Latency", "MtxCnt", "%", "Mutex Name")
}

// content generate a printable result for a row, given the totals
func (mlw Wrapper) content(row, totals mutex_latency.Row) string {
	name := row.Name
	if row.CountStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %8s %8s|%s",
		lib.FormatTime(row.SumTimerWait),
		lib.FormatAmount(row.CountStar),
		lib.FormatPct(lib.Divide(row.SumTimerWait, totals.SumTimerWait)),
		name)
}

type ByValue mutex_latency.Rows

func (rows ByValue) Len() int      { return len(rows) }
func (rows ByValue) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }

// sort by value (descending) but also by "name" (ascending) if the values are the same
func (rows ByValue) Less(i, j int) bool {
	return rows[i].SumTimerWait > rows[j].SumTimerWait
}

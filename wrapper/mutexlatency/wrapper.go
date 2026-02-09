// Package mutexlatency holds the routines which manage the server mutexes
package mutexlatency

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/mutexlatency"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/wrapper"
)

// Wrapper wraps a MutexLatency struct
type Wrapper struct {
	ml *mutexlatency.MutexLatency
}

// NewMutexLatency creates a wrapper around mutexlatency.MutexLatency
func NewMutexLatency(cfg *config.Config, db *sql.DB) *Wrapper {
	return &Wrapper{
		ml: mutexlatency.NewMutexLatency(cfg, db),
	}
}

// ResetStatistics resets the statistics to last values
func (mlw *Wrapper) ResetStatistics() {
	mlw.ml.ResetStatistics()
}

// Collect data from the db, then merge it in.
func (mlw *Wrapper) Collect() {
	mlw.ml.Collect()
	sort.Slice(mlw.ml.Results, func(i, j int) bool {
		return mlw.ml.Results[i].SumTimerWait > mlw.ml.Results[j].SumTimerWait
	})
}

// RowContent returns the rows we need for displaying
func (mlw Wrapper) RowContent() []string {
	n := len(mlw.ml.Results)
	return wrapper.RowsFromGetter(n, func(i int) string {
		return mlw.content(mlw.ml.Results[i], mlw.ml.Totals)
	})
}

// TotalRowContent returns all the totals
func (mlw Wrapper) TotalRowContent() string {
	return wrapper.TotalRowContent(mlw.ml.Totals, mlw.content)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (mlw Wrapper) EmptyRowContent() string {
	return wrapper.EmptyRowContent(mlw.content)
}

// HaveRelativeStats is true for this object
func (mlw Wrapper) HaveRelativeStats() bool {
	return mlw.ml.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (mlw Wrapper) FirstCollectTime() time.Time {
	return mlw.ml.FirstCollected
}

// LastCollectTime returns the time the last value was collected
func (mlw Wrapper) LastCollectTime() time.Time {
	return mlw.ml.LastCollected
}

// WantRelativeStats indicates if we want relative statistics
func (mlw Wrapper) WantRelativeStats() bool {
	return mlw.ml.WantRelativeStats()
}

// Description returns a description of the table
func (mlw Wrapper) Description() string {
	n := len(mlw.ml.Results)
	count := wrapper.CountIf(n, func(i int) bool { return mlw.ml.Results[i].SumTimerWait > 0 })
	return fmt.Sprintf("Mutex Latency (events_waits_summary_global_by_event_name) %d rows", count)
}

// Headings returns the headings for a table
func (mlw Wrapper) Headings() string {
	return fmt.Sprintf("%10s %8s %8s|%s", "Latency", "MtxCnt", "%", "Mutex Name")
}

// content generate a printable result for a row, given the totals
func (mlw Wrapper) content(row, totals mutexlatency.Row) string {
	name := row.Name
	if row.CountStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %8s %8s|%s",
		utils.FormatTime(row.SumTimerWait),
		utils.FormatAmount(row.CountStar),
		utils.FormatPct(utils.Divide(row.SumTimerWait, totals.SumTimerWait)),
		name)
}

// sorting handled inline with sort.Slice to avoid repeated boilerplate types

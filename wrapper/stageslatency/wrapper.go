// Package stageslatency holds the routines which manage the stages table.
package stageslatency

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/model/stageslatency"
)

// Wrapper wraps a Stages struct
type Wrapper struct {
	sl *stageslatency.StagesLatency
}

// NewStagesLatency creates a wrapper around stageslatency
func NewStagesLatency(cfg *config.Config, db *sql.DB) *Wrapper {
	return &Wrapper{
		sl: stageslatency.NewStagesLatency(cfg, db),
	}
}

// ResetStatistics resets the statistics to last values
func (slw *Wrapper) ResetStatistics() {
	slw.sl.ResetStatistics()
}

// Collect data from the db, then merge it in.
func (slw *Wrapper) Collect() {
	slw.sl.Collect()
	sort.Sort(byLatency(slw.sl.Results))
}

// Headings returns the headings for a table
func (slw Wrapper) Headings() string {
	return fmt.Sprintf("%10s %6s %8s|%s", "Latency", "%", "Counter", "Stage Name")

}

// RowContent returns the rows we need for displaying
func (slw Wrapper) RowContent() []string {
	rows := make([]string, 0, len(slw.sl.Results))

	for i := range slw.sl.Results {
		rows = append(rows, slw.content(slw.sl.Results[i], slw.sl.Totals))
	}

	return rows
}

// TotalRowContent returns all the totals
func (slw Wrapper) TotalRowContent() string {
	return slw.content(slw.sl.Totals, slw.sl.Totals)
}

// Len return the length of the result set
func (slw Wrapper) Len() int {
	return len(slw.sl.Results)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (slw Wrapper) EmptyRowContent() string {
	var empty stageslatency.Row

	return slw.content(empty, empty)
}

// Description describe the stages
func (slw Wrapper) Description() string {
	var count int
	for row := range slw.sl.Results {
		if slw.sl.Results[row].SumTimerWait > 0 {
			count++
		}
	}

	return fmt.Sprintf("SQL Stage Latency (events_stages_summary_global_by_event_name) %d rows", count)
}

// HaveRelativeStats is true for this object
func (slw Wrapper) HaveRelativeStats() bool {
	return slw.sl.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (slw Wrapper) FirstCollectTime() time.Time {
	return slw.sl.FirstCollected
}

// LastCollectTime returns the time the last value was collected
func (slw Wrapper) LastCollectTime() time.Time {
	return slw.sl.LastCollected
}

// WantRelativeStats indiates if we want relative statistics
func (slw Wrapper) WantRelativeStats() bool {
	return slw.sl.WantRelativeStats()
}

// generate a printable result
func (slw Wrapper) content(row, totals stageslatency.Row) string {
	name := row.Name
	if row.CountStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s %8s|%s",
		lib.FormatTime(row.SumTimerWait),
		lib.FormatPct(lib.Divide(row.SumTimerWait, totals.SumTimerWait)),
		lib.FormatAmount(row.CountStar),
		name)
}

type byLatency stageslatency.Rows

func (rows byLatency) Len() int      { return len(rows) }
func (rows byLatency) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }

// sort by value (descending) but also by "name" (ascending) if the values are the same
func (rows byLatency) Less(i, j int) bool {
	return (rows[i].SumTimerWait > rows[j].SumTimerWait) ||
		((rows[i].SumTimerWait == rows[j].SumTimerWait) && (rows[i].Name < rows[j].Name))
}

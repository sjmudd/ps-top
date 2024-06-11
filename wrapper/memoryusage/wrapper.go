// Package memoryusage holds the routines which manage the file_summary_by_instance table.
package memoryusage

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/memoryusage"
	"github.com/sjmudd/ps-top/utils"
)

// Wrapper wraps a FileIoLatency struct  representing the contents of the data collected from file_summary_by_instance, but adding formatting for presentation in the terminal
type Wrapper struct {
	mu *memoryusage.MemoryUsage
}

// NewMemoryUsage creates a wrapper around MemoryUsage
func NewMemoryUsage(cfg *config.Config, db *sql.DB) *Wrapper {
	return &Wrapper{
		mu: memoryusage.NewMemoryUsage(cfg, db),
	}
}

// ResetStatistics resets the statistics to last values
func (muw *Wrapper) ResetStatistics() {
	muw.mu.ResetStatistics()
}

// Collect data from the db, then merge it in.
func (muw *Wrapper) Collect() {
	muw.mu.Collect()
	sort.Sort(byBytes(muw.mu.Results))
}

// Headings returns the headings for a table
func (muw Wrapper) Headings() string {
	return "CurBytes         %  High Bytes|MemOps          %|CurAlloc       %   HiAlloc|Memory Area"
	//      1234567890  100.0%  1234567890|123456789  100.0%|12345678  100.0%  12345678|Some memory name
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
	var empty memoryusage.Row

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

	return fmt.Sprintf("Memory Usage (memory_summary_global_by_event_name) %d rows", count)
}

// HaveRelativeStats is true for this object
func (muw Wrapper) HaveRelativeStats() bool {
	return muw.mu.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (muw Wrapper) FirstCollectTime() time.Time {
	return muw.mu.FirstCollected
}

// LastCollectTime returns the time the last value was collected
func (muw Wrapper) LastCollectTime() time.Time {
	return muw.mu.LastCollected
}

// WantRelativeStats indiates if we want relative statistics
func (muw Wrapper) WantRelativeStats() bool {
	return muw.mu.WantRelativeStats()
}

// content generate a printable result for a row, given the totals
func (muw Wrapper) content(row, totals memoryusage.Row) string {
	// assume the data is empty so hide it.
	name := row.Name
	if row.TotalMemoryOps == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s  %6s  %10s|%10s %6s|%8s  %6s  %8s|%s",
		utils.SignedFormatAmount(row.CurrentBytesUsed),
		utils.FormatPct(utils.SignedDivide(row.CurrentBytesUsed, totals.CurrentBytesUsed)),
		utils.SignedFormatAmount(row.HighBytesUsed),
		utils.SignedFormatAmount(row.TotalMemoryOps),
		utils.FormatPct(utils.SignedDivide(row.TotalMemoryOps, totals.TotalMemoryOps)),
		utils.SignedFormatAmount(row.CurrentCountUsed),
		utils.FormatPct(utils.SignedDivide(row.CurrentCountUsed, totals.CurrentCountUsed)),
		utils.SignedFormatAmount(row.HighCountUsed),
		name)
}

type byBytes []memoryusage.Row

func (t byBytes) Len() int      { return len(t) }
func (t byBytes) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t byBytes) Less(i, j int) bool {
	return (t[i].CurrentBytesUsed > t[j].CurrentBytesUsed) ||
		((t[i].CurrentBytesUsed == t[j].CurrentBytesUsed) &&
			(t[i].Name < t[j].Name))

}

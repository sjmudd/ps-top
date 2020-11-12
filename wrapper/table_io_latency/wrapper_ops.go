// Package file_io_latency holds the routines which manage the file_summary_by_instance table.
package file_io_latency

import (
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/table_io_latency"
)

// FileIoLatency represents the contents of the data collected from file_summary_by_instance
type WrapperOps struct {
	tiol *table_io_latency.TableIoLatency
}

// NewTableIoOps creates a wrapper around FileIoLatency
func NewTableIoOps(latency *WrapperLatency) *WrapperOps {
	return &WrapperOps{
		tiol: latency.tiol,
	}
}

// SetFirstFromLast resets the statistics to last values
func (tiolw *WrapperOps) SetFirstFromLast() {
	tiolw.tiol.SetFirstFromLast()
}

// Collect data from the db, then merge it in.
func (tiolw *WrapperOps) Collect() {
	tiolw.tiol.Collect()
}

// Headings returns the headings by operations as a string
func (tiolw WrapperOps) Headings() string {
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
func (tiolw WrapperOps) RowContent() []string {
	return tiolw.tiol.RowContent()
}

// Len return the length of the result set
func (tiolw WrapperOps) Len() int {
	return len(tiolw.tiol.Results)
}

// TotalRowContent returns all the totals
func (tiolw WrapperOps) TotalRowContent() string {
	return tiolw.tiol.TotalRowContent()
}

// EmptyRowContent returns an empty string of data (for filling in)
func (tiolw WrapperOps) EmptyRowContent() string {
	return tiolw.tiol.EmptyRowContent()
}

// Description returns a description of the table
func (tiolw WrapperOps) Description() string {
        var count int
        for row := range tiolw.tiol.Results {
                if tiolw.tiol.Results[row].HasData() {
                        count++
                }
        }

        return fmt.Sprintf("Table Ops (table_io_waits_summary_by_table) %d rows", count)
}

// HaveRelativeStats is true for this object
func (tiolw WrapperOps) HaveRelativeStats() bool {
	return true
}

// FirstCollectTime
func (tiolw WrapperOps) FirstCollectTime() time.Time {
	return tiolw.tiol.FirstCollectTime()
}

// LastCollectTime
func (tiolw WrapperOps) LastCollectTime() time.Time {
	return tiolw.tiol.LastCollectTime()
}

func (tiolw WrapperOps) WantRelativeStats() bool {
	return tiolw.tiol.WantRelativeStats()
}

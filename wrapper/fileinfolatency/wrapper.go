// Package fileinfolatency holds the routines which manage the file_summary_by_instance table.
package fileinfolatency

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/fileinfo"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/wrapper"
)

// Wrapper wraps a FileIoLatency struct representing the contents of the data collected from file_summary_by_instance, but adding formatting for presentation in the terminal
type Wrapper struct {
	fiol *fileinfo.FileIoLatency
}

// NewFileSummaryByInstance creates a wrapper around FileIoLatency
func NewFileSummaryByInstance(cfg *config.Config, db *sql.DB) *Wrapper {
	return &Wrapper{
		fiol: fileinfo.NewFileSummaryByInstance(cfg, db),
	}
}

// ResetStatistics resets the statistics to last values
func (fiolw *Wrapper) ResetStatistics() {
	fiolw.fiol.ResetStatistics()
}

// Collect data from the db, then merge it in.
func (fiolw *Wrapper) Collect() {
	fiolw.fiol.Collect()
	sort.Slice(fiolw.fiol.Results, func(i, j int) bool {
		return (fiolw.fiol.Results[i].SumTimerWait > fiolw.fiol.Results[j].SumTimerWait) ||
			((fiolw.fiol.Results[i].SumTimerWait == fiolw.fiol.Results[j].SumTimerWait) && (fiolw.fiol.Results[i].Name < fiolw.fiol.Results[j].Name))
	})
}

// Headings returns the headings for a table
func (fiolw Wrapper) Headings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s %6s|%8s %8s|%8s %6s %6s %6s|%s",
		"Latency",
		"%",
		"Read",
		"Write",
		"Misc",
		"Rd bytes",
		"Wr bytes",
		"Ops",
		"R Ops",
		"W Ops",
		"M Ops",
		"Table Name")
}

// RowContent returns the rows we need for displaying
func (fiolw Wrapper) RowContent() []string {
	n := len(fiolw.fiol.Results)
	return wrapper.RowsFromGetter(n, func(i int) string {
		return fiolw.content(fiolw.fiol.Results[i], fiolw.fiol.Totals)
	})
}

// TotalRowContent returns all the totals
func (fiolw Wrapper) TotalRowContent() string {
	return wrapper.TotalRowContent(fiolw.fiol.Totals, fiolw.content)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (fiolw Wrapper) EmptyRowContent() string {
	return wrapper.EmptyRowContent(fiolw.content)
}

// Description returns a description of the table
func (fiolw Wrapper) Description() string {
	n := len(fiolw.fiol.Results)
	count := wrapper.CountIf(n, func(i int) bool { return fiolw.fiol.Results[i].HasData() })
	return fmt.Sprintf("File I/O Latency (file_summary_by_instance) %d rows", count)
}

// HaveRelativeStats is true for this object
func (fiolw Wrapper) HaveRelativeStats() bool {
	return fiolw.fiol.HaveRelativeStats()
}

// FirstCollectTime returns the time the first value was collected
func (fiolw Wrapper) FirstCollectTime() time.Time {
	return fiolw.fiol.FirstCollected
}

// LastCollectTime returns the time the last value was collected
func (fiolw Wrapper) LastCollectTime() time.Time {
	return fiolw.fiol.LastCollected
}

// WantRelativeStats indicates if we want relative statistics
func (fiolw Wrapper) WantRelativeStats() bool {
	return fiolw.fiol.WantRelativeStats()
}

// content generate a printable result for a row, given the totals
func (fiolw Wrapper) content(row, totals fileinfo.Row) string {
	var name = row.Name

	// We assume that if CountStar = 0 then there's no data at all...
	// when we have no data we really don't want to show the name either.
	if (row.SumTimerWait == 0 && row.CountStar == 0 && row.SumNumberOfBytesRead == 0 && row.SumNumberOfBytesWrite == 0) && name != "Totals" {
		name = ""
	}

	timeStr, pctStr := wrapper.TimePct(row.SumTimerWait, totals.SumTimerWait)
	pct := wrapper.PctStrings(row.SumTimerWait, row.SumTimerRead, row.SumTimerWrite, row.SumTimerMisc)
	opsPct := wrapper.PctStrings(row.CountStar, row.CountRead, row.CountWrite, row.CountMisc)

	return fmt.Sprintf("%10s %6s|%6s %6s %6s|%8s %8s|%8s %6s %6s %6s|%s",
		timeStr,
		pctStr,
		pct[0],
		pct[1],
		pct[2],
		utils.FormatAmount(row.SumNumberOfBytesRead),
		utils.FormatAmount(row.SumNumberOfBytesWrite),
		utils.FormatAmount(row.CountStar),
		opsPct[0],
		opsPct[1],
		opsPct[2],
		name)
}

// sorting handled inline with sort.Slice to avoid repeated boilerplate types

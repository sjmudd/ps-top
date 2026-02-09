// Package tableiolatency holds the routines which manage the tableio statisticss.
package tableiolatency

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/tableio"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/wrapper"
)

// Wrapper represents the contents of the data collected related to tableio statistics
type Wrapper struct {
	tiol *tableio.TableIo
}

// NewTableIoLatency creates a wrapper around tableio statistics
func NewTableIoLatency(cfg *config.Config, db *sql.DB) *Wrapper {
	return &Wrapper{
		tiol: tableio.NewTableIo(cfg, db),
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
	sort.Slice(tiolw.tiol.Results, func(i, j int) bool {
		return (tiolw.tiol.Results[i].SumTimerWait > tiolw.tiol.Results[j].SumTimerWait) ||
			((tiolw.tiol.Results[i].SumTimerWait == tiolw.tiol.Results[j].SumTimerWait) &&
				(tiolw.tiol.Results[i].Name < tiolw.tiol.Results[j].Name))
	})
}

// Headings returns the latency headings as a string
func (tiolw Wrapper) Headings() string {
	return wrapper.MakeTableIOHeadings("Latency")
}

// RowContent returns the rows we need for displaying
func (tiolw Wrapper) RowContent() []string {
	return wrapper.TableIORowContent(tiolw.tiol.Results, tiolw.tiol.Totals, tiolw.content)
}

// TotalRowContent returns all the totals
func (tiolw Wrapper) TotalRowContent() string {
	return wrapper.TableIOTotalRowContent(tiolw.tiol.Totals, tiolw.content)
}

// EmptyRowContent returns an empty string of data (for filling in)
func (tiolw Wrapper) EmptyRowContent() string {
	return wrapper.TableIOEmptyRowContent(tiolw.content)
}

// Description returns a description of the table
func (tiolw Wrapper) Description() string {
	return wrapper.TableIODescription("Latency", tiolw.tiol.Results, func(r tableio.Row) bool { return r.HasData() })
}

// HaveRelativeStats is true for this object
func (tiolw Wrapper) HaveRelativeStats() bool {
	return tiolw.tiol.HaveRelativeStats()
}

// FirstCollectTime returns the time of the first collection
func (tiolw Wrapper) FirstCollectTime() time.Time {
	return tiolw.tiol.FirstCollected
}

// LastCollectTime returns the time of the last collection
func (tiolw Wrapper) LastCollectTime() time.Time {
	return tiolw.tiol.LastCollected
}

// WantRelativeStats returns if we want to see relative stats
func (tiolw Wrapper) WantRelativeStats() bool {
	return tiolw.tiol.WantRelativeStats()
}

// latencyRowContents returns the printable result
func (tiolw Wrapper) content(row, totals tableio.Row) string {
	// assume the data is empty so hide it.
	name := row.Name
	if row.CountStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s",
		utils.FormatTime(row.SumTimerWait),
		utils.FormatPct(utils.Divide(row.SumTimerWait, totals.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerFetch, row.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerInsert, row.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerUpdate, row.SumTimerWait)),
		utils.FormatPct(utils.Divide(row.SumTimerDelete, row.SumTimerWait)),
		name)
}

// for sorting
// sorting handled inline with sort.Slice to avoid repeated boilerplate types

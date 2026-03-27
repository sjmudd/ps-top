// Package tableioops holds the routines which manage the table ops
package tableioops

import (
	"fmt"
	"slices"
	"time"

	"github.com/sjmudd/ps-top/model/tableio"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/wrapper"
	"github.com/sjmudd/ps-top/wrapper/tableiolatency"
)

// Wrapper represents a wrapper around tableiolatency
// - the latency wrapper is only to be used for common functionality between the 2 structs
type Wrapper struct {
	tiol    *tableio.TableIo
	latency *tableiolatency.Wrapper
}

// NewTableIoOps creates a wrapper around TableIo, sharing the same connection with the tableiolatency wrapper
func NewTableIoOps(latency *tableiolatency.Wrapper) *Wrapper {
	return &Wrapper{
		tiol:    latency.Tiol(),
		latency: latency,
	}
}

// ResetStatistics resets the statistics to last values
func (tiolw *Wrapper) ResetStatistics() {
	tiolw.tiol.ResetStatistics()
}

// Collect data from the db, then merge it in.
func (tiolw *Wrapper) Collect() {
	tiolw.tiol.Collect()

	// sort the results by ops == CountStar (descending), Name
	slices.SortFunc(tiolw.tiol.Results, func(a, b tableio.Row) int {
		if a.CountStar > b.CountStar {
			return -1
		}
		if a.CountStar < b.CountStar {
			return 1
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
}

// Headings returns the headings by operations as a string
func (tiolw Wrapper) Headings() string {
	return wrapper.MakeTableIOHeadings("Ops")
}

// content returns the printable content of a row given the totals details
func (tiolw Wrapper) content(row, totals tableio.Row) string {
	// assume the data is empty so hide it.
	name := row.Name
	if row.CountStar == 0 && name != "Totals" {
		name = ""
	}

	// Read/Write percentages placed before fetch/insert/update/delete with extra separator
	return fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s|%s",
		utils.FormatCounterU(row.CountStar, 10),
		utils.FormatPct(utils.Divide(row.CountStar, totals.CountStar)),
		utils.FormatPct(utils.Divide(row.CountRead, row.CountStar)),
		utils.FormatPct(utils.Divide(row.CountWrite, row.CountStar)),
		utils.FormatPct(utils.Divide(row.CountFetch, row.CountStar)),
		utils.FormatPct(utils.Divide(row.CountInsert, row.CountStar)),
		utils.FormatPct(utils.Divide(row.CountUpdate, row.CountStar)),
		utils.FormatPct(utils.Divide(row.CountDelete, row.CountStar)),
		name)
}

// RowContent returns the rows we need for displaying
func (tiolw Wrapper) RowContent() []string {
	return wrapper.TableIORowContent(tiolw.tiol.Results, tiolw.tiol.Totals, tiolw.content)
}

// TotalRowContent returns all the totals
func (tiolw Wrapper) TotalRowContent() string {
	return tiolw.latency.TotalRowContent()
}

// EmptyRowContent returns an empty string of data (for filling in)
func (tiolw Wrapper) EmptyRowContent() string {
	return tiolw.latency.EmptyRowContent()
}

// Description returns a description of the table
func (tiolw Wrapper) Description() string {
	return wrapper.TableIODescription("Ops", tiolw.tiol.Results, func(r tableio.Row) bool { return r.HasData() })
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

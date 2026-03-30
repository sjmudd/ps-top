// Package tableioops holds the routines which manage table I/O operations statistics.
package tableioops

import (
	"fmt"
	"slices"

	"github.com/sjmudd/ps-top/model/tableio"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/wrapper"
	"github.com/sjmudd/ps-top/wrapper/tableiolatency"
)

var (
	// sort by CountStar (ops) descending, Name ascending.
	defaultSort = func(rows []tableio.Row) {
		slices.SortFunc(rows, func(a, b tableio.Row) int {
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

	defaultHasData = func(r tableio.Row) bool { return r.HasData() }

	defaultContent = func(row, totals tableio.Row) string {
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
)

// Wrapper represents a wrapper around table I/O ops. It shares the same underlying
// TableIo model as the latency wrapper but presents different formatting and sorting.
type Wrapper struct {
	*wrapper.BaseWrapper[tableio.Row, *tableio.TableIo]
	latency *tableiolatency.Wrapper
}

// NewTableIoOps creates a new ops wrapper that shares a TableIo model with the
// latency wrapper. It uses the latency wrapper for TotalRowContent and EmptyRowContent.
func NewTableIoOps(latency *tableiolatency.Wrapper) *Wrapper {
	// Get the shared TableIo model from the latency wrapper.
	tiol := latency.GetModel()

	// Build our own BaseWrapper using ops-specific parameters.
	bw := wrapper.NewBaseWrapper(
		tiol,
		"Table I/O Ops (table_io_waits_summary_by_table)",
		defaultSort,
		defaultHasData,
		defaultContent,
	)

	return &Wrapper{
		BaseWrapper: bw,
		latency:     latency,
	}
}

// TotalRowContent returns the total row content by delegating to the latency wrapper.
func (w *Wrapper) TotalRowContent() string {
	return w.latency.TotalRowContent()
}

// EmptyRowContent returns the empty row content by delegating to the latency wrapper.
func (w *Wrapper) EmptyRowContent() string {
	return w.latency.EmptyRowContent()
}

// Headings returns the ops headings.
func (w *Wrapper) Headings() string {
	return wrapper.MakeTableIOHeadings("Ops")
}

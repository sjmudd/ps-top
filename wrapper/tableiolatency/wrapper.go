// Package tableiolatency holds the routines which manage table IO latency statistics.
package tableiolatency

import (
	"fmt"
	"slices"

	"github.com/sjmudd/ps-top/model/tableio"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/wrapper"
)

// Default functions for BaseWrapper, shared with tests.
var (
	defaultSort = func(rows []tableio.Row) {
		slices.SortFunc(rows, func(a, b tableio.Row) int {
			return utils.SumTimerWaitNameOrdering(
				utils.NewSumTimerWaitName(a.Name, a.SumTimerWait),
				utils.NewSumTimerWaitName(b.Name, b.SumTimerWait),
			)
		})
	}

	defaultHasData = func(r tableio.Row) bool { return r.HasData() }

	defaultContent = func(row, totals tableio.Row) string {
		// assume the data is empty so hide it.
		name := row.Name
		if row.CountStar == 0 && name != "Totals" {
			name = ""
		}
		return fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s|%s",
			utils.FormatTime(row.SumTimerWait),
			utils.FormatPct(utils.Divide(row.SumTimerWait, totals.SumTimerWait)),
			utils.FormatPct(utils.Divide(row.SumTimerRead, row.SumTimerWait)),
			utils.FormatPct(utils.Divide(row.SumTimerWrite, row.SumTimerWait)),
			utils.FormatPct(utils.Divide(row.SumTimerFetch, row.SumTimerWait)),
			utils.FormatPct(utils.Divide(row.SumTimerInsert, row.SumTimerWait)),
			utils.FormatPct(utils.Divide(row.SumTimerUpdate, row.SumTimerWait)),
			utils.FormatPct(utils.Divide(row.SumTimerDelete, row.SumTimerWait)),
			name)
	}
)

// Wrapper wraps a TableIo struct and implements Tabler via BaseWrapper.
type Wrapper struct {
	*wrapper.BaseWrapper[tableio.Row, *tableio.TableIo]
}

// NewTableIoLatency creates a wrapper around tableio statistics using the provided model.
func NewTableIoLatency(model *tableio.TableIo) *Wrapper {
	bw := wrapper.NewBaseWrapper(
		model,
		"Table I/O Latency (table_io_waits_summary_by_table)",
		defaultSort,
		defaultHasData,
		defaultContent,
	)
	return &Wrapper{BaseWrapper: bw}
}

// Headings returns the latency headings as a string.
func (w *Wrapper) Headings() string {
	return wrapper.MakeTableIOHeadings("Latency")
}

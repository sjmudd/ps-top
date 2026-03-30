// Package tableioops holds the routines which manage table I/O operations statistics.
package tableioops

import (
	"fmt"
	"slices"

	"github.com/sjmudd/ps-top/model/tableio"
	"github.com/sjmudd/ps-top/presenter"
	"github.com/sjmudd/ps-top/utils"
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

// Presenter presents table I/O ops. It uses a shared TableIo model
// but different formatting and sorting than the latency presenter.
type Presenter struct {
	*presenter.BasePresenter[tableio.Row, *tableio.TableIo]
}

// NewTableIoOps creates a new ops presenter using the provided shared TableIo model.
func NewTableIoOps(model *tableio.TableIo) *Presenter {
	bp := presenter.NewBasePresenter(
		model,
		"Table I/O Ops (table_io_waits_summary_by_table)",
		defaultSort,
		defaultHasData,
		defaultContent,
	)
	return &Presenter{BasePresenter: bp}
}

// Headings returns the ops headings.
func (p *Presenter) Headings() string {
	return presenter.MakeTableIOHeadings("Ops")
}

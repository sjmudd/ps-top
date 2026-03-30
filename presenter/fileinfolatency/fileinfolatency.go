// Package fileinfolatency holds the routines which manage the file_summary_by_instance table.
package fileinfolatency

import (
	"database/sql"
	"fmt"
	"slices"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/fileinfo"
	"github.com/sjmudd/ps-top/presenter"
	"github.com/sjmudd/ps-top/utils"
)

var (
	defaultSort = func(rows []fileinfo.Row) {
		slices.SortFunc(rows, func(a, b fileinfo.Row) int {
			return utils.SumTimerWaitNameOrdering(
				utils.NewSumTimerWaitName(a.Name, a.SumTimerWait),
				utils.NewSumTimerWaitName(b.Name, b.SumTimerWait),
			)
		})
	}

	defaultHasData = func(r fileinfo.Row) bool { return r.HasData() }

	defaultContent = func(row, totals fileinfo.Row) string {
		var name = row.Name

		// We assume that if CountStar = 0 then there's no data at all...
		// when we have no data we really don't want to show the name either.
		if (row.SumTimerWait == 0 && row.CountStar == 0 && row.SumNumberOfBytesRead == 0 && row.SumNumberOfBytesWrite == 0) && name != "Totals" {
			name = ""
		}

		timeStr, pctStr := presenter.TimePct(row.SumTimerWait, totals.SumTimerWait)
		pct := presenter.PctStrings(row.SumTimerWait, row.SumTimerRead, row.SumTimerWrite, row.SumTimerMisc)
		opsPct := presenter.PctStrings(row.CountStar, row.CountRead, row.CountWrite, row.CountMisc)

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
)

// Presenter presents a FileIoLatency struct.
type Presenter struct {
	*presenter.BasePresenter[fileinfo.Row, *fileinfo.FileIoLatency]
}

// NewFileSummaryByInstance creates a presenter for FileIoLatency.
func NewFileSummaryByInstance(cfg *config.Config, db *sql.DB) *Presenter {
	fiol := fileinfo.NewFileSummaryByInstance(cfg, db)
	bp := presenter.NewBasePresenter(
		fiol,
		"File I/O Latency (file_summary_by_instance)",
		defaultSort,
		defaultHasData,
		defaultContent,
	)
	return &Presenter{BasePresenter: bp}
}

// Headings returns the headings for a table.
func (p *Presenter) Headings() string {
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

// Package memoryusage holds the routines which manage the memory usage table.
package memoryusage

import (
	"database/sql"
	"fmt"
	"slices"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/model/memoryusage"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/wrapper"
)

// Wrapper wraps a MemoryUsage struct and implements the Tabler interface
// via embedded BaseWrapper.
type Wrapper struct {
	*wrapper.BaseWrapper[memoryusage.Row, *memoryusage.MemoryUsage]
}

// NewMemoryUsage creates a wrapper around MemoryUsage.
func NewMemoryUsage(cfg *config.Config, db *sql.DB) *Wrapper {
	mu := memoryusage.NewMemoryUsage(cfg, db)

	// Sort by CurrentBytesUsed descending, then Name ascending.
	sortFn := func(rows []memoryusage.Row) {
		slices.SortFunc(rows, func(a, b memoryusage.Row) int {
			if a.CurrentBytesUsed > b.CurrentBytesUsed {
				return -1
			}
			if a.CurrentBytesUsed < b.CurrentBytesUsed {
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

	// Count rows with meaningful data.
	hasData := func(r memoryusage.Row) bool { return r.HasData() }

	// Format a single row.
	contentFn := func(row, totals memoryusage.Row) string {
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

	bw := wrapper.NewBaseWrapper(mu,
		"Memory Usage (memory_summary_global_by_event_name)",
		sortFn,
		hasData,
		contentFn,
	)
	return &Wrapper{BaseWrapper: bw}
}

// Headings returns the headings for a table.
func (w *Wrapper) Headings() string {
	return "CurBytes         %  High Bytes|MemOps          %|CurAlloc       %   HiAlloc|Memory Area"
	//      1234567890  100.0%  1234567890|123456789  100.0%|12345678  100.0%  12345678|Some memory name
}

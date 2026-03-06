package wrapper

import (
	"fmt"

	"github.com/sjmudd/ps-top/model/tableio"
	"github.com/sjmudd/ps-top/utils"
)

// RowsFromGetter builds a slice of strings by calling the provided getter
// for each index. This centralizes the common loop used by many wrapper
// packages to produce display rows.
func RowsFromGetter(n int, get func(i int) string) []string {
	rows := make([]string, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, get(i))
	}
	return rows
}

// CountIf counts how many indices in [0,n) satisfy the predicate.
// Used by wrappers to implement Description() which counts rows with data.
func CountIf(n int, pred func(i int) bool) int {
	count := 0
	for i := 0; i < n; i++ {
		if pred(i) {
			count++
		}
	}
	return count
}

// TotalRowContent returns the formatted totals row by calling the provided
// content function with the totals value for both row and totals.
// This removes the repeated pattern found in many wrapper packages.
func TotalRowContent[T any](totals T, content func(T, T) string) string {
	return content(totals, totals)
}

// EmptyRowContent returns the formatted empty row by calling the provided
// content function with a zero value for the row and totals. It uses Go
// generics to avoid repeating the same empty-construction pattern.
func EmptyRowContent[T any](content func(T, T) string) string {
	var empty T
	return content(empty, empty)
}

// MakeTableIOHeadings constructs a heading string used by the tableio wrappers.
// The `kind` parameter should be either "Latency" or "Ops" (or similar) and will
// be interpolated into the common table IO heading format.
func MakeTableIOHeadings(kind string) string {
	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s",
		kind,
		"%",
		"Fetch",
		"Insert",
		"Update",
		"Delete",
		"Table Name")
}

// MakeTableIODescription builds the description string for table IO wrappers.
func MakeTableIODescription(kind string, count int) string {
	return fmt.Sprintf("Table %s (table_io_waits_summary_by_table) %d rows", kind, count)
}

// TimePct returns the formatted time and percentage strings for a row's
// SumTimerWait and the total SumTimerWait. This small helper centralizes
// the common prefix used by several wrapper content formatters.
func TimePct(sum, totals uint64) (string, string) {
	return utils.FormatTime(sum), utils.FormatPct(utils.Divide(sum, totals))
}

// PctStrings returns a slice of formatted percentage strings for each value
// relative to the provided total. This centralizes the common pattern of
// calling utils.FormatPct(utils.Divide(value, total)). It helps reduce
// duplicated code across wrapper content formatters.
func PctStrings(total uint64, values ...uint64) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = utils.FormatPct(utils.Divide(v, total))
	}
	return out
}

// RowsFromSlice builds a slice of strings by applying the provided content
// function to each element of `slice` along with the provided totals value.
// This consolidates the common RowContent pattern used by tableio wrappers.
func RowsFromSlice[T any](slice []T, totals T, content func(T, T) string) []string {
	n := len(slice)
	return RowsFromGetter(n, func(i int) string { return content(slice[i], totals) })
}

// TableIO helpers: small delegating helpers specialized for tableio.Row so
// wrappers can call a single helper instead of reimplementing the same
// RowContent/TotalRowContent/EmptyRowContent/Description trio.
func TableIORowContent(slice []tableio.Row, totals tableio.Row, content func(tableio.Row, tableio.Row) string) []string {
	return RowsFromSlice(slice, totals, content)
}

func TableIOTotalRowContent(totals tableio.Row, content func(tableio.Row, tableio.Row) string) string {
	return TotalRowContent(totals, content)
}

func TableIOEmptyRowContent(content func(tableio.Row, tableio.Row) string) string {
	return EmptyRowContent(content)
}

func TableIODescription(kind string, slice []tableio.Row, hasData func(tableio.Row) bool) string {
	n := len(slice)
	count := CountIf(n, func(i int) bool { return hasData(slice[i]) })
	return MakeTableIODescription(kind, count)
}

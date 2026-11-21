// Package presenter turns the difference over a window into table lines.
//
// A view is a definition, not code: which metric, how its raw keys map to
// shown keys, which column orders the rows, and the cells of each line. Only
// the snapshots a view needs are mapped.
//
// # Cells
//
// Each line is groups of cells. A cell shows the row's key or one column of
// the row:
//
//   - Key(heading, width): the row's key (after mapping), "Totals" on the
//     totals line. Width 0 prints it as it is: use that only for a key in last
//     place. A width above 0 left-aligns it in width characters and cuts a
//     longer key, so the cells after it stay aligned.
//   - Value(heading, column, format, width): the column's value, formatted,
//     e.g. "   8.00 ms" or "  1.50 M".
//   - ShareOfTotal(heading, column): the column's value as a share of that
//     column's total over all rows, e.g. " 40.0%".
//   - ShareOf(heading, column, base): the column's value as a share of another
//     column of the same row, e.g. read time of all time: " 25.0%".
//
// Cells in a group are separated by a space and groups by "|". A value cell
// is width characters wide and a share 6; their headings are right-aligned,
// a key's heading left-aligned. The totals line is drawn the same way from
// the column totals.
//
// The key is usually last, in a group of its own: lines are cut at the output
// width, and then only the key is cut.
//
// # Example
//
// The file I/O latency view:
//
//	Cells: [][]Cell{
//		{Value("Latency", "sum_timer_wait", formatter.Time, 10), ShareOfTotal("%", "sum_timer_wait")},
//		{ShareOf("Read", "sum_timer_read", "sum_timer_wait"), ShareOf("Write", "sum_timer_write", "sum_timer_wait"),
//			ShareOf("Misc", "sum_timer_misc", "sum_timer_wait")},
//		{Value("Rd bytes", "sum_number_of_bytes_read", formatter.Amount, 8),
//			Value("Wr bytes", "sum_number_of_bytes_write", formatter.Amount, 8)},
//		{Value("Ops", "count_star", formatter.Amount, 8), ShareOf("R Ops", "count_read", "count_star"),
//			ShareOf("W Ops", "count_write", "count_star"), ShareOf("M Ops", "count_misc", "count_star")},
//		{Key("Table Name", 0)},
//	}
//
// For a row with wait 8 ms (read 2, write 5, misc 1), 1.5 MiB read, 640 KiB
// written, 400 operations (read 100, write 250, misc 50), and a total wait of
// 20 ms over all rows, the heading and the line are:
//
//	Latency      %|  Read  Write   Misc|Rd bytes Wr bytes|     Ops  R Ops  W Ops  M Ops|Table Name
//	8.00 ms  40.0%| 25.0%  62.5%  12.5%|  1.50 M 640.00 k|     400  25.0%  62.5%  12.5%|shop.orders
//
// (Both lines start with three more spaces: "Latency" and "8.00 ms" are
// right-aligned in 10 characters.)
//
// Reading the first group: "   8.00 ms" is Value of sum_timer_wait, " 40.0%"
// is 8 of the 20 ms total. In the last group, " 25.0%" is count_read (100) of
// count_star (400) of the same row.
package presenter

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/sjmudd/ps-top/internal/collector"
	"github.com/sjmudd/ps-top/internal/derive"
	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/transform"
)

// shareWidth is the width of a share cell: formatter.Pct is 6 characters.
const shareWidth = 6

// Source gives the start and end snapshots of the last n snapshots of a metric.
type Source interface {
	Window(ctx context.Context, metric string, n int) (collector.Window, error)
}

type cellKind int

const (
	value cellKind = iota
	shareOfTotal
	shareOf
	key
)

// Cell is one cell of a line. Build it with Key, Value, ShareOfTotal or
// ShareOf.
type Cell struct {
	Heading string
	Column  string              // not used by Key
	Base    string              // ShareOf: the column it is a share of
	Format  func(uint64) string // Value: formats the column's value
	Width   int
	kind    cellKind
}

// Key shows the row's key: as it is if width is 0, else left-aligned in
// width characters and cut to width.
func Key(heading string, width int) Cell {
	return Cell{Heading: heading, Width: width, kind: key}
}

// Value shows column's value formatted by format, right-aligned in width
// characters.
func Value(heading, column string, format func(uint64) string, width int) Cell {
	return Cell{Heading: heading, Column: column, Format: format, Width: width, kind: value}
}

// ShareOfTotal shows column's value as a share of the column's total over
// all rows.
func ShareOfTotal(heading, column string) Cell {
	return Cell{Heading: heading, Column: column, Width: shareWidth, kind: shareOfTotal}
}

// ShareOf shows column's value as a share of column base of the same row.
func ShareOf(heading, column, base string) Cell {
	return Cell{Heading: heading, Column: column, Base: base, Width: shareWidth, kind: shareOf}
}

// View is a table over one metric.
type View struct {
	Title  string // description, e.g. "Table I/O Latency (table_io_waits_summary_by_table)"
	Def    metric.Definition
	MapKey func(string) string // raw key to shown key; nil: as collected
	Cells  [][]Cell            // groups of cells, separated by "|"
	// SortColumn orders the rows, highest first, then by key, and hides
	// rows where it is 0.
	SortColumn string
}

// resolved is a cell with its columns as indexes into the values.
type resolved struct {
	Cell
	col, base int
}

// Render returns the view's lines for a window of n snapshots, each cut at
// width runes: description, headings, one line per key with a non-zero
// SortColumn (highest first, then key), and the totals line.
func Render(ctx context.Context, src Source, v View, n, width int) ([]string, error) {
	sortCol, groups, err := v.resolve()
	if err != nil {
		return nil, err
	}
	w, err := src.Window(ctx, v.Def.Name, n)
	if err != nil {
		return nil, err
	}
	if !w.OK {
		return []string{cut(v.Title+" (waiting for a second snapshot)", width)}, nil
	}
	start, err := transform.Map(v.Def, w.Start, v.MapKey)
	if err != nil {
		return nil, err
	}
	end, err := transform.Map(v.Def, w.End, v.MapKey)
	if err != nil {
		return nil, err
	}
	d, err := derive.Window(v.Def, start, end)
	if err != nil {
		return nil, err
	}

	var shown []derive.Entry
	for _, e := range d.Entries {
		if e.Values[sortCol] > 0 {
			shown = append(shown, e)
		}
	}
	slices.SortFunc(shown, func(a, b derive.Entry) int {
		if c := cmp.Compare(b.Values[sortCol], a.Values[sortCol]); c != 0 {
			return c
		}
		return cmp.Compare(a.Key.Value(), b.Key.Value())
	})

	lines := []string{
		fmt.Sprintf("%s %d rows", v.Title, len(shown)),
		v.headings(),
	}
	for _, e := range shown {
		lines = append(lines, line(groups, e.Values, d.Totals, e.Key.Value()))
	}
	lines = append(lines, line(groups, d.Totals, d.Totals, "Totals"))
	for i := range lines {
		lines[i] = cut(lines[i], width)
	}
	return lines, nil
}

// resolve returns the index of SortColumn and the cells with their column
// indexes.
func (v View) resolve() (int, [][]resolved, error) {
	index := make(map[string]int, len(v.Def.Columns))
	for i, c := range v.Def.Columns {
		index[c.Name] = i
	}
	find := func(name string) (int, error) {
		i, ok := index[name]
		if !ok {
			return 0, fmt.Errorf("presenter: view %q: metric %q has no column %q", v.Title, v.Def.Name, name)
		}
		return i, nil
	}
	sortCol, err := find(v.SortColumn)
	if err != nil {
		return 0, nil, err
	}
	groups := make([][]resolved, len(v.Cells))
	for g, cells := range v.Cells {
		for _, c := range cells {
			r := resolved{Cell: c}
			if c.kind == key {
				groups[g] = append(groups[g], r)
				continue
			}
			if r.col, err = find(c.Column); err != nil {
				return 0, nil, err
			}
			if c.kind == shareOf {
				if r.base, err = find(c.Base); err != nil {
					return 0, nil, err
				}
			}
			groups[g] = append(groups[g], r)
		}
	}
	return sortCol, groups, nil
}

func (v View) headings() string {
	var b strings.Builder
	for g, cells := range v.Cells {
		if g > 0 {
			b.WriteString("|")
		}
		for i, c := range cells {
			if i > 0 {
				b.WriteString(" ")
			}
			if c.kind == key {
				b.WriteString(keyText(c.Heading, c.Width))
				continue
			}
			fmt.Fprintf(&b, "%*s", c.Width, c.Heading)
		}
	}
	return b.String()
}

// keyText returns s as it is if width is 0, else left-aligned in width runes
// and cut to width.
func keyText(s string, width int) string {
	if width == 0 {
		return s
	}
	return fmt.Sprintf("%-*s", width, cut(s, width))
}

// line draws one row, named name, from its values and the column totals.
func line(groups [][]resolved, values, totals []uint64, name string) string {
	var b strings.Builder
	for g, cells := range groups {
		if g > 0 {
			b.WriteString("|")
		}
		for i, c := range cells {
			if i > 0 {
				b.WriteString(" ")
			}
			var s string
			switch c.kind {
			case key:
				b.WriteString(keyText(name, c.Width))
				continue
			case value:
				s = c.Format(values[c.col])
			case shareOfTotal:
				s = formatter.Pct(formatter.Divide(values[c.col], totals[c.col]))
			case shareOf:
				s = formatter.Pct(formatter.Divide(values[c.col], values[c.base]))
			}
			fmt.Fprintf(&b, "%*s", c.Width, s)
		}
	}
	return b.String()
}

// cut returns s limited to width runes.
func cut(s string, width int) string {
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width])
}

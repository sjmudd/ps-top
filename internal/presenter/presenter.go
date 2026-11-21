// Package presenter turns the difference over a window into table lines.
//
// A view is a definition, not code. To build one you need:
//
//   - Title: what the view shows, e.g. "File I/O Latency". The line above
//     the headings adds the table, from the Spec, and the number of rows.
//   - Spec: the metric it shows, e.g. fileio.Spec (see the metrics package
//     for adding a metric).
//   - MapKey: how a raw key becomes the shown key, e.g. a file name to
//     schema.table. Nil shows keys as collected. Rows that map to the same
//     key are added together. Only the snapshots a view needs are mapped.
//   - Cells: what a line shows, in groups of cells.
//   - SortColumn: the column that orders the rows, highest first, then by
//     key. Rows where it is 0 are not shown.
//
// View is generic in the metric's column type: View[fileio.Column] can only
// take fileio columns, so a view cannot name a column of another metric. The
// app keeps views of different metrics in one list as Renderer.
//
// # Cells
//
// A cell shows the row's key or one column of the row:
//
//   - Key[C](heading, width): the row's key (after mapping), "Totals" on the
//     totals line. Width 0 prints it as it is: use that only for a key in last
//     place. A width above 0 left-aligns it in width characters and cuts a
//     longer key, so the cells after it stay aligned. Key names no column, so
//     the column type is written out: Key[fileio.Column]("Table Name", 0).
//   - Value(heading, column, format, width): the column's value, formatted,
//     e.g. "   8.00 ms" or "  1.50 M".
//   - ShareOfTotal(heading, column, width): the column's value as a share of
//     that column's total over all rows, e.g. " 40.0%".
//   - ShareOf(heading, column, base, width): the column's value as a share of
//     another column of the same row, e.g. read time of all time: " 25.0%".
//
// Cells in a group are separated by a space and groups by "|". Value and share
// cells are right-aligned in width characters, and so are their headings. A
// key's heading is left-aligned. A share is formatted in 6 characters, so its
// width is at least 6. The totals line is drawn the same way from the column
// totals.
//
// The key is usually last, in a group of its own: lines are cut at the output
// width, and then only the key is cut.
//
// # Example
//
// The file I/O latency view:
//
//	View[fileio.Column]{
//		Title:  "File I/O Latency",
//		Spec:   fileio.Spec,
//		MapKey: mapKey,
//		Cells: [][]Cell[fileio.Column]{
//			{Value("Latency", fileio.SumTimerWait, formatter.Time, 10), ShareOfTotal("%", fileio.SumTimerWait, 6)},
//			{ShareOf("Read", fileio.SumTimerRead, fileio.SumTimerWait, 6), ...},
//			{Value("Rd bytes", fileio.SumNumberOfBytesRead, formatter.Amount, 8), ...},
//			{Value("Ops", fileio.CountStar, formatter.Amount, 8), ShareOf("R Ops", fileio.CountRead, fileio.CountStar, 6), ...},
//			{Key[fileio.Column]("Table Name", 0)},
//		},
//		SortColumn: fileio.SumTimerWait,
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
// Reading the first group: "   8.00 ms" is Value of SUM_TIMER_WAIT, " 40.0%"
// is 8 of the 20 ms total. In the last group, " 25.0%" is COUNT_READ (100) of
// COUNT_STAR (400) of the same row.
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

// Source gives the start and end snapshots of the last n snapshots of a metric.
type Source interface {
	Window(ctx context.Context, metric string, n int) (collector.Window, error)
}

// Renderer is a view of any metric.
type Renderer interface {
	// Render returns the view's lines for a window of n snapshots, each cut
	// at width runes.
	Render(ctx context.Context, src Source, n, width int) ([]string, error)
}

type cellKind int

const (
	value cellKind = iota
	shareOfTotal
	shareOf
	key
)

// Cell is one cell of a line of a view of a metric with column type C. Build
// it with Key, Value, ShareOfTotal or ShareOf.
type Cell[C ~int] struct {
	Heading string
	Column  C                   // not used by Key
	Base    C                   // ShareOf: the column it is a share of
	Format  func(uint64) string // Value: formats the column's value
	Width   int
	kind    cellKind
}

// Key shows the row's key: as it is if width is 0, else left-aligned in
// width characters and cut to width.
func Key[C ~int](heading string, width int) Cell[C] {
	return Cell[C]{Heading: heading, Width: width, kind: key}
}

// Value shows column's value formatted by format, right-aligned in width
// characters.
func Value[C ~int](heading string, column C, format func(uint64) string, width int) Cell[C] {
	return Cell[C]{Heading: heading, Column: column, Format: format, Width: width, kind: value}
}

// ShareOfTotal shows column's value as a share of the column's total over
// all rows, right-aligned in width characters (at least 6).
func ShareOfTotal[C ~int](heading string, column C, width int) Cell[C] {
	return Cell[C]{Heading: heading, Column: column, Width: width, kind: shareOfTotal}
}

// ShareOf shows column's value as a share of column base of the same row,
// right-aligned in width characters (at least 6).
func ShareOf[C ~int](heading string, column, base C, width int) Cell[C] {
	return Cell[C]{Heading: heading, Column: column, Base: base, Width: width, kind: shareOf}
}

// View is a table over one metric with column type C.
type View[C ~int] struct {
	Title  string // e.g. "Table I/O Latency"
	Spec   metric.Spec[C]
	MapKey func(string) string // raw key to shown key, or nil to show keys as collected
	Cells  [][]Cell[C]         // groups of cells, separated by "|"
	// SortColumn orders the rows, highest first, then by key, and hides
	// rows where it is 0.
	SortColumn C
}

// Render returns the view's lines for a window of n snapshots, each cut at
// width runes: description, headings, one line per key with a non-zero
// SortColumn (highest first, then key), and the totals line.
func (v View[C]) Render(ctx context.Context, src Source, n, width int) ([]string, error) {
	if err := v.check(); err != nil {
		return nil, err
	}
	title := v.Title + " (" + v.Spec.Table + ")"
	def := v.Spec.Definition()
	w, err := src.Window(ctx, def.Name, n)
	if err != nil {
		return nil, err
	}
	if !w.OK {
		return []string{cut(title+" (waiting for a second snapshot)", width)}, nil
	}
	start, err := transform.Map(def, w.Start, v.MapKey)
	if err != nil {
		return nil, err
	}
	end, err := transform.Map(def, w.End, v.MapKey)
	if err != nil {
		return nil, err
	}
	d, err := derive.Window(def, start, end)
	if err != nil {
		return nil, err
	}

	sortCol := int(v.SortColumn)
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
		fmt.Sprintf("%s %d rows", title, len(shown)),
		v.headings(),
	}
	for _, e := range shown {
		lines = append(lines, v.line(e.Values, d.Totals, e.Key.Value()))
	}
	lines = append(lines, v.line(d.Totals, d.Totals, "Totals"))
	for i := range lines {
		lines[i] = cut(lines[i], width)
	}
	return lines, nil
}

// check returns an error if SortColumn or a cell's column is not one of the
// Spec's columns.
func (v View[C]) check() error {
	cols := []C{v.SortColumn}
	for _, cells := range v.Cells {
		for _, c := range cells {
			switch c.kind {
			case value, shareOfTotal:
				cols = append(cols, c.Column)
			case shareOf:
				cols = append(cols, c.Column, c.Base)
			}
		}
	}
	for _, c := range cols {
		if err := v.Spec.Check(c); err != nil {
			return fmt.Errorf("presenter: view %q: %w", v.Title, err)
		}
	}
	return nil
}

func (v View[C]) headings() string {
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

// line draws one row, named name, from its values and the column totals.
func (v View[C]) line(values, totals []uint64, name string) string {
	var b strings.Builder
	for g, cells := range v.Cells {
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
				s = c.Format(values[c.Column])
			case shareOfTotal:
				s = formatter.Pct(formatter.Divide(values[c.Column], totals[c.Column]))
			case shareOf:
				s = formatter.Pct(formatter.Divide(values[c.Column], values[c.Base]))
			}
			fmt.Fprintf(&b, "%*s", c.Width, s)
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

// cut returns s limited to width runes.
func cut(s string, width int) string {
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width])
}

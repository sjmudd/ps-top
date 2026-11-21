// Package presenter turns the difference over a window into table lines.
// A view is a definition, not code: which metric, how its raw keys map to
// shown keys, which column is the total, which columns break it down, and how
// the total is formatted. Only the snapshots a view needs are mapped.
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

// Part is one breakdown column: its heading and the metric column it shows.
type Part struct {
	Label  string
	Column string
}

// View is a table over one metric.
type View struct {
	Title      string // description, e.g. "Table I/O Latency (table_io_waits_summary_by_table)"
	Def        metric.Definition
	Heading    string              // heading of the total column
	Total      string              // metric column shown as the total and sorted on
	Format     func(uint64) string // formats the total
	Groups     [][]Part            // shares of the line's total; groups separated by "|"
	KeyHeading string              // heading of the key column
	MapKey     func(string) string // raw key to shown key; nil: as collected
}

// Render returns the view's lines for a window of n snapshots, each cut at
// width runes: description, headings, one line per key with a non-zero total
// (total descending, then key), and the totals line.
func Render(ctx context.Context, src Source, v View, n, width int) ([]string, error) {
	total, groups, err := v.columns()
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
		if e.Values[total] > 0 {
			shown = append(shown, e)
		}
	}
	slices.SortFunc(shown, func(a, b derive.Entry) int {
		if c := cmp.Compare(b.Values[total], a.Values[total]); c != 0 {
			return c
		}
		return cmp.Compare(a.Key.Value(), b.Key.Value())
	})

	lines := []string{
		fmt.Sprintf("%s %d rows", v.Title, len(shown)),
		v.headings(),
	}
	for _, e := range shown {
		lines = append(lines, v.line(e.Values, d.Totals, total, groups, e.Key.Value()))
	}
	lines = append(lines, v.line(d.Totals, d.Totals, total, groups, "Totals"))
	for i := range lines {
		lines[i] = cut(lines[i], width)
	}
	return lines, nil
}

// columns returns the index of the total column and of each breakdown column.
func (v View) columns() (int, [][]int, error) {
	index := make(map[string]int, len(v.Def.Columns))
	for i, c := range v.Def.Columns {
		index[c.Name] = i
	}
	total, ok := index[v.Total]
	if !ok {
		return 0, nil, fmt.Errorf("presenter: view %q: metric %q has no column %q", v.Title, v.Def.Name, v.Total)
	}
	groups := make([][]int, len(v.Groups))
	for g, parts := range v.Groups {
		for _, p := range parts {
			i, ok := index[p.Column]
			if !ok {
				return 0, nil, fmt.Errorf("presenter: view %q: metric %q has no column %q", v.Title, v.Def.Name, p.Column)
			}
			groups[g] = append(groups[g], i)
		}
	}
	return total, groups, nil
}

func (v View) headings() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%10s %6s", v.Heading, "%")
	for _, parts := range v.Groups {
		b.WriteString("|")
		for i, p := range parts {
			if i > 0 {
				b.WriteString(" ")
			}
			fmt.Fprintf(&b, "%6s", p.Label)
		}
	}
	b.WriteString("|" + v.KeyHeading)
	return b.String()
}

// line shows the total, its share of the column total, each breakdown column
// as a share of the line's total, and the key.
func (v View) line(values, totals []uint64, total int, groups [][]int, key string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%10s %6s", v.Format(values[total]), formatter.Pct(formatter.Divide(values[total], totals[total])))
	for _, cols := range groups {
		b.WriteString("|")
		for i, c := range cols {
			if i > 0 {
				b.WriteString(" ")
			}
			fmt.Fprintf(&b, "%6s", formatter.Pct(formatter.Divide(values[c], values[total])))
		}
	}
	b.WriteString("|" + key)
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

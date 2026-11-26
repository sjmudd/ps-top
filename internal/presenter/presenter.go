// Package presenter turns the difference over a window into table lines.
//
// A view is a definition, not code. To build one you need:
//
//   - Name: the view's name, e.g. "file_io_latency", as v1 names its
//     views. It is the view's key in the metrics registry.
//   - Title: what the view shows, e.g. "File I/O Latency". The line above
//     the headings adds the table, from the Spec, and the number of rows.
//   - Spec: the metric it shows, e.g. fileio.Spec (see the metrics package
//     for adding a metric). The view shows the metric's default type, the
//     first of Spec.Types: current values (CurrentValue) or the change over
//     the window (DifferenceValues).
//   - MapKey: how a raw key becomes the shown key, e.g. a file name to
//     schema.table. Nil shows keys as collected. Rows that map to the same
//     key are added together. Only the snapshots a view needs are mapped.
//   - Anonymise, if the key shows names that may be private: how the shown
//     key is anonymised, e.g. anonymiseTable for schema1.table2. It is used
//     only with Options.Anonymiser set. Rows are still ordered by the real
//     names.
//   - Schema, if rows belong to a schema the query cannot filter on: the
//     schema of a raw key, or "" for none, e.g. a file's schema once its
//     name is mapped. With Options.DatabaseFilter set, only rows of those
//     schemas and rows with no schema are shown. Rows are filtered before
//     mapping.
//   - Cells: what a line shows, in groups of cells.
//   - SortColumn: the column that orders the rows, highest first, then by
//     key. Rows where it is 0 are not shown.
//   - ThenBy and ShowZero, if needed: columns that order rows with the same
//     SortColumn value, highest first, before the key, and whether rows
//     where SortColumn is 0 are shown.
//
// Keys of several parts (e.g. schema and table) are shown joined with ".".
// A column's total on the totals line follows its merge rule: the sum of the
// column, or for a DistinctPart column the distinct count over all raw
// entries.
//
// View is generic in the metric's column type: View[fileio.Column] can only
// take fileio columns, so a view cannot name a column of another metric. The
// app keeps views of different metrics in one list as Renderer.
//
// Each render records its stages (window, map, derive, order, draw) in the
// view's entry in the metrics registry, given by WithStats (see the registry
// package).
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
//   - Average(heading, column, count, format, width): the column's value
//     divided by a count column of the same row, formatted, e.g. the time per
//     query: " 65.0ms". Blank where the count is 0.
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
//		Name:   "file_io_latency",
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

	"github.com/sjmudd/ps-top/v2/internal/anonymise"
	"github.com/sjmudd/ps-top/v2/internal/collector"
	"github.com/sjmudd/ps-top/v2/internal/derive"
	"github.com/sjmudd/ps-top/v2/internal/formatter"
	"github.com/sjmudd/ps-top/v2/internal/metric"
	"github.com/sjmudd/ps-top/v2/internal/registry"
	"github.com/sjmudd/ps-top/v2/internal/snapshot"
	"github.com/sjmudd/ps-top/v2/internal/transform"
)

// Source gives the snapshots of a metric that metric type t needs, over a
// window of n snapshots (see collector.Collector.Window).
type Source interface {
	Window(ctx context.Context, name string, t metric.MetricType, n int) (collector.Window, error)
}

// Renderer is a view of any metric.
type Renderer interface {
	// ViewName returns the view's name, e.g. "table_io_latency".
	ViewName() string
	// Metric returns the name of the metric the view shows.
	Metric() string
	// Style returns the style the view shows, "current" or "difference".
	Style() string
	// Accepts reports whether the view's metric can be shown in style t.
	Accepts(t metric.MetricType) bool
	// WithStyle returns the view showing style t: values up to now
	// (CurrentValue) or the difference over the window (DifferenceValues).
	// It fails if the view's metric does not accept t.
	WithStyle(t metric.MetricType) (Renderer, error)
	// WithStats returns the view recording its renders in st, its entry in
	// the metrics registry.
	WithStats(st *registry.Presenter) Renderer
	// Render returns the view's lines.
	Render(ctx context.Context, src Source, o Options) ([]string, error)
	// Table returns the view as cells, for a caller that draws them itself.
	Table(ctx context.Context, src Source, o Options) (Table, error)
}

// Options says how a view is rendered.
type Options struct {
	N          int  // window of N collect intervals, for DifferenceValues
	Width      int  // lines are cut at Width runes; 0: not cut
	Limit      int  // at most Limit rows, the highest first; 0: no limit. Totals are over all rows
	TotalsOnly bool // no rows: the title, headings and totals line only
	// Anonymiser, if set, anonymises the keys of views that have an
	// Anonymise function. Use one Anonymiser for a whole run, so a name is
	// shown the same way each time.
	Anonymiser *anonymise.Anonymiser
	// DatabaseFilter, if set, limits views that have a Schema function to
	// the rows of these schemas, and rows with no schema.
	DatabaseFilter []string
}

type cellKind int

const (
	value cellKind = iota
	shareOfTotal
	shareOf
	average
	key
)

// Cell is one cell of a line of a view of a metric with column type C. Build
// it with Key, Value, ShareOfTotal or ShareOf.
type Cell[C ~int] struct {
	Heading string
	Column  C                   // not used by Key
	Base    C                   // ShareOf: the column it is a share of; Average: the count it is divided by
	Format  func(uint64) string // Value and Average: formats the value shown
	Width   int
	kind    cellKind
	left    bool // the heading is left-aligned (see LeftHeading)
}

// LeftHeading returns c with its heading left-aligned in its width, as v1
// shows some headings, e.g. "Run Time  " above "    1m 44s".
func (c Cell[C]) LeftHeading() Cell[C] {
	c.left = true
	return c
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

// Average shows column's value divided by column count of the same row (an
// average per count, rounded down), formatted by format and right-aligned in
// width characters. Blank where count is 0. For example, a QUERY_TIMER of
// 812us over a QUERY_COUNT of 5 shows as 162.4us.
func Average[C ~int](heading string, column, count C, format func(uint64) string, width int) Cell[C] {
	return Cell[C]{Heading: heading, Column: column, Base: count, Format: format, Width: width, kind: average}
}

// View is a table over one metric with column type C.
type View[C ~int] struct {
	Name  string // e.g. "table_io_latency", as v1's view names
	Title string // e.g. "Table I/O Latency"
	Spec  metric.Spec[C]
	// MapKey returns the shown key of a raw key, or nil to show keys as
	// collected. For example, file_io_latency maps the file
	// /var/lib/mysql/sbtest/sbtest1.ibd to the table sbtest.sbtest1.
	MapKey func(string) string
	// Anonymise returns the shown key anonymised by a, used with
	// Options.Anonymiser set; nil: the key is always shown as it is.
	Anonymise func(a *anonymise.Anonymiser, key string) string
	// Schema returns the schema of a raw key, or "" if it has none, used
	// with Options.DatabaseFilter set; nil: rows are not filtered here. For
	// example, file_io_latency gives sbtest for
	// /var/lib/mysql/sbtest/sbtest1.ibd, so --database-filter=sbtest keeps
	// that file's row.
	Schema func(rawKey string) string
	Cells  [][]Cell[C] // groups of cells, separated by "|"
	// SortColumn orders the rows, highest first, then the ThenBy columns,
	// highest first, then the key. Rows where it is 0 are hidden unless
	// ShowZero is set.
	SortColumn C
	ThenBy     []C
	ShowZero   bool
	// From and Merge, if set, are for a view of Spec's columns built from
	// another metric: the view asks the collector for metric From, and
	// Merge turns each of its snapshots into one of Spec's columns, at the
	// map step, before filtering and MapKey. For example, ps_top_stages asks
	// for the stages metric, and Merge turns its entries
	// collect.table_io.query, collect.table_io.merge and
	// collect.table_io.snapshot into one entry, collect.table_io, with a
	// column group per stage.
	From  string
	Merge func(*snapshot.Snapshot) (*snapshot.Snapshot, error)
	// Groups, if set, are titles for the groups of Cells, shown centred on
	// a line above the headings, "" for none.
	Groups []string
	// NoTotals leaves out the totals line.
	NoTotals bool

	stats  *registry.Presenter // set by WithStats
	style  metric.MetricType   // set by WithStyle
	styled bool                // style is set; else the metric's default, the first of Spec.Types
}

// ViewName returns the view's name.
func (v View[C]) ViewName() string { return v.Name }

// Metric returns the name of the metric the view shows: From if set, else
// the Spec's.
func (v View[C]) Metric() string {
	if v.From != "" {
		return v.From
	}
	return v.Spec.Name
}

// Style returns the style the view shows: the one set by WithStyle, else
// the default type of its metric, the first of Spec.Types.
func (v View[C]) Style() string {
	t, ok := v.metricType()
	if !ok {
		return ""
	}
	return t.Name()
}

// metricType returns the type the view shows, and false if its metric has
// no types.
func (v View[C]) metricType() (metric.MetricType, bool) {
	if v.styled {
		return v.style, true
	}
	if len(v.Spec.Types) == 0 {
		return 0, false
	}
	return v.Spec.Types[0], true
}

// Accepts reports whether the view's metric can be shown in style t.
func (v View[C]) Accepts(t metric.MetricType) bool { return v.Spec.Definition().Accepts(t) }

// WithStyle returns a copy of v showing style t.
func (v View[C]) WithStyle(t metric.MetricType) (Renderer, error) {
	if !v.Accepts(t) {
		return nil, fmt.Errorf("presenter: view %s cannot show style %s", v.Name, t.Name())
	}
	v.style, v.styled = t, true
	return v, nil
}

// WithStats returns a copy of v recording its renders in st.
func (v View[C]) WithStats(st *registry.Presenter) Renderer {
	v.stats = st
	return v
}

// Render returns the view's lines (see Table), each cut at o.Width runes.
func (v View[C]) Render(ctx context.Context, src Source, o Options) ([]string, error) {
	t, err := v.Table(ctx, src, o)
	if err != nil {
		return nil, err
	}
	return t.Lines(o.Width), nil
}

// Table returns the view as cells: the title, headings, one row per key with
// a non-zero SortColumn (or every key with ShowZero) in sort order, at most
// o.Limit if set, and the totals. With o.TotalsOnly there are no key rows.
// The view's style (see WithStyle) says what is shown: for CurrentValue the
// values of the latest snapshot as they are, for DifferenceValues the
// difference over a window of o.N snapshots. The title ends with the style:
// "current", or "last 10s" with the time the window covers. A DistinctPart
// column's total is the distinct count over the latest snapshot's raw keys,
// not the sum of the column. o.Width is not used: lines are cut by
// Table.Lines, or by whoever draws the cells.
func (v View[C]) Table(ctx context.Context, src Source, o Options) (Table, error) {
	st := v.stats
	if st == nil {
		st = new(registry.Presenter) // not in any registry
	}
	defer st.Render.Start().Stop()

	if err := v.check(); err != nil {
		return Table{}, err
	}
	title := v.Title
	if table := v.Spec.TableName(); table != "" {
		title += " (" + table + ")"
	}
	def := v.Spec.Definition()
	style, ok := v.metricType()
	if !ok {
		return Table{}, fmt.Errorf("presenter: view %q: metric %q has no types", v.Title, def.Name)
	}
	t := st.Window.Start()
	w, err := src.Window(ctx, v.Metric(), style, o.N)
	t.Stop()
	if err != nil {
		return Table{}, err
	}
	if !w.Ready() {
		return Table{Title: title + " (waiting for data)", Waiting: true}, nil
	}
	// The map step, timed as one run per snapshot: Merge, the database
	// filter, then the key mapping. raw holds the snapshots before the key
	// mapping, mapped after it.
	raw := make([]*snapshot.Snapshot, len(w.Snapshots))
	mapped := make([]*snapshot.Snapshot, len(w.Snapshots))
	for i, s := range w.Snapshots {
		t := st.Map.Start()
		raw[i], mapped[i], err = v.mapSnapshot(s, def, o.DatabaseFilter)
		t.Stop()
		if err != nil {
			return Table{}, err
		}
		st.EntriesMapped.Add(uint64(len(raw[i].Entries)))
	}
	t = st.Derive.Start()
	d, err := v.derive(def, w.Type, raw, mapped)
	t.Stop()
	if err != nil {
		return Table{}, err
	}

	t = st.Order.Start()
	shown, names, rows := v.order(d.Entries, o)
	t.Stop()
	st.RowsShown.Store(uint64(len(shown)))
	st.RowsTotal.Store(uint64(len(d.Entries)))

	t = st.Draw.Start()
	tab := v.draw(title, rows, w.Type, d, shown, names)
	t.Stop()
	st.LinesDrawn.Add(uint64(len(tab.Lines(0))))
	return tab, nil
}

// derive returns the values to show from the mapped snapshots: those of the
// one snapshot for a current view, the difference between the first and the
// last for a difference view. A DistinctPart column's total is counted over
// the latest raw snapshot instead. For example, an entry at 10 at the start
// of the window and at 68 at its end shows 58.
func (v View[C]) derive(def metric.Definition, t metric.MetricType, raw, mapped []*snapshot.Snapshot) (derive.Difference, error) {
	var d derive.Difference
	var err error
	switch t {
	case metric.CurrentValue:
		d, err = derive.Current(def, mapped[0])
	case metric.DifferenceValues:
		d, err = derive.Window(def, mapped[0], mapped[1])
	default:
		err = fmt.Errorf("presenter: view %q: metric type %v not handled", v.Title, t)
	}
	if err != nil {
		return derive.Difference{}, err
	}
	for c, total := range transform.DistinctTotals(def, raw[len(raw)-1]) {
		d.Totals[c] = total
	}
	return d, nil
}

// order returns the entries to show, their names as shown and the row count
// for the title. Entries where SortColumn is 0 are left out unless ShowZero
// is set, the rest are sorted by SortColumn, then ThenBy, highest first, then
// by key, and cut to o.Limit. For example, table_io_latency with db.a and
// db.b and --limit=1 shows db.a, the higher, and "1 of 2 rows".
func (v View[C]) order(entries []derive.Entry, o Options) (shown []derive.Entry, names []string, rows string) {
	by := append([]C{v.SortColumn}, v.ThenBy...)
	for _, e := range entries {
		if v.ShowZero || e.Values[v.SortColumn] > 0 {
			shown = append(shown, e)
		}
	}
	slices.SortFunc(shown, func(a, b derive.Entry) int {
		for _, col := range by {
			if c := cmp.Compare(b.Values[col], a.Values[col]); c != 0 {
				return c
			}
		}
		return cmp.Compare(a.Key.Value(), b.Key.Value())
	})
	rows = fmt.Sprintf("%d rows", len(shown))
	if len(shown) == 1 {
		rows = "1 row"
	}
	if o.Limit > 0 && len(shown) > o.Limit {
		rows = fmt.Sprintf("%d of %d rows", o.Limit, len(shown))
		shown = shown[:o.Limit]
	}
	if o.TotalsOnly {
		shown = nil
	}
	names = make([]string, len(shown))
	for i, e := range shown {
		names[i] = v.name(e.Key.Value(), o.Anonymiser)
	}
	return shown, names, rows
}

// draw returns the table of shown: the title with the row count and the
// style, the headings, a row per entry with its name, and the totals unless
// NoTotals is set. For example, the title "ps-top stages 1 row, last 10s".
func (v View[C]) draw(title, rows string, t metric.MetricType, d derive.Difference, shown []derive.Entry, names []string) Table {
	covers := styleText(t, d)
	tab := Table{
		Title:    fmt.Sprintf("%s %s, %s", title, rows, covers),
		Style:    covers,
		Groups:   v.Groups,
		Headings: v.headingGroups(),
	}
	if t == metric.DifferenceValues {
		tab.Covers = d.Elapsed()
	}
	if !v.NoTotals {
		tab.Totals = v.rowGroups(d.Totals, d.Totals, "Totals")
	}
	for i, e := range shown {
		tab.Rows = append(tab.Rows, v.rowGroups(e.Values, d.Totals, names[i]))
	}
	return tab
}

// mapSnapshot returns s merged by Merge if the view has one, filtered to
// schemas if any are given and the view has Schema, and that snapshot with
// its keys mapped by MapKey. For example, file_io_latency with
// --database-filter=sbtest keeps /var/lib/mysql/sbtest/sbtest1.ibd and maps
// it to sbtest.sbtest1. ps_top_stages merges collect.table_io.query, .merge
// and .snapshot into collect.table_io and has no filter or MapKey.
func (v View[C]) mapSnapshot(s *snapshot.Snapshot, def metric.Definition, schemas []string) (raw, mapped *snapshot.Snapshot, err error) {
	raw = s
	if v.Merge != nil {
		if raw, err = v.Merge(raw); err != nil {
			return nil, nil, fmt.Errorf("presenter: view %q: %w", v.Title, err)
		}
	}
	if len(schemas) > 0 && v.Schema != nil {
		if raw, err = v.filter(raw, schemas); err != nil {
			return nil, nil, err
		}
	}
	if mapped, err = transform.Map(def, raw, v.MapKey); err != nil {
		return nil, nil, err
	}
	return raw, mapped, nil
}

// name returns a row's key as shown: anonymised by the view's Anonymise
// function if it has one and a is set, else its parts joined with ".".
func (v View[C]) name(key string, a *anonymise.Anonymiser) string {
	if a != nil && v.Anonymise != nil {
		return v.Anonymise(a, key)
	}
	return snapshot.KeyText(key)
}

// filter returns s with only the entries whose schema (v.Schema) is one of
// schemas, or empty.
func (v View[C]) filter(s *snapshot.Snapshot, schemas []string) (*snapshot.Snapshot, error) {
	var kept []snapshot.Entry
	for _, e := range s.Entries {
		if schema := v.Schema(e.Key.Value()); schema == "" || slices.Contains(schemas, schema) {
			kept = append(kept, e)
		}
	}
	return snapshot.New(s.Time, s.Width, kept)
}

// styleText describes what a render of type t shows: "current", or "last
// 10s" with the time d's window covers, which is shorter than asked for
// while the window is not yet full. The time is in whole seconds, rounded:
// intervals are whole seconds.
func styleText(t metric.MetricType, d derive.Difference) string {
	if t == metric.DifferenceValues {
		return fmt.Sprintf("last %.0fs", d.Elapsed().Seconds())
	}
	return t.Name()
}

// check returns an error if SortColumn, a ThenBy column or a cell's column is
// not one of the Spec's columns.
func (v View[C]) check() error {
	cols := append([]C{v.SortColumn}, v.ThenBy...)
	for _, cells := range v.Cells {
		for _, c := range cells {
			switch c.kind {
			case value, shareOfTotal:
				cols = append(cols, c.Column)
			case shareOf, average:
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

// headingGroups returns the headings as cells, one group per group of
// cells.
func (v View[C]) headingGroups() []TableGroup {
	groups := make([]TableGroup, len(v.Cells))
	for g, cells := range v.Cells {
		for _, c := range cells {
			if c.kind == key {
				groups[g] = append(groups[g], TableCell{Text: keyText(c.Heading, c.Width), Role: RoleHeading})
				continue
			}
			text := fmt.Sprintf("%*s", c.Width, c.Heading)
			if c.left {
				text = fmt.Sprintf("%-*s", c.Width, c.Heading)
			}
			groups[g] = append(groups[g], TableCell{Text: text, Role: RoleHeading})
		}
	}
	return groups
}

// rowGroups returns one row, named name, as cells, from its values and the
// column totals.
func (v View[C]) rowGroups(values, totals []uint64, name string) []TableGroup {
	groups := make([]TableGroup, len(v.Cells))
	for g, cells := range v.Cells {
		for _, c := range cells {
			var s string
			role := RoleValue
			switch c.kind {
			case key:
				groups[g] = append(groups[g], TableCell{Text: keyText(name, c.Width), Role: RoleKey})
				continue
			case value:
				s = c.Format(values[c.Column])
			case shareOfTotal:
				s, role = formatter.Pct(formatter.Divide(values[c.Column], totals[c.Column])), RoleShare
			case shareOf:
				s, role = formatter.Pct(formatter.Divide(values[c.Column], values[c.Base])), RoleShare
			case average:
				if n := values[c.Base]; n > 0 {
					s = c.Format(values[c.Column] / n)
				}
			}
			groups[g] = append(groups[g], TableCell{Text: fmt.Sprintf("%*s", c.Width, s), Role: role})
		}
	}
	return groups
}

// headings returns the headings line.
func (v View[C]) headings() string { return groupsText(v.headingGroups()) }

// keyText returns s as it is if width is 0, else left-aligned in width runes
// and cut to width.
func keyText(s string, width int) string {
	if width == 0 {
		return s
	}
	return fmt.Sprintf("%-*s", width, cut(s, width))
}

// cut returns s limited to width runes, or s as it is if width is 0.
func cut(s string, width int) string {
	if width <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width])
}

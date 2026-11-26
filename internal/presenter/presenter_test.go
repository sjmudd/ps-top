package presenter

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"unique"

	"github.com/sjmudd/ps-top/v2/internal/anonymise"
	"github.com/sjmudd/ps-top/v2/internal/collector"
	"github.com/sjmudd/ps-top/v2/internal/metric"
	"github.com/sjmudd/ps-top/v2/internal/metrics/tableio"
	"github.com/sjmudd/ps-top/v2/internal/metrics/userlatency"
	"github.com/sjmudd/ps-top/v2/internal/registry"
	"github.com/sjmudd/ps-top/v2/internal/snapshot"
)

// fake returns fixed snapshots: Start and End for DifferenceValues, End
// alone for CurrentValue, none if not OK.
type fake struct {
	Start, End *snapshot.Snapshot
	OK         bool
}

func (f fake) Window(_ context.Context, _ string, t metric.MetricType, _ int) (collector.Window, error) {
	w := collector.Window{Type: t}
	switch {
	case !f.OK:
	case t == metric.CurrentValue:
		w.Snapshots = []*snapshot.Snapshot{f.End}
	default:
		w.Snapshots = []*snapshot.Snapshot{f.Start, f.End}
	}
	return w, nil
}

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// entry builds a table I/O entry from count_star and sum_timer_wait, with all
// reads being fetches.
func entry(key string, count, wait uint64) snapshot.Entry {
	v := make([]uint64, 14)
	v[0], v[1] = count, wait // star
	v[2], v[3] = count, wait // read
	v[6], v[7] = count, wait // fetch
	return snapshot.Entry{Key: unique.Make(key), Values: v}
}

func snap(t *testing.T, at time.Duration, entries ...snapshot.Entry) *snapshot.Snapshot {
	t.Helper()
	s, err := snapshot.New(t0.Add(at), 14, entries)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNameAnonymised(t *testing.T) {
	a := anonymise.New()
	file := FileIOLatency(nil)
	tests := []struct {
		name string
		v    interface {
			name(string, *anonymise.Anonymiser) string
		}
		key  string
		a    *anonymise.Anonymiser
		want string
	}{
		{"no anonymiser", TableIOLatency, snapshot.JoinKey("shop", "orders"), nil, "shop.orders"},
		{"table", TableIOLatency, snapshot.JoinKey("shop", "orders"), a, "schema1.table1"},
		{"same schema, other table", TableIOOps, snapshot.JoinKey("shop", "lines"), a, "schema1.table2"},
		{"file view shares the groups", file, snapshot.JoinKey("crm", "orders"), a, "schema2.table1"},
		{"file label kept", file, "<redo_log>", a, "<redo_log>"},
		{"path kept", file, "<datadir>/notes", a, "<datadir>/notes"},
		{"user", UserLatency, "app", a, "user1"},
		{"view without Anonymise", MutexLatency, "innodb/buf_pool_mutex", a, "innodb/buf_pool_mutex"},
	}
	for _, tt := range tests {
		if got := tt.v.name(tt.key, tt.a); got != tt.want {
			t.Errorf("%s: name(%q) = %q, want %q", tt.name, tt.key, got, tt.want)
		}
	}
}

// TestRenderAnonymised checks rows are anonymised, the same way over renders
// with the same Anonymiser, and stay in the order of the real names.
func TestRenderAnonymised(t *testing.T) {
	k := snapshot.JoinKey
	src := fake{
		Start: snap(t, 0, entry(k("shop", "b"), 0, 0), entry(k("shop", "a"), 0, 0), entry(k("crm", "c"), 0, 0)),
		End:   snap(t, time.Second, entry(k("shop", "b"), 5, 1000), entry(k("shop", "a"), 5, 1000), entry(k("crm", "c"), 9, 9000)),
		OK:    true,
	}
	a := anonymise.New()
	keys := func() []string {
		lines, err := TableIOLatency.Render(context.Background(), src, Options{N: 5, Width: 200, Anonymiser: a})
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, l := range lines[2 : len(lines)-1] {
			got = append(got, l[strings.LastIndexByte(l, '|')+1:])
		}
		return got
	}
	// crm.c has the most wait; shop.a and shop.b tie and are ordered by name.
	want := []string{"schema1.table1", "schema2.table2", "schema2.table3"}
	for i := range 2 {
		if got := keys(); !slices.Equal(got, want) {
			t.Errorf("render %d: keys %q, want %q", i+1, got, want)
		}
	}
}

func TestHeadingsMatchOldFormat(t *testing.T) {
	want := fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s|%s", "Latency", "%", "Read", "Write", "Fetch", "Insert", "Update", "Delete", "Table Name")
	if got := TableIOLatency.headings(); got != want {
		t.Errorf("headings\n got %q\nwant %q", got, want)
	}
}

func TestRender(t *testing.T) {
	src := fake{
		Start: snap(t, 0, entry("db.a", 10, 1000000000), entry("db.b", 5, 0), entry("db.idle", 1, 5)),
		End:   snap(t, time.Second, entry("db.a", 30, 4000000000), entry("db.b", 15, 1000000000), entry("db.idle", 1, 5)),
		OK:    true,
	}
	lines, err := TableIOLatency.Render(context.Background(), src, Options{N: 5, Width: 80})
	if err != nil {
		t.Fatal(err)
	}
	// description, headings, db.a, db.b, totals; db.idle has no change.
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if lines[0] != "Table I/O Latency (table_io_waits_summary_by_table) 2 rows, last 1s" {
		t.Errorf("description = %q", lines[0])
	}
	want := fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s|%s", "   3.00 ms", " 75.0%", "100.0%", "", "100.0%", "", "", "", "db.a")
	if lines[2] != want {
		t.Errorf("first line\n got %q\nwant %q", lines[2], want)
	}
	if !strings.HasSuffix(lines[3], "|db.b") || !strings.HasSuffix(lines[4], "|Totals") {
		t.Errorf("order: %q, %q", lines[3], lines[4])
	}
	if !strings.HasPrefix(lines[4], "   4.00 ms 100.0%") {
		t.Errorf("totals = %q", lines[4])
	}
}

// TestRenderStyles checks a counter view shown as current values shows the
// latest snapshot as it is, and a style the metric does not accept is
// refused.
func TestRenderStyles(t *testing.T) {
	src := fake{
		Start: snap(t, 0, entry("db.a", 10, 1000000000), entry("db.b", 5, 0)),
		End:   snap(t, time.Second, entry("db.a", 30, 4000000000), entry("db.b", 15, 1000000000)),
		OK:    true,
	}
	v, err := TableIOLatency.WithStyle(metric.CurrentValue)
	if err != nil {
		t.Fatal(err)
	}
	if v.Style() != "current" || TableIOLatency.Style() != "difference" {
		t.Errorf("styles %q and %q, want current and the default difference", v.Style(), TableIOLatency.Style())
	}
	lines, err := v.Render(context.Background(), src, Options{N: 5, Width: 80})
	if err != nil {
		t.Fatal(err)
	}
	// Up to now: db.a 4 ms of 5 ms in all, not the window's 3 of 4.
	if lines[0] != "Table I/O Latency (table_io_waits_summary_by_table) 2 rows, current" ||
		!strings.HasPrefix(lines[2], "   4.00 ms  80.0%") || !strings.HasPrefix(lines[4], "   5.00 ms 100.0%") {
		t.Errorf("current lines:\n%s", strings.Join(lines, "\n"))
	}
	if _, err := UserLatency.WithStyle(metric.DifferenceValues); err == nil {
		t.Error("user latency as difference: no error")
	}
	if v, err := MemoryUsage.WithStyle(metric.DifferenceValues); err != nil || MemoryUsage.Style() != "current" || v.Style() != "difference" {
		t.Errorf("memory usage: default %q, as difference %v", MemoryUsage.Style(), err)
	}
}

// TestRenderLimitAndTotals checks Limit keeps the highest rows with totals
// over all rows, and TotalsOnly leaves out the rows.
func TestRenderLimitAndTotals(t *testing.T) {
	src := fake{
		Start: snap(t, 0, entry("db.a", 10, 1000000000), entry("db.b", 5, 0)),
		End:   snap(t, time.Second, entry("db.a", 30, 4000000000), entry("db.b", 15, 1000000000)),
		OK:    true,
	}
	lines, err := TableIOLatency.Render(context.Background(), src, Options{N: 5, Width: 80, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 4 || lines[0] != "Table I/O Latency (table_io_waits_summary_by_table) 1 of 2 rows, last 1s" ||
		!strings.HasSuffix(lines[2], "|db.a") || !strings.HasPrefix(lines[3], "   4.00 ms 100.0%") {
		t.Errorf("limit 1:\n%s", strings.Join(lines, "\n"))
	}
	lines, err = TableIOLatency.Render(context.Background(), src, Options{N: 5, Width: 80, TotalsOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 || !strings.HasSuffix(lines[2], "|Totals") {
		t.Errorf("totals only:\n%s", strings.Join(lines, "\n"))
	}
}

// TestRenderStats checks a render records its stages in the view's registry
// entry, counters added to on each render, gauges holding the last render.
func TestRenderStats(t *testing.T) {
	src := fake{
		Start: snap(t, 0, entry("db.a", 10, 1000000000), entry("db.b", 5, 0), entry("db.idle", 1, 5)),
		End:   snap(t, time.Second, entry("db.a", 30, 4000000000), entry("db.b", 15, 1000000000), entry("db.idle", 1, 5)),
		OK:    true,
	}
	st := new(registry.Presenter)
	v := TableIOLatency.WithStats(st)
	if v.ViewName() != "table_io_latency" || v.Style() != "difference" {
		t.Errorf("view %q style %q", v.ViewName(), v.Style())
	}
	for range 2 {
		if _, err := v.Render(context.Background(), src, Options{N: 5, Width: 80}); err != nil {
			t.Fatal(err)
		}
	}
	for name, s := range map[string]*registry.Stage{
		"RENDER": &st.Render, "WINDOW": &st.Window, "DERIVE": &st.Derive, "ORDER": &st.Order, "DRAW": &st.Draw,
	} {
		if got := s.Count.Load(); got != 2 {
			t.Errorf("COUNT_%s = %d, want 2", name, got)
		}
		if s.SumTimer.Load() == 0 {
			t.Errorf("SUM_TIMER_%s = 0, want the time taken", name)
		}
	}
	// Two snapshots of three entries mapped per render.
	if got := st.Map.Count.Load(); got != 4 {
		t.Errorf("COUNT_MAP = %d, want 4", got)
	}
	if got := st.EntriesMapped.Load(); got != 12 {
		t.Errorf("SUM_ENTRIES_MAPPED = %d, want 12", got)
	}
	// Five lines per render. db.idle has no change, so 2 of 3 rows shown.
	if got := st.LinesDrawn.Load(); got != 10 {
		t.Errorf("SUM_LINES_DRAWN = %d, want 10", got)
	}
	if st.RowsShown.Load() != 2 || st.RowsTotal.Load() != 3 {
		t.Errorf("CURRENT_ROWS_SHOWN %d of CURRENT_ROWS_TOTAL %d, want 2 of 3", st.RowsShown.Load(), st.RowsTotal.Load())
	}
}

// TestUserLatency renders the user view from processlist rows through the
// metric's rules: one row per user, every user shown, ordered by total time
// then connections, and Hosts and DBs totals distinct, not summed.
func TestUserLatency(t *testing.T) {
	rows := [][]string{
		{"21", "app", "10.0.0.5:1", "shop", "Sleep", "100", "", ""},
		{"22", "app", "10.0.0.6:2", "shop", "Query", "4", "executing", "SELECT 1"},
		{"23", "app", "10.0.0.6:3", "", "Query", "2", "executing", "CALL p()"},
		{"31", "report", "10.0.0.6:4", "stats", "Query", "70", "executing", "select * from big"},
		{"32", "report", "10.0.0.6:5", "shop", "Sleep", "5", "", ""},
		{"41", "monitor", "10.0.0.7:6", "", "Sleep", "0", "", ""},
		{"51", "repl", "10.0.0.8:7", "", "Binlog Dump GTID", "500", "Source has sent all binlog to replica; waiting for more updates", ""},
		{"61", "system user", "", "", "Connect", "500", "Waiting for source to send event", ""},
	}
	var entries []snapshot.Entry
	for _, f := range rows {
		key, values := userlatency.Row(f)
		entries = append(entries, snapshot.Entry{Key: unique.Make(snapshot.JoinKey(key...)), Values: values})
	}
	s, err := snapshot.New(t0, len(userlatency.Spec.Columns), entries)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := UserLatency.Render(context.Background(), fake{End: s, OK: true}, Options{N: 1, Width: 200})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Activity by Username (processlist) 5 rows, current",
		"Run Time        %|Sleeping        %|Conn Actv Repl|Hosts DBs|Sel Ins Upd Del Other|User",
		// app: 106 s in all, report: 75 s, then by connections and name.
		"        6s   7.9%|    1m 40s  95.2%|   3    2     |    2   1|  1                 1|app",
		"    1m 10s  92.1%|        5s   4.8%|   2    1     |    1   2|  1                  |report",
		"                 |                 |   1          |    1    |                     |monitor",
		"                 |                 |   1         1|    1    |                     |repl",
		"                 |                 |   1         1|         |                     |system user",
		// 4 distinct hosts, where the column sums to 5. 2 distinct dbs (shop,
		// stats), where it sums to 3.
		"    1m 16s 100.0%|    1m 45s 100.0%|   8    3    2|    4   2|  2                 1|Totals",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

// TestTable checks the cells of a view: their roles, the rows in order, and
// Lines giving what Render gives. Waiting has the title only.
func TestTable(t *testing.T) {
	k := snapshot.JoinKey
	src := fake{
		Start: snap(t, 0, entry(k("shop", "a"), 0, 0), entry(k("shop", "b"), 0, 0)),
		End:   snap(t, time.Second, entry(k("shop", "a"), 1, 3000), entry(k("shop", "b"), 1, 1000)),
		OK:    true,
	}
	o := Options{N: 5, Width: 60}
	tab, err := TableIOLatency.Table(context.Background(), src, o)
	if err != nil {
		t.Fatal(err)
	}
	if len(tab.Rows) != 2 || tab.Waiting {
		t.Fatalf("%d rows, waiting %v: want 2, false", len(tab.Rows), tab.Waiting)
	}
	first := tab.Rows[0]
	last := first[len(first)-1]
	if got := first[0][0]; got.Role != RoleValue || got.Text != "   3.00 ns" { // 3000 ps
		t.Errorf("first cell %+v, want the value 3.00 ns", got)
	}
	if got := first[0][1]; got.Role != RoleShare || got.Text != " 75.0%" {
		t.Errorf("second cell %+v, want the share 75.0%%", got)
	}
	if last[0].Role != RoleKey || last[0].Text != "shop.a" {
		t.Errorf("key cell %+v, want shop.a", last[0])
	}
	if got := tab.Totals[len(tab.Totals)-1][0]; got.Role != RoleKey || got.Text != "Totals" {
		t.Errorf("totals key cell %+v", got)
	}
	if h := tab.Headings[0][0]; h.Role != RoleHeading || strings.TrimSpace(h.Text) != "Latency" {
		t.Errorf("first heading %+v", h)
	}
	lines, err := TableIOLatency.Render(context.Background(), src, o)
	if err != nil {
		t.Fatal(err)
	}
	if got := tab.Lines(o.Width); !slices.Equal(got, lines) {
		t.Errorf("Lines\n%s\nRender\n%s", strings.Join(got, "\n"), strings.Join(lines, "\n"))
	}

	waiting, err := TableIOLatency.Table(context.Background(), fake{}, o)
	if err != nil {
		t.Fatal(err)
	}
	if got := waiting.Lines(0); !waiting.Waiting || len(got) != 1 || !strings.HasSuffix(got[0], "(waiting for data)") {
		t.Errorf("waiting: %v %q", waiting.Waiting, got)
	}
}

// TestTableBlank checks an empty row keeps the headings' "|" and nothing
// after the last, and is empty while waiting.
func TestTableBlank(t *testing.T) {
	tab := Table{Headings: []TableGroup{
		{{Text: "   Latency"}, {Text: "     %"}},
		{{Text: "  Read"}},
		{{Text: "Table Name"}},
	}}
	if got, want := tab.Blank(), "                 |      |"; got != want {
		t.Errorf("Blank() = %q, want %q", got, want)
	}
	if got := (Table{Headings: tab.Headings, Waiting: true}).Blank(); got != "" {
		t.Errorf("waiting: Blank() = %q, want empty", got)
	}
}

// shownKeys returns the key of each row line of a view's lines: the text
// after the last "|", without the title, headings and totals lines.
func shownKeys(lines []string) []string {
	var keys []string
	for _, l := range lines[2 : len(lines)-1] {
		keys = append(keys, l[strings.LastIndexByte(l, '|')+1:])
	}
	return keys
}

// TestRenderDatabaseFilter checks the views filtered in presentation show
// only rows of the schemas given, and rows with no schema: connections with
// no db, file labels. The user view's distinct totals count only the
// connections kept. The table view has no Schema function: its query is
// filtered, so its rows are shown as they come.
func TestRenderDatabaseFilter(t *testing.T) {
	filter := []string{"shop"}

	rows := [][]string{
		{"21", "app", "10.0.0.5:1", "shop", "Sleep", "100", "", ""},
		{"22", "app", "10.0.0.6:2", "stats", "Query", "4", "executing", "SELECT 1"},
		{"31", "report", "10.0.0.6:4", "stats", "Query", "70", "executing", "select * from big"},
		{"41", "monitor", "10.0.0.7:6", "", "Sleep", "0", "", ""},
	}
	var conns []snapshot.Entry
	for _, f := range rows {
		key, values := userlatency.Row(f)
		conns = append(conns, snapshot.Entry{Key: unique.Make(snapshot.JoinKey(key...)), Values: values})
	}
	s, err := snapshot.New(t0, len(userlatency.Spec.Columns), conns)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		filter []string
		users  []string
		totals string // the Conn and Hosts DBs groups of the totals line
	}{
		{nil, []string{"app", "report", "monitor"}, "4 3 2"},
		// app's stats connection and report go: one db (shop), two hosts.
		{filter, []string{"app", "monitor"}, "2 2 1"},
	} {
		lines, err := UserLatency.Render(context.Background(), fake{End: s, OK: true}, Options{N: 1, Width: 200, DatabaseFilter: tt.filter})
		if err != nil {
			t.Fatal(err)
		}
		if got := shownKeys(lines); !slices.Equal(got, tt.users) {
			t.Errorf("filter %q: users %q, want %q", tt.filter, got, tt.users)
		}
		g := strings.Split(lines[len(lines)-1], "|")
		if got := strings.Fields(g[2])[0] + " " + strings.Join(strings.Fields(g[3]), " "); got != tt.totals {
			t.Errorf("filter %q: totals Conn, Hosts, DBs %q, want %q", tt.filter, got, tt.totals)
		}
	}

	files := map[string]string{
		"/data/shop/orders.ibd": snapshot.JoinKey("shop", "orders"),
		"/data/crm/leads.ibd":   snapshot.JoinKey("crm", "leads"),
		"/data/ib_logfile0":     "<redo_log>",
	}
	fileEntry := func(name string, wait uint64) snapshot.Entry {
		v := make([]uint64, 10)
		v[0], v[1] = 1, wait // count_star, sum_timer_wait
		return snapshot.Entry{Key: unique.Make(name), Values: v}
	}
	mk := func(at time.Duration, scale uint64) *snapshot.Snapshot {
		s, err := snapshot.New(t0.Add(at), 10, []snapshot.Entry{
			fileEntry("/data/shop/orders.ibd", 3*scale),
			fileEntry("/data/crm/leads.ibd", 2*scale),
			fileEntry("/data/ib_logfile0", 1*scale),
		})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	view := FileIOLatency(func(name string) string { return files[name] })
	lines, err := view.Render(context.Background(), fake{Start: mk(0, 0), End: mk(time.Second, 1000), OK: true}, Options{N: 5, Width: 200, DatabaseFilter: filter})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shownKeys(lines), []string{"shop.orders", "<redo_log>"}; !slices.Equal(got, want) {
		t.Errorf("file view: keys %q, want %q", got, want)
	}

	k := snapshot.JoinKey
	src := fake{
		Start: snap(t, 0, entry(k("shop", "a"), 0, 0), entry(k("crm", "b"), 0, 0)),
		End:   snap(t, time.Second, entry(k("shop", "a"), 1, 2000), entry(k("crm", "b"), 1, 1000)),
		OK:    true,
	}
	lines, err = TableIOLatency.Render(context.Background(), src, Options{N: 5, Width: 200, DatabaseFilter: filter})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := shownKeys(lines), []string{"shop.a", "crm.b"}; !slices.Equal(got, want) {
		t.Errorf("table view: keys %q, want %q (filtered in its query, not here)", got, want)
	}
}

// TestFileIOLatencyDocExample checks the example in the package documentation.
func TestFileIOLatencyDocExample(t *testing.T) {
	// count_star, sum_timer_wait, count_read, sum_timer_read, count_write,
	// sum_timer_write, count_misc, sum_timer_misc, bytes read, bytes written
	orders := []uint64{400, 8000000000, 100, 2000000000, 250, 5000000000, 50, 1000000000, 1572864, 655360}
	other := []uint64{10, 12000000000, 10, 12000000000, 0, 0, 0, 0, 0, 0}
	zero := make([]uint64, 10)
	mk := func(at time.Duration, a, b []uint64) *snapshot.Snapshot {
		s, err := snapshot.New(t0.Add(at), 10, []snapshot.Entry{
			{Key: unique.Make("shop.orders"), Values: a},
			{Key: unique.Make("shop.other"), Values: b},
		})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	src := fake{Start: mk(0, zero, zero), End: mk(time.Second, orders, other), OK: true}
	lines, err := FileIOLatency(nil).Render(context.Background(), src, Options{N: 5, Width: 200})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range map[int]string{
		1: "   Latency      %|  Read  Write   Misc|Rd bytes Wr bytes|     Ops  R Ops  W Ops  M Ops|Table Name",
		3: "   8.00 ms  40.0%| 25.0%  62.5%  12.5%|  1.50 M 640.00 k|     400  25.0%  62.5%  12.5%|shop.orders",
	} {
		if lines[i] != want {
			t.Errorf("line %d\n got %q\nwant %q", i, lines[i], want)
		}
	}
}

// TestKeyFirst checks a key placed first with a width: padded or cut, so the
// cells after it line up.
func TestKeyFirst(t *testing.T) {
	v := TableIOOps
	v.Cells = append([][]Cell[tableio.Column]{{Key[tableio.Column]("Table", 6)}}, v.Cells[0])
	src := fake{
		Start: snap(t, 0, entry("db.a", 0, 0), entry("db.long_name", 0, 0)),
		End:   snap(t, time.Second, entry("db.a", 30, 1), entry("db.long_name", 10, 1)),
		OK:    true,
	}
	lines, err := v.Render(context.Background(), src, Options{N: 5, Width: 80})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{
		"Table |       Ops      %",
		"db.a  |        30  75.0%",
		"db.lon|        10  25.0%",
		"Totals|        40 100.0%",
	} {
		if lines[i+1] != want {
			t.Errorf("line %d\n got %q\nwant %q", i+1, lines[i+1], want)
		}
	}
}

// eventSnap builds a snapshot of event name entries: count_star, sum_timer_wait.
func eventSnap(t *testing.T, at time.Duration, values map[string][2]uint64) *snapshot.Snapshot {
	t.Helper()
	var entries []snapshot.Entry
	for k, v := range values {
		entries = append(entries, snapshot.Entry{Key: unique.Make(k), Values: []uint64{v[0], v[1]}})
	}
	s, err := snapshot.New(t0.Add(at), 2, entries)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestMutexLatencyMatchesOldFormat(t *testing.T) {
	const a, b = "wait/synch/mutex/innodb/trx_sys_mutex", "wait/synch/sxlock/innodb/hash_table_locks"
	src := fake{
		Start: eventSnap(t, 0, map[string][2]uint64{a: {0, 0}, b: {0, 0}}),
		End:   eventSnap(t, time.Second, map[string][2]uint64{a: {30, 3000000000}, b: {2000, 1000000000}}),
		OK:    true,
	}
	lines, err := MutexLatency.Render(context.Background(), src, Options{N: 5, Width: 100})
	if err != nil {
		t.Fatal(err)
	}
	old := "%10s %8s %8s|%s" // v1's presenter/mutexlatency
	for i, want := range map[int]string{
		1: fmt.Sprintf(old, "Latency", "MtxCnt", "%", "Mutex Name"),
		2: fmt.Sprintf(old, "   3.00 ms", "30", " 75.0%", "mutex/innodb/trx_sys_mutex"),
		3: fmt.Sprintf(old, "   1.00 ms", "  1.95 k", " 25.0%", "sxlock/innodb/hash_table_locks"),
	} {
		if lines[i] != want {
			t.Errorf("line %d\n got %q\nwant %q", i, lines[i], want)
		}
	}
}

func TestStagesLatencyMatchesOldFormat(t *testing.T) {
	const a, b = "stage/sql/Sending data", "stage/innodb/buffer pool load"
	src := fake{
		Start: eventSnap(t, 0, map[string][2]uint64{a: {0, 0}, b: {0, 0}}),
		End:   eventSnap(t, time.Second, map[string][2]uint64{a: {30, 3000000000}, b: {1, 1000000000}}),
		OK:    true,
	}
	lines, err := StagesLatency.Render(context.Background(), src, Options{N: 5, Width: 100})
	if err != nil {
		t.Fatal(err)
	}
	old := "%10s %6s %8s|%s" // v1's presenter/stageslatency
	for i, want := range map[int]string{
		1: fmt.Sprintf(old, "Latency", "%", "Counter", "Stage Name"),
		2: fmt.Sprintf(old, "   3.00 ms", " 75.0%", "30", "Sending data"),
		3: fmt.Sprintf(old, "   1.00 ms", " 25.0%", "1", "stage/innodb/buffer pool load"),
	} {
		if lines[i] != want {
			t.Errorf("line %d\n got %q\nwant %q", i, lines[i], want)
		}
	}
}

// TestMemoryUsageMatchesOldFormat checks a CurrentValue view: values are
// shown as they are in the one snapshot sent, not as a difference.
func TestMemoryUsageMatchesOldFormat(t *testing.T) {
	// CURRENT_COUNT_USED, HIGH_COUNT_USED, CURRENT_NUMBER_OF_BYTES_USED,
	// HIGH_NUMBER_OF_BYTES_USED, MEM_OPS
	mk := func(values map[string][]uint64) *snapshot.Snapshot {
		var entries []snapshot.Entry
		for k, v := range values {
			entries = append(entries, snapshot.Entry{Key: unique.Make(k), Values: v})
		}
		s, err := snapshot.New(t0, 5, entries)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	src := fake{End: mk(map[string][]uint64{
		"memory/innodb/buf_buf_pool": {30, 40, 3 << 20, 4 << 20, 900},
		"memory/sql/THD::main_mem":   {10, 12, 1 << 20, 2 << 20, 100},
	}), OK: true}
	lines, err := MemoryUsage.Render(context.Background(), src, Options{N: 10, Width: 132})
	if err != nil {
		t.Fatal(err)
	}
	old := "%10s  %6s  %10s|%10s %6s|%8s  %6s  %8s|%s" // v1's presenter/memoryusage
	for i, want := range map[int]string{
		0: "Memory Usage (memory_summary_global_by_event_name) 2 rows, current",
		// Headings as v1's, with CurBytes and MemOps left-aligned.
		1: "CurBytes         %  High Bytes|MemOps          %|CurAlloc       %   HiAlloc|Memory Area",
		2: fmt.Sprintf(old, "3.00 M", " 75.0%", "4.00 M", "900", " 90.0%", "30", " 75.0%", "40", "memory/innodb/buf_buf_pool"),
		// Exactly 1 MiB is "1024.0 k", as in v1: a unit is used
		// only above its size.
		3: fmt.Sprintf(old, "1024.0 k", " 25.0%", "2.00 M", "100", " 10.0%", "10", " 25.0%", "12", "memory/sql/THD::main_mem"),
		4: fmt.Sprintf(old, "4.00 M", "100.0%", "6.00 M", "1000", "100.0%", "40", "100.0%", "52", "Totals"),
	} {
		if lines[i] != want {
			t.Errorf("line %d\n got %q\nwant %q", i, lines[i], want)
		}
	}
}

func TestTableLockLatencyHeadingMatchesOld(t *testing.T) {
	// v1's presenter/tablelocklatency, which pads the key heading with
	// trailing spaces.
	old := fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s %6s|%6s %6s %6s %6s %6s|%-30s",
		"Latency", "%", "Read", "Write", "S.Lock", "High", "NoIns", "Normal", "Extrnl",
		"AlloWr", "CncIns", "Low", "Normal", "Extrnl", "Table Name")
	if got, want := TableLockLatency.headings(), strings.TrimRight(old, " "); got != want {
		t.Errorf("headings\n got %q\nwant %q", got, want)
	}
}

func TestRenderCutsAtWidth(t *testing.T) {
	long := "db." + strings.Repeat("x", 100)
	src := fake{
		Start: snap(t, 0, entry(long, 0, 0)),
		End:   snap(t, time.Second, entry(long, 1, 1000)),
		OK:    true,
	}
	lines, err := TableIOOps.Render(context.Background(), src, Options{N: 5, Width: 80})
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range lines {
		if n := len([]rune(l)); n > 80 {
			t.Errorf("line of %d runes: %q", n, l)
		}
	}
}

func TestRenderNotReady(t *testing.T) {
	lines, err := TableIOOps.Render(context.Background(), fake{}, Options{N: 5, Width: 80})
	if err != nil || len(lines) != 1 || !strings.Contains(lines[0], "waiting") {
		t.Errorf("not ready: %q %v", lines, err)
	}
}

// TestRenderUnknownColumn checks a column constant with no entry in the
// Spec's Columns is reported, not indexed.
func TestRenderUnknownColumn(t *testing.T) {
	v := TableIOOps
	v.SortColumn = tableio.SumTimerDelete + 1
	if _, err := v.Render(context.Background(), fake{}, Options{N: 5, Width: 80}); err == nil {
		t.Error("unknown column: no error")
	}
}

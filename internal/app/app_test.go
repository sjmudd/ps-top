package app

import (
	"bytes"
	"context"
	"database/sql/driver"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/sjmudd/ps-top/v2/internal/fakesql"
	"github.com/sjmudd/ps-top/v2/internal/metrics"
	"github.com/sjmudd/ps-top/v2/internal/metrics/tableio"
	"github.com/sjmudd/ps-top/v2/internal/metrics/tablelocks"
	"github.com/sjmudd/ps-top/v2/internal/probe"
	"github.com/sjmudd/ps-top/v2/internal/query"
	"github.com/sjmudd/ps-top/v2/internal/registry"
	"github.com/sjmudd/ps-top/v2/internal/tui"
)

// server is a fake MySQL 8.4 for run: the memory table is missing, one mutex
// instrument is disabled, and every metric query returns rows whose values
// grow with each call.
type server struct {
	mu      sync.Mutex
	calls   int64
	columns map[string][]query.Column // metric query to its result columns
	updates [][]driver.Value          // setup_instruments UPDATE arguments, in order
}

func newServer() *server {
	s := &server{columns: map[string][]query.Column{}}
	for _, m := range metrics.All() {
		for _, q := range m.Queries {
			s.columns[q.Query] = m.Columns()
		}
	}
	return s
}

func (s *server) handle(st fakesql.Statement) fakesql.Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	q := st.Query
	switch {
	case q == "SELECT @@version":
		return fakesql.Result{Columns: []string{"@@version"}, Rows: [][]driver.Value{{"8.4.0"}}}
	case strings.HasPrefix(q, "SELECT * FROM (") && strings.Contains(q, "memory_summary_global_by_event_name"):
		return fakesql.Result{Err: &mysql.MySQLError{Number: 1146, Message: "Table 'performance_schema.memory_summary_global_by_event_name' doesn't exist"}}
	case strings.HasPrefix(q, "SELECT * FROM ("), strings.HasSuffix(q, "WHERE 1=0"):
		return fakesql.Result{} // probe of a candidate or an update: usable
	case strings.HasPrefix(q, "SHOW GLOBAL VARIABLES"):
		return fakesql.Result{Columns: []string{"Variable_name", "Value"}, Rows: [][]driver.Value{
			{"datadir", "/var/lib/mysql/"},
			{"log_bin_basename", "/var/lib/mysql/binlog"},
			{"relay_log_basename", "/var/lib/mysql/relay"},
			{"hostname", "db1"},
			{"port", "3306"},
			{"bind_address", "*"},
		}}
	case strings.HasPrefix(q, "SHOW GLOBAL STATUS"):
		return fakesql.Result{Columns: []string{"Variable_name", "Value"}, Rows: [][]driver.Value{{"Uptime", "93784"}}}
	case strings.HasPrefix(q, "SELECT NAME, ENABLED, TIMED FROM performance_schema.setup_instruments"):
		r := fakesql.Result{Columns: []string{"NAME", "ENABLED", "TIMED"}}
		if st.Args[0] == "wait/synch/mutex/%" && len(s.updates) == 0 {
			r.Rows = [][]driver.Value{{"wait/synch/mutex/x", "NO", "NO"}}
		}
		return r
	case strings.HasPrefix(q, "UPDATE performance_schema.setup_instruments SET ENABLED"):
		s.updates = append(s.updates, st.Args)
		return fakesql.Result{RowsAffected: 1}
	}
	// A query limited to some schemas (--database-filter) answers as the
	// query without the limit: every sample row is in schema shop.
	q, _, _ = strings.Cut(q, " AND OBJECT_SCHEMA IN (")
	cols, ok := s.columns[q]
	if !ok {
		return fakesql.Result{Err: errors.New("unexpected statement: " + q)}
	}
	s.calls++
	r := fakesql.Result{}
	row := []driver.Value{}
	for i, c := range cols {
		r.Columns = append(r.Columns, c.Name)
		if c.Type == query.String {
			row = append(row, key(c.Name))
			continue
		}
		row = append(row, s.calls*1000+int64(i))
	}
	r.Rows = [][]driver.Value{row}
	return r
}

// key returns a sample value for a key column.
func key(column string) string {
	switch column {
	case "OBJECT_SCHEMA":
		return "shop"
	case "OBJECT_NAME":
		return "orders"
	case "FILE_NAME":
		return "/var/lib/mysql/shop/orders.ibd"
	case "EVENT_NAME":
		return "wait/synch/mutex/innodb/x"
	}
	return "app"
}

func TestRun(t *testing.T) {
	srv := newServer()
	db := fakesql.OpenFunc(srv.handle)
	defer func() { _ = db.Close() }()
	cfg := Config{CollectInterval: 10 * time.Millisecond, PresentInterval: 60 * time.Millisecond, Window: 20 * time.Millisecond, Width: 132}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	var out bytes.Buffer
	if err := run(ctx, cfg, db.DB, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.HasPrefix(got, "MySQL 8.4.0\n") {
		t.Errorf("first line: %q", strings.SplitN(got, "\n", 2)[0])
	}
	for _, want := range []string{
		"skipped: memory: ",
		"Table I/O Latency (table_io_waits_summary_by_table) 1 row",
		"|shop.orders",
		"File I/O Latency (file_summary_by_instance) 1 row",
		"Mutex Latency (events_waits_summary_global_by_event_name) 1 row",
		"ps-top stages ", "\ncollect.table_io ", "ps-top renders ", "ps-top process 1 row",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output has no %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "metrics registry") {
		t.Errorf("output has the registry printout:\n%s", got)
	}
	// The table I/O key of two parts is shown as schema.table.
	if b := block(got, "Table I/O Latency"); !strings.Contains(b, "|shop.orders\n") {
		t.Errorf("table I/O view does not show shop.orders:\n%q", b)
	}
	// The file I/O row is the .ibd file mapped to its table.
	if b := block(got, "File I/O Latency"); !strings.Contains(b, "|shop.orders\n") {
		t.Errorf("file I/O view does not map the .ibd file to shop.orders:\n%s", b)
	}
	if strings.Contains(got, "Memory Usage") {
		t.Errorf("memory view shown although its metric is skipped:\n%s", got)
	}

	// The disabled instrument is enabled at start and put back at exit.
	srv.mu.Lock()
	defer srv.mu.Unlock()
	want := [][]driver.Value{{"YES", "YES", "wait/synch/mutex/x"}, {"NO", "NO", "wait/synch/mutex/x"}}
	if len(srv.updates) != 2 {
		t.Fatalf("setup_instruments updates = %v, want enable then restore", srv.updates)
	}
	for i := range want {
		for j := range want[i] {
			if srv.updates[i][j] != want[i][j] {
				t.Errorf("update %d = %v, want %v", i, srv.updates[i], want[i])
			}
		}
	}
}

// block returns the last view in out whose first line starts with title, up
// to the blank line after it.
func block(out, title string) string {
	i := strings.LastIndex(out, "\n"+title)
	if i < 0 {
		return ""
	}
	b := out[i+1:]
	if j := strings.Index(b, "\n\n"); j >= 0 {
		b = b[:j+1]
	}
	return b
}

func TestParseViews(t *testing.T) {
	got, err := ParseViews("table_io_latency:current,file_io_latency")
	want := []ViewChoice{{"table_io_latency", "current"}, {"file_io_latency", ""}}
	if err != nil || len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("ParseViews = %v, %v, want %v", got, err, want)
	}
	if got, err := ParseViews(""); got != nil || err != nil {
		t.Errorf(`ParseViews("") = %v, %v, want none`, got, err)
	}
	if _, err := ParseViews("a,,b"); err == nil {
		t.Error("empty view name: no error")
	}
}

func TestChoose(t *testing.T) {
	all := allViews(nil)
	vs, err := choose(all, all, []ViewChoice{{"file_io_latency", "current"}, {"table_io_latency", ""}})
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 2 || vs[0].ViewName() != "file_io_latency" || vs[0].Style() != "current" || vs[1].Style() != "difference" {
		t.Errorf("chosen: %s:%s, %s:%s", vs[0].ViewName(), vs[0].Style(), vs[1].ViewName(), vs[1].Style())
	}
	if vs, _ := choose(all, all, nil); len(vs) != len(all) {
		t.Errorf("no choices: %d views, want all %d", len(vs), len(all))
	}
	for _, tt := range []struct {
		choices []ViewChoice
		want    string
	}{
		{[]ViewChoice{{"no_such_view", ""}}, "views are table_io_latency, table_io_ops"},
		{[]ViewChoice{{"table_io_latency", "sideways"}}, "unknown style"},
		{[]ViewChoice{{"user_latency", "difference"}}, "cannot show style difference"},
		{[]ViewChoice{{"table_io_latency", ""}, {"table_io_latency", "difference"}}, "given twice"},
	} {
		if _, err := choose(all, all, tt.choices); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%v: err = %v, want %q", tt.choices, err, tt.want)
		}
	}
	// A view whose metric is skipped is not in available.
	if _, err := choose(all, all[:1], []ViewChoice{{"memory_usage", ""}}); err == nil || !strings.Contains(err.Error(), "skipped") {
		t.Errorf("skipped metric: err = %v", err)
	}
}

// TestRunBatch checks the chosen views only, the first block showing values
// up to now, the limit, and stopping after Count blocks.
// TestRunAnonymise checks --anonymise shows the schema, table and user
// anonymised in every block, the same in the table and file views, and never
// the real names.
func TestRunAnonymise(t *testing.T) {
	srv := newServer()
	db := fakesql.OpenFunc(srv.handle)
	defer func() { _ = db.Close() }()
	cfg := Config{
		CollectInterval: 10 * time.Millisecond, PresentInterval: 40 * time.Millisecond, Window: 20 * time.Millisecond,
		Views: []ViewChoice{{"table_io_latency", ""}, {"file_io_latency", ""}, {"user_latency", ""}}, Count: 2,
		Anonymise: true,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var out bytes.Buffer
	if err := run(ctx, cfg, db.DB, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"|schema1.table1\n", "|user1\n"} {
		if n := strings.Count(got, want); n < 2 {
			t.Errorf("%q shown %d times, want at least 2 (both blocks):\n%s", want, n, got)
		}
	}
	if b := block(got, "File I/O Latency"); !strings.Contains(b, "|schema1.table1\n") {
		t.Errorf("file I/O view does not show the table anonymised:\n%s", b)
	}
	for _, name := range []string{"shop", "orders", "|app\n"} {
		if strings.Contains(got, name) {
			t.Errorf("output shows %q:\n%s", name, got)
		}
	}
}

// TestServerInfo checks the top line's server details: the host shown by
// v1's rule, the port and version, and the start time from the
// uptime.
func TestServerInfo(t *testing.T) {
	probed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	srv := &probe.Server{
		Version:   "8.4.11",
		Variables: map[string]string{"hostname": "db1", "port": "3306", "bind_address": "10.0.0.5"},
		Status:    map[string]string{"uptime": "93784"}, // 1d 2h 3m 4s
	}
	got, err := serverInfo(srv, probed)
	want := tui.ServerInfo{Host: "10.0.0.5", Port: "3306", Version: "8.4.11", Started: probed.Add(-93784 * time.Second)}
	if err != nil || got != want {
		t.Errorf("serverInfo = %+v, %v, want %+v", got, err, want)
	}
	srv.Status["uptime"] = ""
	if _, err := serverInfo(srv, probed); err == nil {
		t.Error("no uptime: no error")
	}
	for _, tt := range []struct{ bind, want string }{
		{"", "db1"}, {"*", "db1"}, {"0.0.0.0", "db1"}, {"::", "db1"}, {"127.0.0.1", "127.0.0.1"},
	} {
		if got := displayHost("db1", tt.bind); got != tt.want {
			t.Errorf("displayHost(db1, %q) = %q, want %q", tt.bind, got, tt.want)
		}
	}
}

func TestParseDatabaseFilter(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"shop", []string{"shop"}},
		{" shop, crm ", []string{"shop", "crm"}},
		{"shop,,crm,", []string{"shop", "crm"}}, // empty names left out
		{"shop,my db", []string{"shop"}},        // a name with a space left out, as v1
		{" , ", nil},
	} {
		if got := ParseDatabaseFilter(tt.in); !slices.Equal(got, tt.want) {
			t.Errorf("ParseDatabaseFilter(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestRunDatabaseFilter checks --database-filter limits the table queries
// to the schemas given, with the names as arguments, leaves the other
// queries as they are, and filters the file and user views in presentation:
// the file view keeps the shop table, the user view drops the connection
// whose db is not given.
func TestRunDatabaseFilter(t *testing.T) {
	srv := newServer()
	db := fakesql.OpenFunc(srv.handle)
	defer func() { _ = db.Close() }()
	cfg := Config{
		CollectInterval: 10 * time.Millisecond, PresentInterval: 40 * time.Millisecond, Window: 20 * time.Millisecond,
		Views: []ViewChoice{{"table_io_latency", ""}, {"file_io_latency", ""}, {"user_latency", ""}}, Count: 2,
		DatabaseFilter: []string{"shop", "crm"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var out bytes.Buffer
	if err := run(ctx, cfg, db.DB, &out); err != nil {
		t.Fatal(err)
	}
	filtered := map[string]bool{}
	for _, st := range db.Statements() {
		q, _, limited := strings.Cut(st.Query, " AND OBJECT_SCHEMA IN (?,?)")
		if limited {
			filtered[q] = true
			if !slices.Equal(st.Args, []driver.Value{"shop", "crm"}) {
				t.Errorf("%s: args %v, want shop, crm", st.Query, st.Args)
			}
			continue
		}
		if q == tableio.Spec.Query(tableio.Spec.Tables[0]) || q == tablelocks.Spec.Query(tablelocks.Spec.Tables[0]) {
			t.Errorf("table query run without the filter: %s", q)
		}
	}
	for _, q := range []string{tableio.Spec.Query(tableio.Spec.Tables[0]), tablelocks.Spec.Query(tablelocks.Spec.Tables[0])} {
		if !filtered[q] {
			t.Errorf("query not run with the filter: %s", q)
		}
	}
	got := out.String()
	if b := block(got, "File I/O Latency"); !strings.Contains(b, "|shop.orders\n") {
		t.Errorf("file view does not show the shop table:\n%s", b)
	}
	// Every sample connection has db "app", which is not given.
	if b := block(got, "Activity by Username"); b == "" || strings.Contains(b, "|app\n") {
		t.Errorf("user view shows a connection whose db is not given:\n%s", b)
	}
}

func TestRunBatch(t *testing.T) {
	srv := newServer()
	db := fakesql.OpenFunc(srv.handle)
	defer func() { _ = db.Close() }()
	cfg := Config{
		CollectInterval: 10 * time.Millisecond, PresentInterval: 40 * time.Millisecond, Window: 20 * time.Millisecond,
		Views: []ViewChoice{{"table_io_latency", ""}, {"user_latency", ""}}, Count: 2, Limit: 1,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var out bytes.Buffer
	start := time.Now()
	if err := run(ctx, cfg, db.DB, &out); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("run took %v: it did not stop after %d blocks", time.Since(start), cfg.Count)
	}
	got := out.String()
	var titles []string
	blocks := 0
	for line := range strings.Lines(got) {
		switch {
		case strings.HasPrefix(line, "--- ") && !strings.HasPrefix(line, "--- metrics registry"):
			blocks++
		case strings.HasPrefix(line, "Table I/O Latency"), strings.HasPrefix(line, "Activity by Username"):
			titles = append(titles, strings.TrimSpace(line))
		}
	}
	if blocks != 2 {
		t.Errorf("%d blocks, want 2:\n%s", blocks, got)
	}
	want := []string{
		"Table I/O Latency (table_io_waits_summary_by_table) 1 row, current", // first block: up to now
		"Activity by Username (processlist) 1 row, current",
	}
	if len(titles) != 4 || titles[0] != want[0] || titles[1] != want[1] ||
		!strings.Contains(titles[2], "1 row, last ") || titles[3] != want[1] {
		t.Errorf("titles:\n%s", strings.Join(titles, "\n"))
	}
	if strings.Contains(got, "File I/O Latency") || strings.Contains(got, "Mutex Latency") {
		t.Errorf("a view not chosen is shown:\n%s", got)
	}
}

func TestRunConfigErrors(t *testing.T) {
	db := fakesql.Open(fakesql.Result{Err: errors.New("not used")})
	defer func() { _ = db.Close() }()
	for _, cfg := range []Config{
		{CollectInterval: 0, PresentInterval: time.Second, Window: time.Second},
		{CollectInterval: time.Second, PresentInterval: 0, Window: time.Second},
		{CollectInterval: time.Second, PresentInterval: time.Second, Window: 500 * time.Millisecond},
		{CollectInterval: time.Second, PresentInterval: time.Second, Window: time.Second, Count: -1},
		{CollectInterval: time.Second, PresentInterval: time.Second, Window: time.Second, Views: []ViewChoice{{"no_such_view", ""}}},
		{CollectInterval: time.Second, PresentInterval: time.Second, Window: time.Second, Views: []ViewChoice{{"user_latency", "difference"}}},
	} {
		if err := run(context.Background(), cfg, db.DB, &bytes.Buffer{}); err == nil {
			t.Errorf("%+v: no error", cfg)
		}
	}
	if n := len(db.Statements()); n != 0 {
		t.Errorf("%d statements run for invalid settings, want 0", n)
	}
}

func TestRunNoUsableMetric(t *testing.T) {
	db := fakesql.OpenFunc(func(st fakesql.Statement) fakesql.Result {
		switch {
		case st.Query == "SELECT @@version":
			return fakesql.Result{Columns: []string{"@@version"}, Rows: [][]driver.Value{{"8.4.0"}}}
		case strings.HasPrefix(st.Query, "SELECT * FROM ("):
			return fakesql.Result{Err: &mysql.MySQLError{Number: 1142, Message: "SELECT command denied"}}
		}
		return fakesql.Result{Columns: []string{"a", "b", "c"}} // no variables, no instruments
	})
	defer func() { _ = db.Close() }()
	cfg := Config{CollectInterval: time.Second, PresentInterval: time.Second, Window: time.Second, Width: 80}
	err := run(context.Background(), cfg, db.DB, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "no metric is usable") {
		t.Errorf("err = %v, want no metric is usable", err)
	}
}

// TestRegistryMetrics checks the registry is collected as two metrics, its
// stages then its process figures, read without the database and recorded
// in entries of their own.
// TestForTUI checks the TUI shows ps-top's own metrics as one view, the
// printer as three.
func TestForTUI(t *testing.T) {
	names := func(tui bool) []string {
		var out []string
		for _, v := range forTUI(allViews(nil), tui) {
			out = append(out, v.ViewName())
		}
		return out
	}
	tuiNames, printer := names(true), names(false)
	if tuiNames[len(tuiNames)-1] != "ps_top" || slices.Contains(tuiNames, "ps_top_stages") {
		t.Errorf("TUI views %v", tuiNames)
	}
	if want := []string{"ps_top_stages", "ps_top_renders", "ps_top_process"}; !slices.Equal(printer[len(printer)-3:], want) || slices.Contains(printer, "ps_top") {
		t.Errorf("printer views %v", printer)
	}
	if len(tuiNames) != 9 || len(printer) != 11 {
		t.Errorf("%d TUI views, %d printer views, want 9 and 11", len(tuiNames), len(printer))
	}
}

func TestRegistryMetrics(t *testing.T) {
	reg := registry.New(time.Now())
	cms, err := registryMetrics(reg)
	if err != nil {
		t.Fatal(err)
	}
	if len(cms) != 2 || cms[0].Def.Name != registry.StagesMetric || cms[1].Def.Name != registry.ProcessMetric {
		t.Fatalf("metrics %v", cms)
	}
	for _, m := range cms {
		entries, err := m.Collect(nil)
		if err != nil || len(entries) == 0 {
			t.Errorf("%s: %d entries, err %v", m.Def.Name, len(entries), err)
		}
		if m.Stats.Query.Count.Load() != 1 {
			t.Errorf("%s: read not recorded", m.Def.Name)
		}
	}
	if _, err := registryMetrics(reg); err == nil {
		t.Error("second call: no error for entries already made")
	}
}

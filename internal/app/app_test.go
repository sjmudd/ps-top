package app

import (
	"bytes"
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/sjmudd/ps-top/internal/fakesql"
	"github.com/sjmudd/ps-top/internal/metrics"
	"github.com/sjmudd/ps-top/internal/query"
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
		}}
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
		"Table I/O Latency (table_io_waits_summary_by_table) 1 rows",
		"|shop.orders",
		"File I/O Latency (file_summary_by_instance) 1 rows",
		"Mutex Latency (events_waits_summary_global_by_event_name) 1 rows",
		"--- metrics registry ---",
		"METRIC_NAME", "VIEW_NAME", "PROCESS",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output has no %q:\n%s", want, got)
		}
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

func TestRunConfigErrors(t *testing.T) {
	db := fakesql.Open(fakesql.Result{Err: errors.New("not used")})
	defer func() { _ = db.Close() }()
	for _, cfg := range []Config{
		{CollectInterval: 0, PresentInterval: time.Second, Window: time.Second},
		{CollectInterval: time.Second, PresentInterval: 0, Window: time.Second},
		{CollectInterval: time.Second, PresentInterval: time.Second, Window: 1500 * time.Millisecond},
		{CollectInterval: time.Second, PresentInterval: time.Second, Window: 500 * time.Millisecond},
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

package metrics

import (
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/sjmudd/ps-top/internal/fakesql"
	"github.com/sjmudd/ps-top/internal/metrics/fileio"
	"github.com/sjmudd/ps-top/internal/metrics/memory"
	"github.com/sjmudd/ps-top/internal/metrics/stages"
	"github.com/sjmudd/ps-top/internal/metrics/synchwaits"
	"github.com/sjmudd/ps-top/internal/metrics/tableio"
	"github.com/sjmudd/ps-top/internal/metrics/tablelocks"
	"github.com/sjmudd/ps-top/internal/metrics/userlatency"
	"github.com/sjmudd/ps-top/internal/registry"
)

// TestQueries checks each Spec is valid and builds the query written by hand
// before queries were built from specs.
func TestQueries(t *testing.T) {
	tests := []struct {
		err   error
		query string
		want  string
	}{
		{tableio.Spec.Validate(), tableio.Spec.Query(tableio.Spec.Tables[0]),
			`SELECT OBJECT_SCHEMA, OBJECT_NAME, COUNT_STAR, SUM_TIMER_WAIT, COUNT_READ, SUM_TIMER_READ, COUNT_WRITE, SUM_TIMER_WRITE, COUNT_FETCH, SUM_TIMER_FETCH, COUNT_INSERT, SUM_TIMER_INSERT, COUNT_UPDATE, SUM_TIMER_UPDATE, COUNT_DELETE, SUM_TIMER_DELETE FROM performance_schema.table_io_waits_summary_by_table WHERE COUNT_STAR > 0`},
		{fileio.Spec.Validate(), fileio.Spec.Query(fileio.Spec.Tables[0]),
			`SELECT FILE_NAME, COUNT_STAR, SUM_TIMER_WAIT, COUNT_READ, SUM_TIMER_READ, COUNT_WRITE, SUM_TIMER_WRITE, COUNT_MISC, SUM_TIMER_MISC, SUM_NUMBER_OF_BYTES_READ, SUM_NUMBER_OF_BYTES_WRITE FROM performance_schema.file_summary_by_instance WHERE COUNT_STAR > 0`},
		{synchwaits.Spec.Validate(), synchwaits.Spec.Query(synchwaits.Spec.Tables[0]),
			`SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT FROM performance_schema.events_waits_summary_global_by_event_name WHERE COUNT_STAR > 0 AND (EVENT_NAME LIKE 'wait/synch/mutex/%' OR EVENT_NAME LIKE 'wait/synch/sxlock/innodb/%')`},
		{stages.Spec.Validate(), stages.Spec.Query(stages.Spec.Tables[0]),
			`SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT FROM performance_schema.events_stages_summary_global_by_event_name WHERE COUNT_STAR > 0`},
		{tablelocks.Spec.Validate(), tablelocks.Spec.Query(tablelocks.Spec.Tables[0]),
			`SELECT OBJECT_SCHEMA, OBJECT_NAME, SUM_TIMER_WAIT, SUM_TIMER_READ, SUM_TIMER_WRITE, SUM_TIMER_READ_WITH_SHARED_LOCKS, SUM_TIMER_READ_HIGH_PRIORITY, SUM_TIMER_READ_NO_INSERT, SUM_TIMER_READ_NORMAL, SUM_TIMER_READ_EXTERNAL, SUM_TIMER_WRITE_ALLOW_WRITE, SUM_TIMER_WRITE_CONCURRENT_INSERT, SUM_TIMER_WRITE_LOW_PRIORITY, SUM_TIMER_WRITE_NORMAL, SUM_TIMER_WRITE_EXTERNAL FROM performance_schema.table_lock_waits_summary_by_table WHERE COUNT_STAR > 0`},
		// The old program's columns, with MEM_OPS as in its query.
		{memory.Spec.Validate(), memory.Spec.Query(memory.Spec.Tables[0]),
			`SELECT EVENT_NAME, CURRENT_COUNT_USED, HIGH_COUNT_USED, CURRENT_NUMBER_OF_BYTES_USED, HIGH_NUMBER_OF_BYTES_USED, COUNT_ALLOC + COUNT_FREE AS MEM_OPS FROM performance_schema.memory_summary_global_by_event_name WHERE HIGH_COUNT_USED > 0`},
	}
	for _, tt := range tests {
		if tt.err != nil {
			t.Error(tt.err)
		}
		if tt.query != tt.want {
			t.Errorf("query\n got %s\nwant %s", tt.query, tt.want)
		}
	}
}

// TestUserLatency checks the processlist metric has a candidate per table, in
// order, each grouped by user and not containing the keywords it counts.
func TestUserLatency(t *testing.T) {
	if err := userlatency.Spec.Validate(); err != nil {
		t.Fatal(err)
	}
	m := From(userlatency.Spec)
	want := []string{"performance_schema.processlist", "INFORMATION_SCHEMA.PROCESSLIST"}
	if len(m.Queries) != len(want) {
		t.Fatalf("%d candidates, want %d", len(m.Queries), len(want))
	}
	for i, c := range m.Queries {
		if c.Name != want[i] || !strings.Contains(c.Query, " FROM "+want[i]+" GROUP BY USER") {
			t.Errorf("candidate %d = %q: %s", i, c.Name, c.Query)
		}
		for _, kw := range []string{"INSERT", "UPDATE", "DELETE"} {
			if strings.Contains(c.Query, kw) {
				t.Errorf("candidate %d contains %s, so it would count itself", i, kw)
			}
		}
	}
}

// TestCollect checks Collect against a fake server: rows with the same key
// are merged, and the query and the merge are recorded in the registry entry.
func TestCollect(t *testing.T) {
	db := fakesql.Open(fakesql.Result{
		Columns: []string{"EVENT_NAME", "COUNT_STAR", "SUM_TIMER_WAIT"},
		Rows: [][]driver.Value{
			{"stage/sql/a", int64(2), int64(100)},
			{"stage/sql/b", int64(1), int64(50)},
			{"stage/sql/a", int64(3), int64(200)},
		},
	})
	defer func() { _ = db.Close() }()
	m := From(stages.Spec)
	st := new(registry.Collector)
	entries, err := m.Collect(db.DB, m.Queries[0].Query, st)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]uint64{}
	for _, e := range entries {
		got[e.Key.Value()] = e.Values
	}
	if a := got["stage/sql/a"]; len(entries) != 2 || len(a) != 2 || a[0] != 5 || a[1] != 300 {
		t.Errorf("entries = %v, want stage/sql/a [5 300] and stage/sql/b", got)
	}
	if st.Query.Count.Load() != 1 || st.Query.SumTimer.Load() == 0 {
		t.Errorf("COUNT_QUERY %d, SUM_TIMER_QUERY %d: want 1 and the time taken", st.Query.Count.Load(), st.Query.SumTimer.Load())
	}
	if st.RowsRead.Load() != 3 || st.Merge.Count.Load() != 1 || st.Entries.Load() != 2 || st.QueryErrors.Load() != 0 {
		t.Errorf("SUM_ROWS_READ %d, COUNT_MERGE %d, SUM_ENTRIES %d, COUNT_QUERY_ERROR %d: want 3, 1, 2, 0",
			st.RowsRead.Load(), st.Merge.Count.Load(), st.Entries.Load(), st.QueryErrors.Load())
	}
}

// TestCollectError checks a failed query is counted and nothing is merged.
func TestCollectError(t *testing.T) {
	db := fakesql.Open(fakesql.Result{Err: errors.New("server gone")})
	defer func() { _ = db.Close() }()
	m := From(stages.Spec)
	st := new(registry.Collector)
	if _, err := m.Collect(db.DB, m.Queries[0].Query, st); err == nil {
		t.Fatal("no error")
	}
	if st.Query.Count.Load() != 1 || st.QueryErrors.Load() != 1 || st.Merge.Count.Load() != 0 {
		t.Errorf("COUNT_QUERY %d, COUNT_QUERY_ERROR %d, COUNT_MERGE %d: want 1, 1, 0",
			st.Query.Count.Load(), st.QueryErrors.Load(), st.Merge.Count.Load())
	}
}

func TestRequest(t *testing.T) {
	req := Request()
	n := 0
	for _, m := range All() {
		n += len(m.Queries)
	}
	if len(req.Candidates) != n || len(req.Variables) != 0 {
		t.Errorf("Request() = %+v, want every metric's candidates and no variables", req)
	}
}

package registry

import (
	"slices"
	"testing"
	"time"

	"github.com/sjmudd/ps-top/v2/internal/metric"
	"github.com/sjmudd/ps-top/v2/internal/snapshot"
)

func TestDefinitions(t *testing.T) {
	for _, d := range []metric.Definition{StagesDefinition(), ProcessDefinition()} {
		if err := d.Validate(); err != nil {
			t.Error(err)
		}
	}
	p := ProcessDefinition()
	if len(p.Columns) != len(runtimeNames) || p.Columns[p.ResetColumn].Name != "COUNT_GC_CYCLES" {
		t.Errorf("process columns %v, reset column %d", p.Columns, p.ResetColumn)
	}
	for _, c := range p.Columns {
		if gauge := c.Name[:8] == "CURRENT_"; gauge != (c.Kind == metric.Gauge) {
			t.Errorf("column %s has kind %d", c.Name, c.Kind)
		}
	}
}

// TestStages checks one entry per collector stage and per render, with
// count, time and rows, and the read recorded in its own entry.
func TestStages(t *testing.T) {
	r := New(time.Now())
	tio, _ := r.Collector("table_io")
	own, _ := r.Collector(StagesMetric)
	view, _ := r.Presenter("table_io_latency", "difference")
	tio.Query.Count.Store(4)
	tio.Query.SumTimer.Store(4000)
	tio.RowsRead.Store(40)
	tio.Merge.Count.Store(4)
	tio.Snapshot.Count.Store(4)
	view.Render.Count.Store(2)
	view.Render.SumTimer.Store(900)
	view.EntriesMapped.Store(80)

	got := map[string][]uint64{}
	for _, e := range r.Stages(own) {
		got[snapshot.KeyText(e.Key.Value())] = e.Values
	}
	want := map[string][]uint64{
		"collect.table_io.query":                     {4, 4000, 40},
		"collect.table_io.merge":                     {4, 0, 0},
		"collect.table_io.snapshot":                  {4, 0, 0},
		"collect." + StagesMetric + ".query":         {0, 0, 0}, // read before its own read ends
		"collect." + StagesMetric + ".merge":         {0, 0, 0},
		"collect." + StagesMetric + ".snapshot":      {0, 0, 0},
		"present.table_io_latency difference.render": {2, 900, 80},
	}
	if len(got) != len(want) {
		t.Errorf("%d entries, want %d: %v", len(got), len(want), got)
	}
	for k, v := range want {
		if !slices.Equal(got[k], v) {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
	if own.Query.Count.Load() != 1 || own.RowsRead.Load() != 7 {
		t.Errorf("own read: count %d, rows %d, want 1 and 7", own.Query.Count.Load(), own.RowsRead.Load())
	}
}

// TestProcess checks one entry of the runtime's figures, and the read
// recorded.
func TestProcess(t *testing.T) {
	r := New(time.Now())
	st, _ := r.Collector(ProcessMetric)
	entries := r.Process(st)
	if len(entries) != 1 || snapshot.KeyText(entries[0].Key.Value()) != "process" || len(entries[0].Values) != len(runtimeNames) {
		t.Fatalf("entries %v", entries)
	}
	for i, n := range runtimeNames {
		if n.column == "CURRENT_MEMORY_BYTES" && entries[0].Values[i] == 0 {
			t.Errorf("%s = 0", n.column)
		}
	}
	if st.Query.Count.Load() != 1 {
		t.Errorf("own read: count %d, want 1", st.Query.Count.Load())
	}
}

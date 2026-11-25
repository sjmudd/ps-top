package presenter

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"
	"unique"

	"github.com/sjmudd/ps-top/internal/collector"
	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/registry"
	"github.com/sjmudd/ps-top/internal/snapshot"
)

// stagesSnap returns a stages metric snapshot at t0+at: values per (part,
// name, stage) key, written with "." between the parts.
func stagesSnap(t *testing.T, at time.Duration, values map[[3]string][3]uint64) *snapshot.Snapshot {
	t.Helper()
	var entries []snapshot.Entry
	for k, v := range values {
		entries = append(entries, snapshot.Entry{Key: unique.Make(snapshot.JoinKey(k[0], k[1], k[2])), Values: []uint64{v[0], v[1], v[2]}})
	}
	s, err := snapshot.New(t0.Add(at), 3, entries)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func ownSource(t *testing.T) fake {
	zero := map[[3]string][3]uint64{
		{"collect", "table_io", "query"}: {}, {"collect", "table_io", "merge"}: {}, {"collect", "table_io", "snapshot"}: {},
		{"collect", "memory", "query"}: {}, {"collect", "memory", "merge"}: {}, {"collect", "memory", "snapshot"}: {},
		{"present", "table_io_latency difference", "render"}: {},
	}
	return fake{
		Start: stagesSnap(t, 0, zero),
		End: stagesSnap(t, 10*time.Second, map[[3]string][3]uint64{
			{"collect", "table_io", "query"}:                     {5, 812000000, 40},
			{"collect", "table_io", "merge"}:                     {5, 20000000, 0},
			{"collect", "table_io", "snapshot"}:                  {5, 12000000, 0},
			{"collect", "memory", "query"}:                       {0, 0, 0}, // not collected: hidden
			{"collect", "memory", "merge"}:                       {0, 0, 0},
			{"collect", "memory", "snapshot"}:                    {0, 0, 0},
			{"present", "table_io_latency difference", "render"}: {2, 1200000000, 80},
		}),
		OK: true,
	}
}

func TestPsTopStages(t *testing.T) {
	if PsTopStages.Metric() != registry.StagesMetric || PsTopRenders.Metric() != registry.StagesMetric {
		t.Errorf("metrics %q and %q, want %q", PsTopStages.Metric(), PsTopRenders.Metric(), registry.StagesMetric)
	}
	lines, err := PsTopStages.Render(context.Background(), ownSource(t), Options{N: 1})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"ps-top stages 1 row, last 10s",
		"                      |         Query          |     Merge     |   Snapshot    | Total",
		"Name                  | Count     Time     Rows| Count     Time| Count     Time|",
		"collect.table_io      |     5  812.0us       40|     5   20.0us|     5   12.0us| 844.0us",
	}
	check(t, lines, want)
}

func TestPsTopRenders(t *testing.T) {
	lines, err := PsTopRenders.Render(context.Background(), ownSource(t), Options{N: 1})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"ps-top renders 1 row, last 10s",
		"                                     |         Render",
		"Name                                 | Count     Time     Rows",
		"present.table_io_latency difference  |     2    1.2ms       80",
	}
	check(t, lines, want)
}

// TestPsTopProcess checks the view's columns are the registry's, and one
// line of values: gauges as they are, counters as their increase.
func TestPsTopProcess(t *testing.T) {
	d := registry.ProcessDefinition()
	for i, c := range processSpec.Columns {
		if c != d.Columns[i] {
			t.Errorf("column %d is %v, registry has %v", i, c, d.Columns[i])
		}
	}
	key := unique.Make(snapshot.JoinKey("process"))
	snap := func(at time.Duration, v ...uint64) *snapshot.Snapshot {
		s, err := snapshot.New(t0.Add(at), len(v), []snapshot.Entry{{Key: key, Values: v}})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	src := fake{
		Start: snap(0, 10<<20, 4<<20, 100<<20, 10, 12, 100),
		End:   snap(10*time.Second, 12<<20, 5<<20, 120<<20, 13, 14, 127),
		OK:    true,
	}
	lines, err := PsTopProcess.Render(context.Background(), src, Options{N: 1})
	if err != nil {
		t.Fatal(err)
	}
	check(t, lines, []string{
		"ps-top process 1 row, last 10s",
		"   Memory      Heap     Alloc     GC Goroutines  Created",
		"  12.00 M    5.00 M   20.00 M      3         14       27",
	})
}

// byMetric is a Source with a fake per metric.
type byMetric map[string]fake

func (b byMetric) Window(ctx context.Context, name string, t metric.MetricType, n int) (collector.Window, error) {
	return b[name].Window(ctx, name, t, n)
}

// TestPsTop checks the combined view: its parts' tables in order, one style,
// and one render recorded, in its own entry.
func TestPsTop(t *testing.T) {
	key := unique.Make(snapshot.JoinKey("process"))
	snap := func(at time.Duration) *snapshot.Snapshot {
		s, err := snapshot.New(t0.Add(at), 6, []snapshot.Entry{{Key: key, Values: []uint64{1, 1, 1, 1, 1, 1}}})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	src := byMetric{
		registry.StagesMetric:  ownSource(t),
		registry.ProcessMetric: {Start: snap(0), End: snap(10 * time.Second), OK: true},
	}
	st := new(registry.Presenter)
	v, err := PsTop.WithStyle(metric.CurrentValue)
	if err != nil {
		t.Fatal(err)
	}
	tab, err := v.WithStats(st).Table(context.Background(), src, Options{N: 1})
	if err != nil {
		t.Fatal(err)
	}
	if tab.Title != "ps-top internals" || tab.Style != "current" || tab.Waiting || len(tab.Parts) != 3 {
		t.Fatalf("table %q, style %q, waiting %v, %d parts", tab.Title, tab.Style, tab.Waiting, len(tab.Parts))
	}
	for i, prefix := range []string{"ps-top stages ", "ps-top renders ", "ps-top process "} {
		if p := tab.Parts[i]; !strings.HasPrefix(p.Title, prefix) || p.Style != "current" {
			t.Errorf("part %d: %q, style %q", i, p.Title, p.Style)
		}
	}
	if st.Render.Count.Load() != 1 {
		t.Errorf("%d renders recorded, want 1", st.Render.Count.Load())
	}
	lines := tab.Lines(0)
	if lines[0] != "ps-top internals" || lines[1] != tab.Parts[0].Title || !slices.Contains(lines, "") {
		t.Errorf("lines %q", lines)
	}
	if PsTop.Accepts(metric.CurrentValue) != true || PsTop.Style() != "difference" {
		t.Errorf("accepts current %v, style %q", PsTop.Accepts(metric.CurrentValue), PsTop.Style())
	}
}

func check(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%d lines, want %d:\n%q", len(got), len(want), got)
	}
	for i := range min(len(got), len(want)) {
		if got[i] != want[i] {
			t.Errorf("line %d\n got %q\nwant %q", i, got[i], want[i])
		}
	}
}

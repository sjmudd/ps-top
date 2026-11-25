package presenter

import (
	"fmt"
	"unique"

	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/registry"
	"github.com/sjmudd/ps-top/internal/snapshot"
)

// Views of ps-top's own metrics (see registry.StagesMetric). The stages
// metric has one entry per stage: (collect, metric, stage) for query, merge
// and snapshot, (present, view and style, render) for renders. Its views
// merge these at the map step into one row per metric or per view, each
// stage a group of columns.

// StageColumn is a column of the merged collection stages.
type StageColumn int

// The merged collection stage columns, in stagesSpec.Columns order.
const (
	QueryCount StageColumn = iota
	QueryTimer
	QueryRows
	MergeCount
	MergeTimer
	SnapshotCount
	SnapshotTimer
	TotalTimer // the sum of the three timers
)

var stagesSpec = metric.Spec[StageColumn]{
	Name: registry.StagesMetric,
	Columns: []metric.Column{
		QueryCount:    {Name: "QUERY_COUNT", Kind: metric.Counter},
		QueryTimer:    {Name: "QUERY_TIMER", Kind: metric.Counter},
		QueryRows:     {Name: "QUERY_ROWS", Kind: metric.Counter},
		MergeCount:    {Name: "MERGE_COUNT", Kind: metric.Counter},
		MergeTimer:    {Name: "MERGE_TIMER", Kind: metric.Counter},
		SnapshotCount: {Name: "SNAPSHOT_COUNT", Kind: metric.Counter},
		SnapshotTimer: {Name: "SNAPSHOT_TIMER", Kind: metric.Counter},
		TotalTimer:    {Name: "TOTAL_TIMER", Kind: metric.Counter},
	},
	Types:       []metric.MetricType{metric.DifferenceValues, metric.CurrentValue},
	ResetColumn: QueryCount,
}

// RenderColumn is a column of the merged render stages.
type RenderColumn int

// The merged render columns, in rendersSpec.Columns order.
const (
	RenderCount RenderColumn = iota
	RenderTimer
	RenderRows
)

var rendersSpec = metric.Spec[RenderColumn]{
	Name: "ps_top_renders",
	Columns: []metric.Column{
		RenderCount: {Name: "RENDER_COUNT", Kind: metric.Counter},
		RenderTimer: {Name: "RENDER_TIMER", Kind: metric.Counter},
		RenderRows:  {Name: "RENDER_ROWS", Kind: metric.Counter},
	},
	Types:       []metric.MetricType{metric.DifferenceValues, metric.CurrentValue},
	ResetColumn: RenderCount,
}

// The stages metric's values, in registry.StagesDefinition order.
const (
	stageCount = iota
	stageTimer
	stageRows
)

// mergeStages returns the collection stages of s, a stages metric snapshot,
// as one entry per metric, keyed (collect, metric), in stagesSpec columns.
func mergeStages(s *snapshot.Snapshot) (*snapshot.Snapshot, error) {
	return mergeParts(s, "collect", len(stagesSpec.Columns), func(stage string, in, out []uint64) error {
		switch stage {
		case "query":
			out[QueryCount], out[QueryTimer], out[QueryRows] = in[stageCount], in[stageTimer], in[stageRows]
		case "merge":
			out[MergeCount], out[MergeTimer] = in[stageCount], in[stageTimer]
		case "snapshot":
			out[SnapshotCount], out[SnapshotTimer] = in[stageCount], in[stageTimer]
		default:
			return fmt.Errorf("no collection stage %q", stage)
		}
		out[TotalTimer] += in[stageTimer]
		return nil
	})
}

// mergeRenders returns the renders of s, a stages metric snapshot, as one
// entry per view and style, keyed (present, view and style), in rendersSpec
// columns.
func mergeRenders(s *snapshot.Snapshot) (*snapshot.Snapshot, error) {
	return mergeParts(s, "present", len(rendersSpec.Columns), func(stage string, in, out []uint64) error {
		if stage != "render" {
			return fmt.Errorf("no render stage %q", stage)
		}
		out[RenderCount], out[RenderTimer], out[RenderRows] = in[stageCount], in[stageTimer], in[stageRows]
		return nil
	})
}

// mergeParts returns the entries of s whose key's first part is part, one
// per (part, name), width values each, set by add from each stage's values,
// in first-seen order.
func mergeParts(s *snapshot.Snapshot, part string, width int, add func(stage string, in, out []uint64) error) (*snapshot.Snapshot, error) {
	var entries []snapshot.Entry
	index := map[unique.Handle[string]]int{}
	for _, e := range s.Entries {
		p := snapshot.KeyParts(e.Key.Value())
		if len(p) != 3 {
			return nil, fmt.Errorf("stages key %q has %d parts, want 3", snapshot.KeyText(e.Key.Value()), len(p))
		}
		if p[0] != part {
			continue
		}
		k := unique.Make(snapshot.JoinKey(p[0], p[1]))
		i, ok := index[k]
		if !ok {
			i = len(entries)
			index[k] = i
			entries = append(entries, snapshot.Entry{Key: k, Values: make([]uint64, width)})
		}
		if err := add(p[2], e.Values, entries[i].Values); err != nil {
			return nil, err
		}
	}
	return snapshot.New(s.Time, width, entries)
}

// PsTopStages shows ps-top's collection stages: per metric, the count, time
// and rows of its queries, the count and time of its merges and snapshots,
// and the total time.
var PsTopStages = View[StageColumn]{
	Name:  "ps_top_stages",
	Title: "ps-top stages",
	Spec:  stagesSpec,
	From:  registry.StagesMetric,
	Merge: mergeStages,
	Cells: [][]Cell[StageColumn]{
		{Key[StageColumn]("Name", 22)},
		{Value("Count", QueryCount, formatter.Count, 6), Value("Time", QueryTimer, formatter.ShortTime, 8), Value("Rows", QueryRows, formatter.Count, 8)},
		{Value("Count", MergeCount, formatter.Count, 6), Value("Time", MergeTimer, formatter.ShortTime, 8)},
		{Value("Count", SnapshotCount, formatter.Count, 6), Value("Time", SnapshotTimer, formatter.ShortTime, 8)},
		{Value("", TotalTimer, formatter.ShortTime, 8)},
	},
	Groups:     []string{"", "Query", "Merge", "Snapshot", "Total"},
	SortColumn: TotalTimer,
	NoTotals:   true,
}

// PsTopRenders shows ps-top's renders: per view and style, the count, time
// and rows received.
var PsTopRenders = View[RenderColumn]{
	Name:  "ps_top_renders",
	Title: "ps-top renders",
	Spec:  rendersSpec,
	From:  registry.StagesMetric,
	Merge: mergeRenders,
	Cells: [][]Cell[RenderColumn]{
		{Key[RenderColumn]("Name", 37)},
		{Value("Count", RenderCount, formatter.Count, 6), Value("Time", RenderTimer, formatter.ShortTime, 8), Value("Rows", RenderRows, formatter.Count, 8)},
	},
	Groups:     []string{"", "Render"},
	SortColumn: RenderTimer,
	NoTotals:   true,
}

// PsTop shows ps-top's own metrics on one screen, for the TUI: stages,
// renders and process, one under the other.
var PsTop = Combined{
	Name:  "ps_top",
	Title: "ps-top internals",
	Parts: []Renderer{PsTopStages, PsTopRenders, PsTopProcess},
}

// ProcessColumn is a column of the process metric.
type ProcessColumn int

// The process metric's columns, in registry.ProcessDefinition order.
const (
	CurrentMemoryBytes ProcessColumn = iota
	CurrentHeapObjectsBytes
	SumHeapAllocBytes
	CountGCCycles
	CurrentGoroutines
	CountGoroutinesCreated
)

// processSpec is registry.ProcessDefinition as a Spec, differences by
// default: gauges show their latest value, counters their increase.
var processSpec = func() metric.Spec[ProcessColumn] {
	d := registry.ProcessDefinition()
	return metric.Spec[ProcessColumn]{
		Name:        d.Name,
		Columns:     d.Columns,
		Types:       []metric.MetricType{metric.DifferenceValues, metric.CurrentValue},
		ResetColumn: ProcessColumn(d.ResetColumn),
	}
}()

// PsTopProcess shows ps-top's Go runtime figures on one line.
var PsTopProcess = View[ProcessColumn]{
	Name:  "ps_top_process",
	Title: "ps-top process",
	Spec:  processSpec,
	Cells: [][]Cell[ProcessColumn]{{
		Value("Memory", CurrentMemoryBytes, formatter.Amount, 9),
		Value("Heap", CurrentHeapObjectsBytes, formatter.Amount, 9),
		Value("Alloc", SumHeapAllocBytes, formatter.Amount, 9),
		Value("GC", CountGCCycles, formatter.Count, 6),
		Value("Goroutines", CurrentGoroutines, formatter.Count, 10),
		Value("Created", CountGoroutinesCreated, formatter.Count, 8),
	}},
	SortColumn: CurrentMemoryBytes,
	ShowZero:   true,
	NoTotals:   true,
}

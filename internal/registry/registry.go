// Package registry is the metrics registry: what ps-top's own pipeline does,
// so its costs can be seen. The running code pushes its values: each part is
// given a pointer to its own entry when it is built and updates it as it
// runs. Nothing is pulled from the parts.
//
// There are three sets of entries:
//
//   - Collector, one per metric (key METRIC_NAME): the query, merge and
//     snapshot stages of each collection.
//   - Presenter, one per view and view style (keys VIEW_NAME, VIEW_STYLE):
//     the window, map, derive, order and draw stages of each render.
//   - Global, one (key PROCESS): when ps-top started, and the Go runtime's
//     memory, GC and goroutine figures, read when the registry is reported.
//
// Column names follow performance_schema: COUNT_<stage> and SUM_<...> are
// counters, added to, so cumulative from ps-top's start. CURRENT_<...> are
// gauges holding their latest value. SUM_TIMER_<stage> is in picoseconds, as
// in performance_schema, so SUM_TIMER_<stage> / COUNT_<stage> is the average
// time of the stage over ps-top's run time. No history is kept.
//
// Fields are atomic: the collector and the presenter run in different
// goroutines. Entries are created before either runs.
package registry

import (
	"fmt"
	"io"
	"runtime/metrics"
	"strconv"
	"strings"
	"sync/atomic"
	"text/tabwriter"
	"time"

	"github.com/sjmudd/ps-top/internal/formatter"
)

// Stage is one stage of the pipeline: how often it ran and its total time.
type Stage struct {
	Count    atomic.Uint64 // COUNT_<stage>
	SumTimer atomic.Uint64 // SUM_TIMER_<stage>, picoseconds
}

// Start starts timing one run of s. Stop the returned timer when the run
// ends: t := s.Start(); ...; t.Stop(), or defer s.Start().Stop().
func (s *Stage) Start() Timer { return Timer{stage: s, start: time.Now()} }

// Timer times one run of a stage.
type Timer struct {
	stage *Stage
	start time.Time
}

// Stop adds the run to its stage: one to the count, the elapsed time to the
// sum.
func (t Timer) Stop() {
	t.stage.Count.Add(1)
	t.stage.SumTimer.Add(uint64(time.Since(t.start).Nanoseconds()) * 1000)
}

// Collector is what collecting one metric did.
type Collector struct {
	Query        Stage         // COUNT_QUERY, SUM_TIMER_QUERY: queries run, round trip and scan
	QueryErrors  atomic.Uint64 // COUNT_QUERY_ERROR: queries that failed
	RowsRead     atomic.Uint64 // SUM_ROWS_READ: result rows read
	Merge        Stage         // COUNT_MERGE, SUM_TIMER_MERGE: rows with the same key merged
	Entries      atomic.Uint64 // SUM_ENTRIES: entries stored after merging
	Snapshot     Stage         // COUNT_SNAPSHOT, SUM_TIMER_SNAPSHOT: snapshot built and added to the ring
	Snapshots    atomic.Uint64 // CURRENT_SNAPSHOTS: snapshots held in the ring
	RingCapacity atomic.Uint64 // CURRENT_RING_CAPACITY: snapshots the ring can hold
}

// Presenter is what rendering one view in one style did.
type Presenter struct {
	Render        Stage         // COUNT_RENDER, SUM_TIMER_RENDER: whole renders
	Window        Stage         // COUNT_WINDOW, SUM_TIMER_WINDOW: requests to the collector, with the wait for the reply
	Map           Stage         // COUNT_MAP, SUM_TIMER_MAP: snapshots mapped with the view's key mapping
	EntriesMapped atomic.Uint64 // SUM_ENTRIES_MAPPED: entries mapped
	Derive        Stage         // COUNT_DERIVE, SUM_TIMER_DERIVE: current values or the difference over the window
	Order         Stage         // COUNT_ORDER, SUM_TIMER_ORDER: rows chosen and sorted
	Draw          Stage         // COUNT_DRAW, SUM_TIMER_DRAW: cells formatted, lines built and cut to width
	LinesDrawn    atomic.Uint64 // SUM_LINES_DRAWN: lines drawn
	RowsShown     atomic.Uint64 // CURRENT_ROWS_SHOWN: rows shown in the last render
	RowsTotal     atomic.Uint64 // CURRENT_ROWS_TOTAL: rows derived in the last render
}

// Go runtime metrics read for the Global entry. Names checked against
// go1.27.4: all six present, /gc/cycles/total, /gc/heap/allocs and
// /sched/goroutines-created cumulative (COUNT_, SUM_), the others not
// (CURRENT_). If a name is no longer supported its value is reported as
// unsupported.
var runtimeNames = []struct{ column, name string }{
	{"CURRENT_MEMORY_BYTES", "/memory/classes/total:bytes"},
	{"CURRENT_HEAP_OBJECTS_BYTES", "/memory/classes/heap/objects:bytes"},
	{"SUM_HEAP_ALLOC_BYTES", "/gc/heap/allocs:bytes"},
	{"COUNT_GC_CYCLES", "/gc/cycles/total:gc-cycles"},
	{"CURRENT_GOROUTINES", "/sched/goroutines:goroutines"},
	{"COUNT_GOROUTINES_CREATED", "/sched/goroutines-created:goroutines"},
}

type collectorEntry struct {
	metric string
	stats  *Collector
}

type presenterEntry struct {
	view, style string
	stats       *Presenter
}

// Registry holds the entries of one run of ps-top.
type Registry struct {
	start      time.Time
	collectors []collectorEntry
	presenters []presenterEntry
	samples    []metrics.Sample
}

// New returns an empty registry for a run that started at start
// (START_TIME).
func New(start time.Time) *Registry {
	r := &Registry{start: start, samples: make([]metrics.Sample, len(runtimeNames))}
	for i, n := range runtimeNames {
		r.samples[i].Name = n.name
	}
	return r
}

// Collector returns a new entry for metric. Call it before collection
// starts. A metric has one entry.
func (r *Registry) Collector(metric string) (*Collector, error) {
	for _, e := range r.collectors {
		if e.metric == metric {
			return nil, fmt.Errorf("registry: collector %q already has an entry", metric)
		}
	}
	s := new(Collector)
	r.collectors = append(r.collectors, collectorEntry{metric: metric, stats: s})
	return s, nil
}

// Presenter returns a new entry for view shown in style. Call it before
// presenting starts. A view and style have one entry.
func (r *Registry) Presenter(view, style string) (*Presenter, error) {
	for _, e := range r.presenters {
		if e.view == view && e.style == style {
			return nil, fmt.Errorf("registry: presenter %q %q already has an entry", view, style)
		}
	}
	s := new(Presenter)
	r.presenters = append(r.presenters, presenterEntry{view: view, style: style, stats: s})
	return s, nil
}

// Report writes every entry's values to w, one table per set, at any width.
// now is the time of the report, for the time running. Report reads the Go
// runtime metrics, so it is not safe to call from several goroutines at once.
func (r *Registry) Report(w io.Writer, now time.Time) error {
	var b strings.Builder
	t := table{tw: tabwriter.NewWriter(&b, 0, 0, 1, ' ', tabwriter.AlignRight), b: &b}

	t.row("METRIC_NAME", "COUNT_QUERY", "SUM_TIMER_QUERY", "COUNT_QUERY_ERROR", "SUM_ROWS_READ",
		"COUNT_MERGE", "SUM_TIMER_MERGE", "SUM_ENTRIES", "COUNT_SNAPSHOT", "SUM_TIMER_SNAPSHOT",
		"CURRENT_SNAPSHOTS", "CURRENT_RING_CAPACITY")
	for _, e := range r.collectors {
		s := e.stats
		t.row(e.metric, count(&s.Query.Count), timer(&s.Query), count(&s.QueryErrors), count(&s.RowsRead),
			count(&s.Merge.Count), timer(&s.Merge), count(&s.Entries),
			count(&s.Snapshot.Count), timer(&s.Snapshot), count(&s.Snapshots), count(&s.RingCapacity))
	}
	t.end()

	t.row("VIEW_NAME", "VIEW_STYLE", "COUNT_RENDER", "SUM_TIMER_RENDER", "COUNT_WINDOW", "SUM_TIMER_WINDOW",
		"COUNT_MAP", "SUM_TIMER_MAP", "SUM_ENTRIES_MAPPED", "COUNT_DERIVE", "SUM_TIMER_DERIVE",
		"COUNT_ORDER", "SUM_TIMER_ORDER", "COUNT_DRAW", "SUM_TIMER_DRAW", "SUM_LINES_DRAWN",
		"CURRENT_ROWS_SHOWN", "CURRENT_ROWS_TOTAL")
	for _, e := range r.presenters {
		s := e.stats
		t.row(e.view, e.style, count(&s.Render.Count), timer(&s.Render), count(&s.Window.Count), timer(&s.Window),
			count(&s.Map.Count), timer(&s.Map), count(&s.EntriesMapped),
			count(&s.Derive.Count), timer(&s.Derive), count(&s.Order.Count), timer(&s.Order),
			count(&s.Draw.Count), timer(&s.Draw), count(&s.LinesDrawn), count(&s.RowsShown), count(&s.RowsTotal))
	}
	t.end()

	metrics.Read(r.samples)
	head := []string{"PROCESS", "START_TIME", "RUNNING"}
	row := []string{"ps-top", r.start.Format(time.RFC3339), now.Sub(r.start).Round(time.Millisecond).String()}
	for i, n := range runtimeNames {
		head = append(head, n.column)
		row = append(row, runtimeValue(r.samples[i].Value))
	}
	t.row(head...)
	t.row(row...)
	t.end()

	if t.err != nil {
		return t.err
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// table writes aligned rows into b, keeping the first error.
type table struct {
	tw  *tabwriter.Writer
	b   *strings.Builder
	err error
}

// row writes one row of cells.
func (t *table) row(cells ...string) {
	if t.err == nil {
		_, t.err = io.WriteString(t.tw, strings.Join(cells, "\t")+"\t\n")
	}
}

// end ends one table and leaves a blank line after it.
func (t *table) end() {
	if t.err == nil {
		t.err = t.tw.Flush()
	}
	t.b.WriteString("\n")
}

// count formats a counter or gauge.
func count(v *atomic.Uint64) string { return strconv.FormatUint(v.Load(), 10) }

// timer formats a stage's total time as views show times, and zero as "0".
func timer(s *Stage) string {
	ps := s.SumTimer.Load()
	if ps == 0 {
		return "0"
	}
	return strings.TrimSpace(formatter.Time(ps))
}

func runtimeValue(v metrics.Value) string {
	switch v.Kind() {
	case metrics.KindUint64:
		return fmt.Sprint(v.Uint64())
	case metrics.KindFloat64:
		return fmt.Sprint(v.Float64())
	case metrics.KindBad:
		return "unsupported"
	}
	return "unknown kind"
}

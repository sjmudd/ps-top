package registry

import (
	"runtime/metrics"
	"strings"
	"unique"

	"github.com/sjmudd/ps-top/v2/internal/metric"
	"github.com/sjmudd/ps-top/v2/internal/snapshot"
)

// The registry is collected as two metrics, so ps-top's own figures are
// held in rings and shown as differences over a window like any metric:
//
//   - StagesMetric: one entry per stage, keyed by the part of the pipeline,
//     the metric or the view and style, and the stage, e.g. (collect,
//     table_io, query) or (present, table_io_latency difference, render).
//     Columns COUNT, SUM_TIMER (picoseconds) and SUM_ROWS, the rows the
//     stage took in: rows read by a query, entries received by a render, 0
//     for merge and snapshot. Of presentation only renders are collected.
//   - ProcessMetric: one entry, (process), with the Global figures of the Go
//     runtime, as Report shows them.
const (
	StagesMetric  = "ps_top_stages"
	ProcessMetric = "ps_top_process"
)

// StagesDefinition is the definition of StagesMetric: counters, differences
// by default.
func StagesDefinition() metric.Definition {
	return metric.Definition{
		Name: StagesMetric,
		Columns: []metric.Column{
			{Name: "COUNT", Kind: metric.Counter},
			{Name: "SUM_TIMER", Kind: metric.Counter},
			{Name: "SUM_ROWS", Kind: metric.Counter},
		},
		Types: []metric.MetricType{metric.DifferenceValues, metric.CurrentValue},
	}
}

// ProcessDefinition is the definition of ProcessMetric: the runtimeNames
// columns, CURRENT_ ones gauges and the others counters. Current values by
// default. The GC cycle count detects a reset.
func ProcessDefinition() metric.Definition {
	d := metric.Definition{Name: ProcessMetric, Types: []metric.MetricType{metric.CurrentValue, metric.DifferenceValues}}
	for i, n := range runtimeNames {
		kind := metric.Counter
		if strings.HasPrefix(n.column, "CURRENT_") {
			kind = metric.Gauge
		}
		if n.column == "COUNT_GC_CYCLES" {
			d.ResetColumn = i
		}
		d.Columns = append(d.Columns, metric.Column{Name: n.column, Kind: kind})
	}
	return d
}

// Stages returns StagesMetric's entries: every collector's query, merge and
// snapshot stages, then every presenter's render. The read is recorded in
// st, its query stage, with the entries read as its rows.
func (r *Registry) Stages(st *Collector) []snapshot.Entry {
	t := st.Query.Start()
	var entries []snapshot.Entry
	add := func(part, name, stage string, s *Stage, rows uint64) {
		entries = append(entries, snapshot.Entry{
			Key:    unique.Make(snapshot.JoinKey(part, name, stage)),
			Values: []uint64{s.Count.Load(), s.SumTimer.Load(), rows},
		})
	}
	for _, e := range r.collectors {
		s := e.stats
		add("collect", e.metric, "query", &s.Query, s.RowsRead.Load())
		add("collect", e.metric, "merge", &s.Merge, 0)
		add("collect", e.metric, "snapshot", &s.Snapshot, 0)
	}
	for _, e := range r.presenters {
		s := e.stats
		add("present", e.view+" "+e.style, "render", &s.Render, s.EntriesMapped.Load())
	}
	t.Stop()
	st.RowsRead.Add(uint64(len(entries)))
	st.Entries.Add(uint64(len(entries)))
	return entries
}

// Process returns ProcessMetric's one entry, the Go runtime's figures now. A
// figure the runtime no longer supports is 0. The read is recorded in st,
// its query stage.
func (r *Registry) Process(st *Collector) []snapshot.Entry {
	t := st.Query.Start()
	samples := make([]metrics.Sample, len(runtimeNames)) // r.samples is Report's
	for i, n := range runtimeNames {
		samples[i].Name = n.name
	}
	metrics.Read(samples)
	values := make([]uint64, len(samples))
	for i, s := range samples {
		if s.Value.Kind() == metrics.KindUint64 {
			values[i] = s.Value.Uint64()
		}
	}
	t.Stop()
	st.RowsRead.Add(1)
	st.Entries.Add(1)
	return []snapshot.Entry{{Key: unique.Make(snapshot.JoinKey("process")), Values: values}}
}

package registry

import (
	"strings"
	"testing"
	"time"
)

func TestTimerAddsToStage(t *testing.T) {
	var s Stage
	for range 3 {
		timer := s.Start()
		time.Sleep(time.Millisecond)
		timer.Stop()
	}
	if got := s.Count.Load(); got != 3 {
		t.Errorf("Count = %d, want 3", got)
	}
	// At least 3 ms, in picoseconds: counters are added to, not overwritten.
	if got := s.SumTimer.Load(); got < 3*1000*1000*1000 {
		t.Errorf("SumTimer = %d ps, want at least 3 ms", got)
	}
}

func TestOneEntryEach(t *testing.T) {
	r := New(time.Now())
	if _, err := r.Collector("table_io"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Collector("table_io"); err == nil {
		t.Error("second collector entry for table_io: no error")
	}
	if _, err := r.Presenter("table_io_latency", "DifferenceValues"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Presenter("table_io_latency", "CurrentValue"); err != nil {
		t.Errorf("same view, other style: %v", err)
	}
	if _, err := r.Presenter("table_io_latency", "DifferenceValues"); err == nil {
		t.Error("second presenter entry for table_io_latency DifferenceValues: no error")
	}
}

func TestReport(t *testing.T) {
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	r := New(start)
	c, err := r.Collector("table_io")
	if err != nil {
		t.Fatal(err)
	}
	c.Query.Count.Add(4)
	c.Query.SumTimer.Add(8_000_000_000) // 8 ms
	c.RowsRead.Add(120)
	c.Snapshots.Store(5)
	c.RingCapacity.Store(11)
	p, err := r.Presenter("table_io_latency", "DifferenceValues")
	if err != nil {
		t.Fatal(err)
	}
	p.Render.Count.Add(2)
	p.RowsShown.Store(7)

	var b strings.Builder
	if err := r.Report(&b, start.Add(90*time.Second)); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{
		"METRIC_NAME", "SUM_TIMER_QUERY", "CURRENT_RING_CAPACITY",
		"VIEW_NAME", "VIEW_STYLE", "CURRENT_ROWS_SHOWN",
		"PROCESS", "START_TIME", "RUNNING", "CURRENT_MEMORY_BYTES", "COUNT_GC_CYCLES",
		"2026-01-02T03:04:05Z", "1m30s", "8.00 ms",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report has no %q:\n%s", want, out)
		}
	}
	// The collector row: name, then the values in column order
	// (COUNT_QUERY 4, SUM_TIMER_QUERY "8.00 ms", COUNT_QUERY_ERROR 0,
	// SUM_ROWS_READ 120, ...).
	var row []string
	for line := range strings.Lines(out) {
		if f := strings.Fields(line); len(f) > 0 && f[0] == "table_io" {
			row = f
		}
	}
	if len(row) < 6 || row[1] != "4" || row[2]+" "+row[3] != "8.00 ms" || row[4] != "0" || row[5] != "120" {
		t.Errorf("collector row = %q, want table_io 4 8.00 ms 0 120 ...", row)
	}
	if strings.Contains(out, "unsupported") {
		t.Errorf("a runtime metric name is not supported by this Go version:\n%s", out)
	}
}

package derive

import (
	"math"
	"testing"
	"time"
	"unique"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/snapshot"
	"github.com/sjmudd/ps-top/internal/store"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// counters: one counter column, used as the reset column.
var counters = metric.Definition{
	Name:    "io",
	Columns: []metric.Column{{Name: "sum_timer_wait", Kind: metric.Counter}},
}

// point is the input for one snapshot: time offset and rows.
type point struct {
	at   time.Duration
	rows []snapshot.Row
}

func r(key string, values ...uint64) snapshot.Row {
	return snapshot.Row{Key: unique.Make(key), Values: values}
}

func sec(n float64) time.Duration {
	return time.Duration(n * float64(time.Second))
}

func ring(t *testing.T, width int, points ...point) *store.Ring {
	t.Helper()
	rg, err := store.New(max(len(points), 2))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range points {
		s, err := snapshot.New(t0.Add(p.at), width, p.rows)
		if err != nil {
			t.Fatal(err)
		}
		if err := rg.Add(s); err != nil {
			t.Fatal(err)
		}
	}
	return rg
}

// values returns key -> first value of each derived row.
func values(res Difference) map[string]uint64 {
	m := make(map[string]uint64, len(res.Rows))
	for _, row := range res.Rows {
		m[row.Key.Value()] = row.Values[0]
	}
	return m
}

func window(t *testing.T, def metric.Definition, rg *store.Ring, n int) Difference {
	t.Helper()
	res, err := Window(def, rg, n)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestNotReady(t *testing.T) {
	res := window(t, counters, ring(t, 1, point{0, []snapshot.Row{r("a", 5)}}), 10)
	if res.Ready || len(res.Rows) != 0 {
		t.Errorf("one snapshot: Ready=%v rows=%d, want not ready and no rows", res.Ready, len(res.Rows))
	}
}

func TestSlidingWindow(t *testing.T) {
	var pts []point
	for s := 0; s <= 15; s++ {
		pts = append(pts, point{sec(float64(s)), []snapshot.Row{r("a", uint64(s*10))}})
	}
	res := window(t, counters, ring(t, 1, pts...), 10)
	if !res.Ready || res.Reset {
		t.Fatalf("Ready=%v Reset=%v", res.Ready, res.Reset)
	}
	if got := values(res)["a"]; got != 100 {
		t.Errorf("difference = %d, want 100 (150 - 50)", got)
	}
	if !res.From.Equal(t0.Add(sec(5))) || res.Elapsed() != sec(10) {
		t.Errorf("From=%v Elapsed=%v, want +5s and 10s", res.From, res.Elapsed())
	}
}

func TestHistoryShorterThanWindow(t *testing.T) {
	res := window(t, counters, ring(t, 1,
		point{0, []snapshot.Row{r("a", 10)}},
		point{sec(1), []snapshot.Row{r("a", 15)}},
		point{sec(2), []snapshot.Row{r("a", 30)}},
	), 10)
	if got := values(res)["a"]; got != 20 {
		t.Errorf("difference = %d, want 20 (from oldest)", got)
	}
	if res.Elapsed() != sec(2) {
		t.Errorf("Elapsed = %v, want 2s", res.Elapsed())
	}
}

func TestWindowStartBySnapshotCount(t *testing.T) {
	// Irregular snapshot times do not matter: n=2 is two snapshots back.
	res := window(t, counters, ring(t, 1,
		point{0, []snapshot.Row{r("a", 0)}},
		point{sec(0.3), []snapshot.Row{r("a", 10)}},
		point{sec(2.9), []snapshot.Row{r("a", 15)}},
		point{sec(3), []snapshot.Row{r("a", 40)}},
	), 2)
	if !res.From.Equal(t0.Add(sec(0.3))) || res.Elapsed() != sec(2.7) {
		t.Errorf("From = %v Elapsed = %v, want +0.3s and 2.7s", res.From.Sub(t0), res.Elapsed())
	}
	if got := values(res)["a"]; got != 30 {
		t.Errorf("difference = %d, want 30", got)
	}
}

func TestNewRows(t *testing.T) {
	res := window(t, counters, ring(t, 1,
		point{0, []snapshot.Row{r("old", 10)}},
		point{sec(1), []snapshot.Row{r("old", 68), r("new", 7)}},
	), 10)
	if res.Reset {
		t.Fatal("Reset = true, want false")
	}
	got := values(res)
	if got["old"] != 58 {
		t.Errorf("old = %d, want 58", got["old"])
	}
	if got["new"] != 7 {
		t.Errorf("new = %d, want 7 (no earlier value: subtract 0)", got["new"])
	}
}

func TestLowerSnapshotTotalIsReset(t *testing.T) {
	// Each snapshot's total is over all its rows: 60 then 32 is a reset.
	res := window(t, counters, ring(t, 1,
		point{0, []snapshot.Row{r("old", 10), r("gone", 50)}},
		point{sec(1), []snapshot.Row{r("old", 25), r("new", 7)}},
	), 10)
	if !res.Reset || res.Elapsed() != 0 {
		t.Errorf("Reset=%v Elapsed=%v, want reset to latest", res.Reset, res.Elapsed())
	}
	if got := values(res); got["old"] != 0 || got["new"] != 0 {
		t.Errorf("old=%d new=%d, want 0 and 0", got["old"], got["new"])
	}
}

func TestSingleRowDropIsZero(t *testing.T) {
	// Total rises, so no reset; one row's counter restarted (e.g. table recreated).
	res := window(t, counters, ring(t, 1,
		point{0, []snapshot.Row{r("a", 100), r("b", 10)}},
		point{sec(1), []snapshot.Row{r("a", 30), r("b", 500)}},
	), 10)
	if res.Reset {
		t.Error("Reset = true, want false")
	}
	if got := values(res); got["a"] != 0 || got["b"] != 490 {
		t.Errorf("a=%d b=%d, want 0 and 490", got["a"], got["b"])
	}
}

func TestResetMovesStart(t *testing.T) {
	var pts []point
	for s := 0; s <= 4; s++ {
		pts = append(pts, point{sec(float64(s)), []snapshot.Row{r("a", 1000)}})
	}
	// TRUNCATE at +5s, then activity resumes.
	for s, v := range []uint64{0, 100, 200, 300, 400, 500} {
		pts = append(pts, point{sec(float64(5 + s)), []snapshot.Row{r("a", v)}})
	}
	res := window(t, counters, ring(t, 1, pts...), 10)
	if !res.Reset {
		t.Error("Reset = false, want true")
	}
	if !res.From.Equal(t0.Add(sec(5))) {
		t.Errorf("From = %v, want +5s (first total <= latest)", res.From.Sub(t0))
	}
	if got := values(res)["a"]; got != 500 {
		t.Errorf("difference = %d, want 500", got)
	}
}

func TestResetToLatestGivesZero(t *testing.T) {
	res := window(t, counters, ring(t, 1,
		point{0, []snapshot.Row{r("a", 1000)}},
		point{sec(1), []snapshot.Row{r("a", 1000)}},
		point{sec(2), []snapshot.Row{r("a", 50)}},
	), 10)
	if !res.Ready || !res.Reset {
		t.Fatalf("Ready=%v Reset=%v, want both true", res.Ready, res.Reset)
	}
	if res.Elapsed() != 0 {
		t.Errorf("Elapsed = %v, want 0 (window restarts at latest)", res.Elapsed())
	}
	if got := values(res)["a"]; got != 0 {
		t.Errorf("difference = %d, want 0", got)
	}
}

func TestGauge(t *testing.T) {
	def := metric.Definition{
		Name: "memory",
		Columns: []metric.Column{
			{Name: "count_alloc", Kind: metric.Counter},
			{Name: "current_bytes", Kind: metric.Gauge},
		},
	}
	res := window(t, def, ring(t, 2,
		point{0, []snapshot.Row{r("buf", 10, 800), r("tmp", 1, 50)}},
		point{sec(1), []snapshot.Row{r("buf", 12, 600), r("new", 1, 40)}},
	), 10)
	byKey := map[string]Row{}
	for _, row := range res.Rows {
		byKey[row.Key.Value()] = row
	}
	buf := byKey["buf"]
	if buf.Values[0] != 2 || buf.Values[1] != 600 || buf.Change[1] != -200 {
		t.Errorf("buf = %+v, want counter 2, gauge 600 change -200", buf)
	}
	if n := byKey["new"]; n.Values[1] != 40 || n.Change[1] != 40 {
		t.Errorf("new = %+v, want gauge 40 change +40 (no start row)", n)
	}
	if res.Totals[1] != 640 {
		t.Errorf("gauge total = %d, want 640", res.Totals[1])
	}
}

func TestTotalsAndPercent(t *testing.T) {
	res := window(t, counters, ring(t, 1,
		point{0, []snapshot.Row{r("a", 0), r("b", 0)}},
		point{sec(1), []snapshot.Row{r("a", 30), r("b", 10)}},
	), 10)
	if res.Totals[0] != 40 {
		t.Fatalf("total = %d, want 40", res.Totals[0])
	}
	for i, row := range res.Rows {
		want := map[string]float64{"a": 75, "b": 25}[row.Key.Value()]
		if got := res.Percent(i, 0); math.Abs(got-want) > 1e-9 {
			t.Errorf("%s percent = %v, want %v", row.Key.Value(), got, want)
		}
	}
}

func TestPercentZeroTotal(t *testing.T) {
	res := window(t, counters, ring(t, 1,
		point{0, []snapshot.Row{r("a", 5)}},
		point{sec(1), []snapshot.Row{r("a", 5)}},
	), 10)
	if got := res.Percent(0, 0); got != 0 {
		t.Errorf("percent with zero total = %v, want 0", got)
	}
}

func TestErrors(t *testing.T) {
	rg := ring(t, 2,
		point{0, []snapshot.Row{r("a", 1, 1)}},
		point{sec(1), []snapshot.Row{r("a", 2, 2)}},
	)
	if _, err := Window(counters, rg, 10); err == nil {
		t.Error("width mismatch: no error")
	}
	if _, err := Window(counters, ring(t, 1), 0); err == nil {
		t.Error("zero window: no error")
	}
	if _, err := Window(metric.Definition{}, ring(t, 1), 10); err == nil {
		t.Error("invalid definition: no error")
	}
}

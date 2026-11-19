package derive

import (
	"testing"
	"time"
	"unique"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/snapshot"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// counters: one counter column, used as the reset column.
var counters = metric.Definition{
	Name:    "io",
	Columns: []metric.Column{{Name: "sum_timer_wait", Kind: metric.Counter}},
}

func r(key string, values ...uint64) snapshot.Entry {
	return snapshot.Entry{Key: unique.Make(key), Values: values}
}

func sec(n float64) time.Duration {
	return time.Duration(n * float64(time.Second))
}

func snap(t *testing.T, at time.Duration, width int, entries ...snapshot.Entry) *snapshot.Snapshot {
	t.Helper()
	s, err := snapshot.New(t0.Add(at), width, entries)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// values returns key -> first value of each derived entry.
func values(res Difference) map[string]uint64 {
	m := make(map[string]uint64, len(res.Entries))
	for _, e := range res.Entries {
		m[e.Key.Value()] = e.Values[0]
	}
	return m
}

func window(t *testing.T, def metric.Definition, start, end *snapshot.Snapshot) Difference {
	t.Helper()
	res, err := Window(def, start, end)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestDifference(t *testing.T) {
	res := window(t, counters, snap(t, sec(5), 1, r("a", 50)), snap(t, sec(15), 1, r("a", 150)))
	if got := values(res)["a"]; got != 100 {
		t.Errorf("difference = %d, want 100 (150 - 50)", got)
	}
	if !res.From.Equal(t0.Add(sec(5))) || res.Elapsed() != sec(10) {
		t.Errorf("From=%v Elapsed=%v, want +5s and 10s", res.From.Sub(t0), res.Elapsed())
	}
}

func TestNewEntries(t *testing.T) {
	res := window(t, counters,
		snap(t, 0, 1, r("old", 10)),
		snap(t, sec(1), 1, r("old", 68), r("new", 7)))
	got := values(res)
	if got["old"] != 58 {
		t.Errorf("old = %d, want 58", got["old"])
	}
	if got["new"] != 7 {
		t.Errorf("new = %d, want 7 (no earlier value: subtract 0)", got["new"])
	}
}

func TestLowerTotalIsReset(t *testing.T) {
	// Each snapshot's total is over all its entries: 60 then 32 is a reset, so
	// the end values are the increase.
	res := window(t, counters,
		snap(t, 0, 1, r("old", 10), r("gone", 50)),
		snap(t, sec(1), 1, r("old", 25), r("new", 7)))
	if got := values(res); got["old"] != 25 || got["new"] != 7 {
		t.Errorf("old=%d new=%d, want 25 and 7", got["old"], got["new"])
	}
	if res.Totals[0] != 32 {
		t.Errorf("total = %d, want 32", res.Totals[0])
	}
}

func TestSingleEntryDropIsZero(t *testing.T) {
	// Total rises, so no reset; one entry's counter restarted (e.g. table recreated).
	res := window(t, counters,
		snap(t, 0, 1, r("a", 100), r("b", 10)),
		snap(t, sec(1), 1, r("a", 30), r("b", 500)))
	if got := values(res); got["a"] != 0 || got["b"] != 490 {
		t.Errorf("a=%d b=%d, want 0 and 490", got["a"], got["b"])
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
	res := window(t, def,
		snap(t, 0, 2, r("buf", 10, 800), r("tmp", 1, 50)),
		snap(t, sec(1), 2, r("buf", 12, 600), r("new", 1, 40)))
	byKey := map[string]Entry{}
	for _, e := range res.Entries {
		byKey[e.Key.Value()] = e
	}
	buf := byKey["buf"]
	if buf.Values[0] != 2 || buf.Values[1] != 600 || buf.Change[1] != -200 {
		t.Errorf("buf = %+v, want counter 2, gauge 600 change -200", buf)
	}
	if n := byKey["new"]; n.Values[1] != 40 || n.Change[1] != 40 {
		t.Errorf("new = %+v, want gauge 40 change +40 (no start entry)", n)
	}
	if res.Totals[1] != 640 {
		t.Errorf("gauge total = %d, want 640", res.Totals[1])
	}
}

func TestTotals(t *testing.T) {
	res := window(t, counters,
		snap(t, 0, 1, r("a", 0), r("b", 0)),
		snap(t, sec(1), 1, r("a", 30), r("b", 10)))
	if res.Totals[0] != 40 {
		t.Fatalf("total = %d, want 40", res.Totals[0])
	}
}

func TestErrors(t *testing.T) {
	one := snap(t, 0, 1, r("a", 1))
	two := snap(t, sec(1), 2, r("a", 2, 2))
	if _, err := Window(counters, one, two); err == nil {
		t.Error("width mismatch: no error")
	}
	if _, err := Window(counters, nil, one); err == nil {
		t.Error("nil start: no error")
	}
	if _, err := Window(metric.Definition{}, one, one); err == nil {
		t.Error("invalid definition: no error")
	}
}

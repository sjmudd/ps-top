package presenter

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
	"unique"

	"github.com/sjmudd/ps-top/internal/collector"
	"github.com/sjmudd/ps-top/internal/snapshot"
)

// fake returns a fixed window.
type fake struct{ w collector.Window }

func (f fake) Window(context.Context, string, int) (collector.Window, error) { return f.w, nil }

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// entry builds a table I/O entry from count_star and sum_timer_wait, with all
// reads being fetches.
func entry(key string, count, wait uint64) snapshot.Entry {
	v := make([]uint64, 14)
	v[0], v[1] = count, wait // star
	v[2], v[3] = count, wait // read
	v[6], v[7] = count, wait // fetch
	return snapshot.Entry{Key: unique.Make(key), Values: v}
}

func snap(t *testing.T, at time.Duration, entries ...snapshot.Entry) *snapshot.Snapshot {
	t.Helper()
	s, err := snapshot.New(t0.Add(at), 14, entries)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestHeadingsMatchOldFormat(t *testing.T) {
	want := fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s|%s", "Latency", "%", "Read", "Write", "Fetch", "Insert", "Update", "Delete", "Table Name")
	if got := TableIOLatency.headings(); got != want {
		t.Errorf("headings\n got %q\nwant %q", got, want)
	}
}

func TestRender(t *testing.T) {
	src := fake{collector.Window{
		Start: snap(t, 0, entry("db.a", 10, 1000000000), entry("db.b", 5, 0), entry("db.idle", 1, 5)),
		End:   snap(t, time.Second, entry("db.a", 30, 4000000000), entry("db.b", 15, 1000000000), entry("db.idle", 1, 5)),
		OK:    true,
	}}
	lines, err := Render(context.Background(), src, TableIOLatency, 5, 80)
	if err != nil {
		t.Fatal(err)
	}
	// description, headings, db.a, db.b, totals; db.idle has no change.
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if lines[0] != "Table I/O Latency (table_io_waits_summary_by_table) 2 rows" {
		t.Errorf("description = %q", lines[0])
	}
	want := fmt.Sprintf("%10s %6s|%6s %6s|%6s %6s %6s %6s|%s", "   3.00 ms", " 75.0%", "100.0%", "", "100.0%", "", "", "", "db.a")
	if lines[2] != want {
		t.Errorf("first line\n got %q\nwant %q", lines[2], want)
	}
	if !strings.HasSuffix(lines[3], "|db.b") || !strings.HasSuffix(lines[4], "|Totals") {
		t.Errorf("order: %q, %q", lines[3], lines[4])
	}
	if !strings.HasPrefix(lines[4], "   4.00 ms 100.0%") {
		t.Errorf("totals = %q", lines[4])
	}
}

func TestRenderCutsAtWidth(t *testing.T) {
	long := "db." + strings.Repeat("x", 100)
	src := fake{collector.Window{
		Start: snap(t, 0, entry(long, 0, 0)),
		End:   snap(t, time.Second, entry(long, 1, 1000)),
		OK:    true,
	}}
	lines, err := Render(context.Background(), src, TableIOOps, 5, 80)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range lines {
		if n := len([]rune(l)); n > 80 {
			t.Errorf("line of %d runes: %q", n, l)
		}
	}
}

func TestRenderNotReady(t *testing.T) {
	lines, err := Render(context.Background(), fake{}, TableIOOps, 5, 80)
	if err != nil || len(lines) != 1 || !strings.Contains(lines[0], "waiting") {
		t.Errorf("not ready: %q %v", lines, err)
	}
}

func TestRenderUnknownColumn(t *testing.T) {
	v := TableIOOps
	v.Total = "nope"
	if _, err := Render(context.Background(), fake{}, v, 5, 80); err == nil {
		t.Error("unknown column: no error")
	}
}

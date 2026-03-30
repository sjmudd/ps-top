package tableioops

import (
	"strings"
	"testing"

	"github.com/sjmudd/ps-top/model/tableio"
	"github.com/sjmudd/ps-top/wrapper/tableiolatency"
)

// newTableIo creates a Wrapper for testing with the given rows and totals.
// It constructs a latency wrapper and derives the ops wrapper from it,
// injecting the test data into the shared underlying TableIo model.
func newTableIo(rows []tableio.Row, totals tableio.Row) *Wrapper {
	// Create a latency wrapper with nil config/db (sufficient for tests).
	latency := tableiolatency.NewTableIoLatency(nil, nil)
	// Inject test data into the underlying TableIo model.
	model := latency.GetModel()
	model.Results = rows
	model.Totals = totals
	// Create ops wrapper using the latency wrapper.
	return NewTableIoOps(latency)
}

// TestRowContentUsesCounts verifies that RowContent uses CountStar and Count* fields
// to display operations counts and percentages.
func TestRowContentUsesCounts(t *testing.T) {
	rows := []tableio.Row{
		{Name: "db1.t1", CountStar: 100, CountFetch: 30},
		{Name: "db2.t2", CountStar: 100, CountFetch: 50},
	}
	totals := tableio.Row{CountStar: 200}
	w := newTableIo(rows, totals)

	lines := w.RowContent()
	if len(lines) != 2 {
		t.Fatalf("RowContent returned %d rows, want 2", len(lines))
	}

	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) != 4 {
			t.Fatalf("expected 4 parts, got %d: %q", len(parts), line)
		}
		left := parts[0]

		if !strings.Contains(left, "100") {
			t.Errorf("missing count 100 in left: %q", left)
		}
		if !strings.Contains(left, "50.0%") {
			t.Errorf("missing total %% 50.0%% in left: %q", left)
		}
	}
}

// TestHeadings checks that the headings contain the "Ops" label.
func TestHeadings(t *testing.T) {
	// Construct a wrapper using a latency wrapper (with nil args).
	latency := tableiolatency.NewTableIoLatency(nil, nil)
	w := NewTableIoOps(latency)
	h := w.Headings()
	if !strings.Contains(h, "Ops") {
		t.Errorf("Headings missing 'Ops': %q", h)
	}
}

// TestDescription verifies that Description includes the expected "Ops" label.
func TestDescription(t *testing.T) {
	rows := []tableio.Row{{Name: "db.t", CountStar: 100}}
	w := newTableIo(rows, tableio.Row{})
	d := w.Description()
	if !strings.Contains(d, "Ops") {
		t.Errorf("Description missing 'Ops': %q", d)
	}
}

// TestRowContentOperationPercentages validates that Fetch/Insert/Update/Delete
// percentages are computed correctly from Count* fields divided by row CountStar.
func TestRowContentOperationPercentages(t *testing.T) {
	row := tableio.Row{
		Name:        "db.t",
		CountStar:   270,
		CountFetch:  100,
		CountInsert: 50,
		CountUpdate: 30,
		CountDelete: 20,
		CountRead:   150,
		CountWrite:  120,
	}
	totals := tableio.Row{CountStar: 270}
	w := newTableIo([]tableio.Row{row}, totals)

	line := w.RowContent()[0]
	parts := strings.Split(line, "|")
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts, got %d: %q", len(parts), line)
	}
	mid := parts[2]

	expPcts := []string{"37.0%", "18.5%", "11.1%", "7.4%"}
	for _, exp := range expPcts {
		if !strings.Contains(mid, exp) {
			t.Errorf("missing expected percentage %s in mid: %q", exp, mid)
		}
	}
}

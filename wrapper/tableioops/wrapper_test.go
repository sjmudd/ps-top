package tableioops

import (
	"strings"
	"testing"

	"github.com/sjmudd/ps-top/model/tableio"
)

// TestRowContentUsesCounts verifies that RowContent uses CountStar and Count* fields, not SumTimer*.
func TestRowContentUsesCounts(t *testing.T) {
	// Two rows to ensure totals match sum of CountStar
	rows := []tableio.Row{
		{Name: "db1.t1", CountStar: 100, CountFetch: 30},
		{Name: "db2.t2", CountStar: 100, CountFetch: 50},
	}
	totals := tableio.Row{CountStar: 200} // sum of rows
	tiol := &tableio.TableIo{Results: rows, Totals: totals}
	w := &Wrapper{tiol: tiol}

	lines := w.RowContent()
	if len(lines) != 2 {
		t.Fatalf("RowContent returned %d rows, want 2", len(lines))
	}

	// Inspect each line's columns to ensure correct calculation per row.
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			t.Fatalf("expected 3 parts, got %d: %q", len(parts), line)
		}
		left := parts[0] // contains count, total %, right-aligned 10-char count then 6-char %

		// Each row's CountStar should be 100 and appear as right-aligned in 10-char field.
		if !strings.Contains(left, "100") {
			t.Errorf("missing count 100 in left: %q", left)
		}
		// Total percentage (row.CountStar/totals.CountStar) = 100/200 = 50.0%
		if !strings.Contains(left, "50.0%") {
			t.Errorf("missing total %% 50.0%% in left: %q", left)
		}
		// Extract the name.
		name := strings.TrimSpace(parts[2])
		if name != "db1.t1" && name != "db2.t2" {
			t.Errorf("unexpected name: %q", name)
		}
	}

	// Now verify fetch percentages: one line should have 30.0% (30/100) and other 50.0% (50/100).
	mid1 := strings.Split(lines[0], "|")[1]
	mid2 := strings.Split(lines[1], "|")[1]
	// Both should contain 50.0% mark for fetch? Actually row2 fetch% = 50%, row1 fetch% = 30%.
	has30 := strings.Contains(mid1, "30.0%") || strings.Contains(mid2, "30.0%")
	has50inmid := strings.Contains(mid1, "50.0%") || strings.Contains(mid2, "50.0%")
	if !has30 || !has50inmid {
		t.Errorf("missing expected fetch percentages. mid1=%q mid2=%q", mid1, mid2)
	}
}

// TestHeadings checks that headings contain "Ops".
func TestHeadings(t *testing.T) {
	w := &Wrapper{}
	h := w.Headings()
	if !strings.Contains(h, "Ops") {
		t.Errorf("Headings missing 'Ops': %q", h)
	}
}

// TestDescription checks that description contains "Ops".
func TestDescription(t *testing.T) {
	rows := []tableio.Row{{Name: "db.t", CountStar: 100}}
	w := &Wrapper{tiol: &tableio.TableIo{Results: rows}}
	d := w.Description()
	if !strings.Contains(d, "Ops") {
		t.Errorf("Description missing 'Ops': %q", d)
	}
}

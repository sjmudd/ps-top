package tableioops

import (
	"strings"
	"testing"

	"github.com/sjmudd/ps-top/model/tableio"
)

// TestRowContentUsesCounts verifies that RowContent uses CountStar and Count* fields.
func TestRowContentUsesCounts(t *testing.T) {
	rows := []tableio.Row{
		{Name: "db1.t1", CountStar: 100, CountFetch: 30},
		{Name: "db2.t2", CountStar: 100, CountFetch: 50},
	}
	totals := tableio.Row{CountStar: 200}
	tiol := &tableio.TableIo{Results: rows, Totals: totals}
	w := &Wrapper{tiol: tiol}

	lines := w.RowContent()
	if len(lines) != 2 {
		t.Fatalf("RowContent returned %d rows, want 2", len(lines))
	}

	// Inspect each line's columns.
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) != 4 {
			t.Fatalf("expected 4 parts, got %d: %q", len(parts), line)
		}
		left := parts[0]

		// Each row's CountStar is 100.
		if !strings.Contains(left, "100") {
			t.Errorf("missing count 100 in left: %q", left)
		}
		// Total percentage = 50.0%
		if !strings.Contains(left, "50.0%") {
			t.Errorf("missing total %% 50.0%% in left: %q", left)
		}
	}
}

// TestHeadings checks that headings contain "Ops".
func TestHeadings(t *testing.T) {
	h := (&Wrapper{}).Headings()
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

// TestRowContentOperationPercentages verifies that Fetch/Insert/Update/Delete percentages
// are calculated from Count* fields divided by row.CountStar.
// It uses realistic values satisfying MySQL constraints:
// - CountStar = CountRead + CountWrite
// - CountRead >= CountFetch
// - CountWrite >= CountInsert + CountUpdate + CountDelete
func TestRowContentOperationPercentages(t *testing.T) {
	// Realistic distribution:
	// CountFetch=100 (part of read), other reads=50 -> CountRead=150 (>= fetch)
	// CountInsert=50, CountUpdate=30, CountDelete=20 -> sum=100, plus write overhead=20 -> CountWrite=120
	// CountStar = 150 + 120 = 270
	row := tableio.Row{
		Name:        "db.t",
		CountStar:   270,
		CountFetch:  100, // 100/270 ≈ 37.0%
		CountInsert: 50,  // 18.5%
		CountUpdate: 30,  // 11.1%
		CountDelete: 20,  // 7.4%
		CountRead:   150, // ≥ CountFetch
		CountWrite:  120, // ≥ insert+update+delete (100)
	}
	totals := tableio.Row{CountStar: 270}
	tiol := &tableio.TableIo{Results: []tableio.Row{row}, Totals: totals}
	w := &Wrapper{tiol: tiol}

	line := w.RowContent()[0]
	parts := strings.Split(line, "|")
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts, got %d: %q", len(parts), line)
	}
	// parts[1] contains read%, write%; parts[2] contains fetch%, insert%, update%, delete%
	mid := parts[2]

	// Expected percentages (rounded):
	expPcts := []string{"37.0%", "18.5%", "11.1%", "7.4%"}
	for _, exp := range expPcts {
		if !strings.Contains(mid, exp) {
			t.Errorf("missing expected percentage %s in mid: %q", exp, mid)
		}
	}
}

package tableiolatency

import (
	"strings"
	"testing"

	"github.com/sjmudd/ps-top/model/tableio"
)

// TestRowContentUsesSumTimerWait verifies that RowContent produces output based on SumTimerWait.
func TestRowContentUsesSumTimerWait(t *testing.T) {
	// Create multiple rows with SumTimerWait values
	rows := []tableio.Row{
		{Name: "db1.t1", CountStar: 1, SumTimerWait: 1000000}, // 1ms, should be 25%
		{Name: "db2.t2", CountStar: 1, SumTimerWait: 3000000}, // 3ms, should be 75%
	}
	// Sum = 4ms
	totals := tableio.Row{SumTimerWait: 4000000}
	tiol := &tableio.TableIo{Results: rows, Totals: totals}
	w := &Wrapper{tiol: tiol}

	lines := w.RowContent()
	if len(lines) != 2 {
		t.Fatalf("RowContent returned %d rows, want 2", len(lines))
	}

	// Combine all lines to search for expected values regardless of order.
	all := strings.Join(lines, " ")

	// Should contain 1ms time and 25% for the smaller row.
	if !strings.Contains(all, "1.00") || !strings.Contains(all, "25.0%") {
		t.Errorf("output missing 1ms/25%%: %q", all)
	}
	// Should contain 3ms time and 75% for the larger row.
	if !strings.Contains(all, "3.00") || !strings.Contains(all, "75.0%") {
		t.Errorf("output missing 3ms/75%%: %q", all)
	}
	// Both table names present.
	if !strings.Contains(all, "db1.t1") || !strings.Contains(all, "db2.t2") {
		t.Errorf("missing table names: %q", all)
	}
}

// TestHeadings checks that headings contain "Latency".
func TestHeadings(t *testing.T) {
	w := &Wrapper{}
	h := w.Headings()
	if !strings.Contains(h, "Latency") {
		t.Errorf("Headings missing 'Latency': %q", h)
	}
}

// TestDescription checks that description contains "Latency".
func TestDescription(t *testing.T) {
	rows := []tableio.Row{{Name: "db.t", SumTimerWait: 1000}}
	w := &Wrapper{tiol: &tableio.TableIo{Results: rows}}
	d := w.Description()
	if !strings.Contains(d, "Latency") {
		t.Errorf("Description missing 'Latency': %q", d)
	}
}

// TestRowContentOperationPercentages verifies that Fetch/Insert/Update/Delete percentages
// are calculated from SumTimer* fields divided by row.SumTimerWait.
// It uses realistic values satisfying MySQL constraints:
// - SumTimerWait = SumTimerRead + SumTimerWrite
// - SumTimerRead >= SumTimerFetch
// - SumTimerWrite >= SumTimerInsert + SumTimerUpdate + SumTimerDelete
func TestRowContentOperationPercentages(t *testing.T) {
	// Realistic distribution:
	// Fetch=250 (part of read), other reads=50 -> SumTimerRead=300 (>= fetch)
	// Insert=100, Update=50, Delete=50 -> sum=200, plus write overhead=50 -> SumTimerWrite=250
	// SumTimerWait = 300+250 = 550
	row := tableio.Row{
		Name:           "db.t",
		CountStar:      1,
		SumTimerWait:   550,
		SumTimerFetch:  250, // 250/550 ≈ 45.5%
		SumTimerInsert: 100, // 18.2%
		SumTimerUpdate: 50,  // 9.1%
		SumTimerDelete: 50,  // 9.1%
		SumTimerRead:   300, // read total ≥ fetch
		SumTimerWrite:  250, // write total ≥ insert+update+delete (200)
	}
	totals := tableio.Row{SumTimerWait: 550}
	tiol := &tableio.TableIo{Results: []tableio.Row{row}, Totals: totals}
	w := &Wrapper{tiol: tiol}

	line := w.RowContent()[0]
	parts := strings.Split(line, "|")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d: %q", len(parts), line)
	}
	mid := parts[1]

	// Expected percentages (rounded to 1 decimal):
	// fetch=45.5%, insert=18.2%, update=9.1%, delete=9.1%
	expPcts := []string{"45.5%", "18.2%", "9.1%", "9.1%"}
	for _, exp := range expPcts {
		if !strings.Contains(mid, exp) {
			t.Errorf("missing expected percentage %s in mid: %q", exp, mid)
		}
	}
}

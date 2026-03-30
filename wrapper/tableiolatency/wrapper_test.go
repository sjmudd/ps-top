package tableiolatency

import (
	"strings"
	"testing"

	"github.com/sjmudd/ps-top/model"
	"github.com/sjmudd/ps-top/model/tableio"
	"github.com/sjmudd/ps-top/wrapper"
)

// newTableIo creates a Wrapper for testing with the given rows and totals.
// This helper constructs a TableIo model with injected data and wraps it
// using BaseWrapper with the default functions defined in this package.
func newTableIo(rows []tableio.Row, totals tableio.Row) *Wrapper {
	// Create a TableIo with a BaseCollector, manually set Results/Totals.
	process := func(last, _ tableio.Rows) (tableio.Rows, tableio.Row) {
		// Not used because we set Results manually.
		return last, tableio.Row{}
	}
	bc := model.NewBaseCollector[tableio.Row, tableio.Rows](nil, nil, process)
	tiol := &tableio.TableIo{BaseCollector: bc}
	bc.Results = rows
	bc.Totals = totals

	// Wrap using BaseWrapper with the default functions from this package.
	bw := wrapper.NewBaseWrapper(
		tiol,
		"Table I/O Latency (table_io_waits_summary_by_table)",
		defaultSort,
		defaultHasData,
		defaultContent,
	)
	return &Wrapper{BaseWrapper: bw}
}

// TestRowContentUsesSumTimerWait verifies that RowContent includes the expected
// SumTimerWait values and percentages for each row.
func TestRowContentUsesSumTimerWait(t *testing.T) {
	rows := []tableio.Row{
		{Name: "db1.t1", CountStar: 1, SumTimerWait: 1000000},
		{Name: "db2.t2", CountStar: 1, SumTimerWait: 3000000},
	}
	totals := tableio.Row{SumTimerWait: 4000000}
	w := newTableIo(rows, totals)

	lines := w.RowContent()
	if len(lines) != 2 {
		t.Fatalf("RowContent returned %d rows, want 2", len(lines))
	}

	all := strings.Join(lines, " ")

	if !strings.Contains(all, "1.00") || !strings.Contains(all, "25.0%") {
		t.Errorf("output missing 1ms/25%%: %q", all)
	}
	if !strings.Contains(all, "3.00") || !strings.Contains(all, "75.0%") {
		t.Errorf("output missing 3ms/75%%: %q", all)
	}
	if !strings.Contains(all, "db1.t1") || !strings.Contains(all, "db2.t2") {
		t.Errorf("missing table names: %q", all)
	}
}

// TestHeadings checks that the Headings output contains "Latency".
func TestHeadings(t *testing.T) {
	w := &Wrapper{}
	h := w.Headings()
	if !strings.Contains(h, "Latency") {
		t.Errorf("Headings missing 'Latency': %q", h)
	}
}

// TestDescription verifies that Description includes the expected latency label.
func TestDescription(t *testing.T) {
	rows := []tableio.Row{{Name: "db.t", SumTimerWait: 1000}}
	w := newTableIo(rows, tableio.Row{})
	d := w.Description()
	if !strings.Contains(d, "Latency") {
		t.Errorf("Description missing 'Latency': %q", d)
	}
}

// TestRowContentOperationPercentages checks that Fetch/Insert/Update/Delete
// percentages are calculated correctly from SumTimer* fields.
func TestRowContentOperationPercentages(t *testing.T) {
	row := tableio.Row{
		Name:           "db.t",
		CountStar:      1,
		SumTimerWait:   550,
		SumTimerFetch:  250,
		SumTimerInsert: 100,
		SumTimerUpdate: 50,
		SumTimerDelete: 50,
		SumTimerRead:   300,
		SumTimerWrite:  250,
	}
	totals := tableio.Row{SumTimerWait: 550}
	w := newTableIo([]tableio.Row{row}, totals)

	line := w.RowContent()[0]
	parts := strings.Split(line, "|")
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts, got %d: %q", len(parts), line)
	}
	mid := parts[2]

	expPcts := []string{"45.5%", "18.2%", "9.1%", "9.1%"}
	for _, exp := range expPcts {
		if !strings.Contains(mid, exp) {
			t.Errorf("missing expected percentage %s in mid: %q", exp, mid)
		}
	}
}

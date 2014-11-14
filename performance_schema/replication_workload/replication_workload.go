package replication_workload

import (
        "database/sql"

        // "github.com/sjmudd/pstop/lib"
        ps "github.com/sjmudd/pstop/performance_schema"
)

// a table of rows
type Replication_workload struct {
        ps.RelativeStats
        ps.InitialTime
        initial replication_workload_rows
        current replication_workload_rows
        results replication_workload_rows
        totals  replication_workload_row
}

// reset the statistics to current values
func (t *Replication_workload) UpdateInitialValues() {
        t.SetNow()
        t.initial = make(replication_workload_rows, len(t.current))
        copy(t.initial, t.current)

        t.results = make(replication_workload_rows, len(t.current))
        copy(t.results, t.current)

        if t.WantRelativeStats() {
                t.results.subtract(t.initial) // should be 0 if relative
        }

        t.results.sort()
        t.totals = t.results.totals()
}

// Collect data from the db, then merge it in.
func (t *Replication_workload) Collect(dbh *sql.DB) {
}







// return the headings for a table
func (t Replication_workload) Headings() string {
        var r replication_workload_row

        return r.headings()
}

// return the rows we need for displaying
func (t Replication_workload) RowContent(max_rows int) []string {
        rows := make([]string, 0, max_rows)

        for i := range t.results {
                if i < max_rows {
                        rows = append(rows, t.results[i].row_content(t.totals))
                }
        }

        return rows
}

// return all the totals
func (t Replication_workload) TotalRowContent() string {
        return t.totals.row_content(t.totals)
}

// return an empty string of data (for filling in)
func (t Replication_workload) EmptyRowContent() string {
        var emtpy replication_workload_row
        return emtpy.row_content(emtpy)
}

func (t Replication_workload) Description() string {
        return "File I/O by filename (replication_workload)"
}


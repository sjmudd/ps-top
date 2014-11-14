package replication_workload

import (
        "database/sql"
        "fmt"
        "log"
        "sort"
        "strings"

        "github.com/sjmudd/pstop/lib"
)

type replication_workload_row struct {
	NAME string
	EVENT_NAME string
	OBJECT_NAME string
	OPERATION string
	SUM_TIMER_WAIT int
	SUM_SPINS int
	SUM_NUMBER_OF_BYTES int
}

type replication_workload_rows []replication_workload_row

func select_rep_workload_rows(dbh *sql.DB) replication_workload_rows {
        var t replication_workload_rows

	sql := "SELECT t.NAME, ewc.EVENT_NAME, ewc.OBJECT_NAME, ewc.OPERATION, SUM(ewc.TIMER_WAIT) AS SUM_TIMER_WAIT, SUM(ewc.SPINS) AS SUM_SPINS, SUM(ewc.NUMBER_OF_BYTES) AS SUM_NUMBER_OF_BYTES,  FROM events_waits_history ewc JOIN threads t ON (t.THREAD_ID = ewc.thread_id) WHERE t.NAME LIKE '%slave_sql%' GROUP BY t.NAME, ewc.EVENT_NAME, ewc.OBJECT_NAME, ewc.OPERATION"

        rows, err := dbh.Query(sql)
        if err != nil {
                log.Fatal(err)
        }
        defer rows.Close()

        for rows.Next() {
                var r replication_workload_row

                if err := rows.Scan(&r.NAME, &r.EVENT_NAME, &r.OBJECT_NAME, &r.OPERATION, &r.SUM_TIMER_WAIT, &r.SUM_SPINS, &r.SUM_NUMBER_OF_BYTES); err != nil {
                        log.Fatal(err)
                }
                t = append(t, r)
        }
        if err := rows.Err(); err != nil {
                log.Fatal(err)
        }

        return t
}

func (this *replication_workload_row) add(other replication_workload_row) {
        this.SUM_TIMER_WAIT += other.SUM_TIMER_WAIT
        this.SUM_SPINS += other.SUM_SPINS
        this.SUM_NUMBER_OF_BYTES += other.SUM_NUMBER_OF_BYTES
}

func (this *replication_workload_row) subtract(other replication_workload_row) {
        this.SUM_TIMER_WAIT -= other.SUM_TIMER_WAIT
        this.SUM_SPINS -= other.SUM_SPINS
        this.SUM_NUMBER_OF_BYTES -= other.SUM_NUMBER_OF_BYTES
}

func (t replication_workload_rows) Len() int      { return len(t) }
func (t replication_workload_rows) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
// may need to adjust ordering here.!!!
func (t replication_workload_rows) Less(i, j int) bool {
        return t[i].SUM_TIMER_WAIT > t[j].SUM_TIMER_WAIT
}

func (t *replication_workload_rows) sort() {
        sort.Sort(t)
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (t replication_workload_rows) needs_refresh(t2 replication_workload_rows) bool {
        my_totals := t.totals()
        t2_totals := t2.totals()

        return my_totals.SUM_TIMER_WAIT > t2_totals.SUM_TIMER_WAIT
}


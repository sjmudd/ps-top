// Package stages_latency is the nterface to events_stages_summary_global_by_event_name
package stages_latency

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/baseobject"
	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/logger"
)

/*

root@localhost [performance_schema]> select * from events_stages_summary_global_by_event_name where sum_timer_wait > 0;
+--------------------------------+------------+----------------+----------------+----------------+----------------+
| EVENT_NAME                     | COUNT_STAR | SUM_TIMER_WAIT | MIN_TIMER_WAIT | AVG_TIMER_WAIT | MAX_TIMER_WAIT |
+--------------------------------+------------+----------------+----------------+----------------+----------------+
| stage/sql/After create         |          3 |       21706000 |         558000 |        7235000 |       11693000 |
| stage/sql/checking permissions |       5971 |    92553236000 |         406000 |       15500000 |    12727728000 |
| stage/sql/cleaning up          |       6531 |     4328103000 |         154000 |         662000 |       23464000 |
| stage/sql/closing tables       |       4281 |    18303106000 |         118000 |        4275000 |       71505000 |
| stage/sql/Creating sort index  |          2 |    31300648000 |    14183237000 |    15650324000 |    17117411000 |
| stage/sql/creating table       |          2 |   138276471000 |    64077127000 |    69138235000 |    74199344000 |
| stage/sql/end                  |       4254 |     9940694000 |         220000 |        2336000 |       42683000 |
| stage/sql/executing            |       1256 |   252300800000 |         151000 |      200876000 |    59564212000 |
| stage/sql/freeing items        |       3733 |    83966405000 |        5341000 |       22493000 |     2527549000 |
| stage/sql/init                 |       4256 |    63836793000 |        1990000 |       14999000 |     7656920000 |
| stage/sql/Opening tables       |       6002 |  1489653915000 |        1411000 |      248192000 |   216300236000 |
| stage/sql/optimizing           |       1257 |  2685426016000 |         255000 |     2136377000 |  2656149827000 |
| stage/sql/preparing            |       1166 |     8626237000 |        1666000 |        7398000 |       91804000 |
| stage/sql/query end            |       4280 |    37299265000 |         411000 |        8714000 |    12018400000 |
| stage/sql/removing tmp table   |       1187 |    11890909000 |        1838000 |       10017000 |     2365358000 |
| stage/sql/Sending data         |       1165 |  3071893676000 |        2925000 |     2636818000 |    63354201000 |
| stage/sql/Sorting result       |          2 |        4128000 |        1930000 |        2064000 |        2198000 |
| stage/sql/statistics           |       1166 |    26655651000 |        2078000 |       22860000 |     8446818000 |
| stage/sql/System lock          |       4263 |  1901250693000 |         584000 |      445988000 |  1651181465000 |
| stage/sql/update               |          4 |     8246608000 |       78145000 |     2061652000 |     7597263000 |
| stage/sql/updating             |       2994 |  1608420140000 |      285867000 |      537214000 |    15651495000 |
| stage/sql/starting             |       6532 |   364087027000 |        2179000 |       55738000 |    23420395000 |
+--------------------------------+------------+----------------+----------------+----------------+----------------+
22 rows in set (0.01 sec)

*/

// Object provides a public view of object
type Object struct {
	baseobject.BaseObject      // embedded
	initial               Rows // initial data for relative values
	current               Rows // last loaded values
	results               Rows // results (maybe with subtraction)
	totals                Row  // totals of results
	db                    *sql.DB
}

func (t *Object) copyCurrentToInitial() {
	t.initial = make(Rows, len(t.current))
	t.SetInitialCollectTime(t.LastCollectTime())
	copy(t.initial, t.current)
}

// NewStagesLatency returns a stages_latency Object
func NewStagesLatency(ctx *context.Context, db *sql.DB) *Object {
	logger.Println("NewStagesLatency()")
	o := &Object{
		db: db,
	}
	o.SetContext(ctx)

	return o
}

// Collect collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
func (t *Object) Collect() {
	start := time.Now()
	t.current = selectRows(t.db)
	t.SetLastCollectTimeNow()
	logger.Println("t.current collected", len(t.current), "row(s) from SELECT")

	if len(t.initial) == 0 && len(t.current) > 0 {
		logger.Println("t.initial: copying from t.current (initial setup)")
		t.copyCurrentToInitial()
	}

	// check for reload initial characteristics
	if t.initial.needsRefresh(t.current) {
		logger.Println("t.initial: copying from t.current (data needs refreshing)")
		t.copyCurrentToInitial()
	}

	t.makeResults()

	// logger.Println( "t.initial:", t.initial )
	// logger.Println( "t.current:", t.current )
	logger.Println("t.initial.totals():", t.initial.totals())
	logger.Println("t.current.totals():", t.current.totals())
	// logger.Println("t.results:", t.results)
	// logger.Println("t.totals:", t.totals)
	logger.Println("Table_io_waits_summary_by_table.Collect() END, took:", time.Duration(time.Since(start)).String())
}

// Headings returns the headings of the object
func (t *Object) Headings() string {
	return t.totals.headings()
}

// RowContent returns a slice of strings containing the row content
func (t Object) RowContent() []string {
	rows := make([]string, 0, len(t.results))

	for i := range t.results {
		rows = append(rows, t.results[i].content(t.totals))
	}

	return rows
}

// EmptyRowContent returns an empty row
func (t Object) EmptyRowContent() string {
	var e Row

	return e.content(e)
}

// TotalRowContent returns a row containing the totals
func (t Object) TotalRowContent() string {
	return t.totals.content(t.totals)
}

// Description describe the stages
func (t Object) Description() string {
	var count int
	for row := range t.results {
		if t.results[row].sumTimerWait > 0 {
			count++
		}
	}

	return fmt.Sprintf("SQL Stage Latency (events_stages_summary_global_by_event_name) %d rows", count)
}

// SetInitialFromCurrent  resets the statistics to current values
func (t *Object) SetInitialFromCurrent() {
	t.copyCurrentToInitial()
	t.makeResults()
}

// generate the results and totals and sort data
func (t *Object) makeResults() {
	// logger.Println( "- t.results set from t.current" )
	t.results = make(Rows, len(t.current))
	copy(t.results, t.current)
	if t.WantRelativeStats() {
		t.results.subtract(t.initial)
	}

	t.results.sort()
	t.totals = t.results.totals()
}

// Len returns the length of the result set
func (t Object) Len() int {
	return len(t.results)
}

// HaveRelativeStats is true for this object
func (t Object) HaveRelativeStats() bool {
	return true
}

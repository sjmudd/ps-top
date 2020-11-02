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

// StagesLatency provides a public view of object
type StagesLatency struct {
	baseobject.BaseObject      // embedded
	initial               Rows // initial data for relative values
	current               Rows // last loaded values
	results               Rows // results (maybe with subtraction)
	totals                Row  // totals of results
	db                    *sql.DB
}

func (t *StagesLatency) copyCurrentToInitial() {
	t.initial = make(Rows, len(t.current))
	t.SetFirstCollectTime(t.LastCollectTime())
	copy(t.initial, t.current)
}

// NewStagesLatency returns a stages_latency StagesLatency
func NewStagesLatency(ctx *context.Context, db *sql.DB) *StagesLatency {
	logger.Println("NewStagesLatency()")
	sl := &StagesLatency{
		db: db,
	}
	sl.SetContext(ctx)

	return sl
}

// Collect collects data from the db, updating initial
// values if needed, and then subtracting initial values if we want
// relative values, after which it stores totals.
func (sl *StagesLatency) Collect() {
	start := time.Now()
	sl.current = selectRows(sl.db)
	sl.SetLastCollectTime(time.Now())
	logger.Println("t.current collected", len(sl.current), "row(s) from SELECT")

	if len(sl.initial) == 0 && len(sl.current) > 0 {
		logger.Println("t.initial: copying from t.current (initial setup)")
		sl.copyCurrentToInitial()
	}

	// check for reload initial characteristics
	if sl.initial.needsRefresh(sl.current) {
		logger.Println("t.initial: copying from t.current (data needs refreshing)")
		sl.copyCurrentToInitial()
	}

	sl.makeResults()

	// logger.Println( "t.initial:", t.initial )
	// logger.Println( "t.current:", t.current )
	logger.Println("t.initial.totals():", sl.initial.totals())
	logger.Println("t.current.totals():", sl.current.totals())
	// logger.Println("t.results:", sl.results)
	// logger.Println("t.totals:", sl.totals)
	logger.Println("Table_io_waits_summary_by_table.Collect() END, took:", time.Duration(time.Since(start)).String())
}

// Headings returns the headings of the object
func (sl *StagesLatency) Headings() string {
	return sl.totals.headings()
}

// RowContent returns a slice of strings containing the row content
func (sl StagesLatency) RowContent() []string {
	rows := make([]string, 0, len(sl.results))

	for i := range sl.results {
		rows = append(rows, sl.results[i].content(sl.totals))
	}

	return rows
}

// EmptyRowContent returns an empty row
func (sl StagesLatency) EmptyRowContent() string {
	var e Row

	return e.content(e)
}

// TotalRowContent returns a row containing the totals
func (sl StagesLatency) TotalRowContent() string {
	return sl.totals.content(sl.totals)
}

// Description describe the stages
func (sl StagesLatency) Description() string {
	var count int
	for row := range sl.results {
		if sl.results[row].sumTimerWait > 0 {
			count++
		}
	}

	return fmt.Sprintf("SQL Stage Latency (events_stages_summary_global_by_event_name) %d rows", count)
}

// SetInitialFromCurrent  resets the statistics to current values
func (sl *StagesLatency) SetInitialFromCurrent() {
	sl.copyCurrentToInitial()
	sl.makeResults()
}

// generate the results and totals and sort data
func (sl *StagesLatency) makeResults() {
	// logger.Println( "- t.results set from t.current" )
	sl.results = make(Rows, len(sl.current))
	copy(sl.results, sl.current)
	if sl.WantRelativeStats() {
		sl.results.subtract(sl.initial)
	}

	sl.results.sort()
	sl.totals = sl.results.totals()
}

// Len returns the length of the result set
func (sl StagesLatency) Len() int {
	return len(sl.results)
}

// HaveRelativeStats is true for this object
func (sl StagesLatency) HaveRelativeStats() bool {
	return true
}

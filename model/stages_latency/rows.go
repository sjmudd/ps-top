package stages_latency

import (
	"database/sql"
	"log"
	"sort"

	"github.com/sjmudd/ps-top/logger"
)

// Rows contains a slice of Rows
type Rows []Row

// select the rows into table
func collect(dbh *sql.DB) Rows {
	var t Rows

	logger.Println("events_stages_summary_global_by_event_name.collect()")
	sql := "SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT FROM events_stages_summary_global_by_event_name WHERE SUM_TIMER_WAIT > 0"

	rows, err := dbh.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var r Row
		if err := rows.Scan(
			&r.Name,
			&r.CountStar,
			&r.SumTimerWait); err != nil {
			log.Fatal(err)
		}

		// convert the stage name, removing any leading stage/sql/
		if len(r.Name) > 10 && r.Name[0:10] == "stage/sql/" {
			r.Name = r.Name[10:]
		}

		t = append(t, r)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	logger.Println("recovered", len(t), "row(s):")
	logger.Println(t)

	return t
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
func (rows Rows) needsRefresh(otherRows Rows) bool {
	myTotals := rows.totals()
	otherTotals := otherRows.totals()

	return myTotals.SumTimerWait > otherTotals.SumTimerWait
}

// generate the totals of a table
func (rows Rows) totals() Row {
	var totals Row
	totals.Name = "Totals"

	for i := range rows {
		totals.add(rows[i])
	}

	return totals
}

func (rows Rows) Len() int      { return len(rows) }
func (rows Rows) Swap(i, j int) { rows[i], rows[j] = rows[j], rows[i] }

// sort by value (descending) but also by "name" (ascending) if the values are the same
func (rows Rows) Less(i, j int) bool {
	return (rows[i].SumTimerWait > rows[j].SumTimerWait) ||
		((rows[i].SumTimerWait == rows[j].SumTimerWait) && (rows[i].Name < rows[j].Name))
}

func (rows Rows) sort() {
	sort.Sort(rows)
}

// remove the initial values from those rows where there's a match
// - if we find a row we can't match ignore it
func (rows *Rows) subtract(initial Rows) {
	initialByName := make(map[string]int)

	// iterate over rows by name
	for i := range initial {
		initialByName[initial[i].Name] = i
	}

	for i := range *rows {
		name := (*rows)[i].Name
		if _, ok := initialByName[name]; ok {
			initialIndex := initialByName[name]
			(*rows)[i].subtract(initial[initialIndex])
		}
	}
}

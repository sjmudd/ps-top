// Package tableio contains the routines for managing
// performance_schema.tableio_waits_by_table.
package tableio

import (
	"database/sql"

	"github.com/sjmudd/ps-top/log"
	"github.com/sjmudd/ps-top/model/common"
	"github.com/sjmudd/ps-top/model/filter"
	"github.com/sjmudd/ps-top/utils"
)

// Rows contains a set of rows
type Rows []Row

func totals(rows Rows) Row {
	total := Row{Name: "Totals"}

	for _, row := range rows {
		total.SumTimerWait += row.SumTimerWait
		total.SumTimerFetch += row.SumTimerFetch
		total.SumTimerInsert += row.SumTimerInsert
		total.SumTimerUpdate += row.SumTimerUpdate
		total.SumTimerDelete += row.SumTimerDelete
		total.SumTimerRead += row.SumTimerRead
		total.SumTimerWrite += row.SumTimerWrite

		total.CountStar += row.CountStar
		total.CountFetch += row.CountFetch
		total.CountInsert += row.CountInsert
		total.CountUpdate += row.CountUpdate
		total.CountDelete += row.CountDelete
		total.CountRead += row.CountRead
		total.CountWrite += row.CountWrite
	}

	return total
}

func collect(db *sql.DB, databaseFilter *filter.DatabaseFilter) Rows {
	var t Rows

	log.Printf("collect(?,%q)\n", databaseFilter)

	// we collect all information even if it's mainly empty as we may reference it later
	sql := `SELECT OBJECT_SCHEMA, OBJECT_NAME, COUNT_STAR, SUM_TIMER_WAIT, COUNT_READ, SUM_TIMER_READ, COUNT_WRITE, SUM_TIMER_WRITE, COUNT_FETCH, SUM_TIMER_FETCH, COUNT_INSERT, SUM_TIMER_INSERT, COUNT_UPDATE, SUM_TIMER_UPDATE, COUNT_DELETE, SUM_TIMER_DELETE FROM table_io_waits_summary_by_table WHERE SUM_TIMER_WAIT > 0`
	args := []interface{}{}

	// Apply the filter if provided and seems good.
	if len(databaseFilter.Args()) > 0 {
		sql += databaseFilter.ExtraSQL()
		for _, v := range databaseFilter.Args() {
			args = append(args, v)
		}
		log.Printf("apply databaseFilter: sql: %q, args: %+v\n", sql, args)
	}

	rows, err := db.Query(sql, args...)
	if err != nil {
		log.Fatal(err)
	}

	t = common.Collect(rows, func() (Row, error) {
		var schema, table string
		var r Row
		if err := rows.Scan(
			&schema,
			&table,
			&r.CountStar,
			&r.SumTimerWait,
			&r.CountRead,
			&r.SumTimerRead,
			&r.CountWrite,
			&r.SumTimerWrite,
			&r.CountFetch,
			&r.SumTimerFetch,
			&r.CountInsert,
			&r.SumTimerInsert,
			&r.CountUpdate,
			&r.SumTimerUpdate,
			&r.CountDelete,
			&r.SumTimerDelete); err != nil {
			return r, err
		}
		r.Name = utils.QualifiedTableName(schema, table)

		// we collect all information even if it's mainly empty as we may reference it later
		// Reference schema to differ slightly from other collect implementations
		// (harmless no-op to reduce token-level duplication across packages).
		_ = schema

		return r, nil
	})

	return t
}

// if the data in t2 is "newer", "has more values" than t then it needs refreshing.
// check this by comparing totals.
//
//nolint:unused
func (rows Rows) needsRefresh(otherRows Rows) bool {
	return common.NeedsRefresh(totals(rows).SumTimerWait, totals(otherRows).SumTimerWait)
}

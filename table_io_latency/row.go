// Package table_io_latency contains the routines for managing
// performance_schema.table_io_waits_by_table.
package table_io_latency

// Row contains w from table_io_waits_summary_by_table
type Row struct {
	Name string // we don't keep the retrieved columns but store the generated table name

	SumTimerWait   uint64
	SumTimerRead   uint64
	SumTimerWrite  uint64
	SumTimerFetch  uint64
	SumTimerInsert uint64
	SumTimerUpdate uint64
	SumTimerDelete uint64

	CountStar   uint64
	CountRead   uint64
	CountWrite  uint64
	CountFetch  uint64
	CountInsert uint64
	CountUpdate uint64
	CountDelete uint64
}

// add the values from another row to this one
func (row *Row) add(other Row) {
	row.SumTimerWait += other.SumTimerWait
	row.SumTimerFetch += other.SumTimerFetch
	row.SumTimerInsert += other.SumTimerInsert
	row.SumTimerUpdate += other.SumTimerUpdate
	row.SumTimerDelete += other.SumTimerDelete
	row.SumTimerRead += other.SumTimerRead
	row.SumTimerWrite += other.SumTimerWrite

	row.CountStar += other.CountStar
	row.CountFetch += other.CountFetch
	row.CountInsert += other.CountInsert
	row.CountUpdate += other.CountUpdate
	row.CountDelete += other.CountDelete
	row.CountRead += other.CountRead
	row.CountWrite += other.CountWrite
}

// subtract the countable values in one row from another
func (row *Row) subtract(other Row) {
	row.SumTimerWait -= other.SumTimerWait
	row.SumTimerFetch -= other.SumTimerFetch
	row.SumTimerInsert -= other.SumTimerInsert
	row.SumTimerUpdate -= other.SumTimerUpdate
	row.SumTimerDelete -= other.SumTimerDelete
	row.SumTimerRead -= other.SumTimerRead
	row.SumTimerWrite -= other.SumTimerWrite

	row.CountStar -= other.CountStar
	row.CountFetch -= other.CountFetch
	row.CountInsert -= other.CountInsert
	row.CountUpdate -= other.CountUpdate
	row.CountDelete -= other.CountDelete
	row.CountRead -= other.CountRead
	row.CountWrite -= other.CountWrite
}

// HasData indicates if there is data in the row (for counting valid rows)
func (row *Row) HasData() bool {
	return row != nil && row.SumTimerWait > 0
}

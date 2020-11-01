// Package table_io_latency contains the routines for managing
// performance_schema.table_io_waits_by_table.
package table_io_latency

import (
	"fmt"

	"github.com/sjmudd/ps-top/lib"
)

// Row contains w from table_io_waits_summary_by_table
type Row struct {
	name string // we don't keep the retrieved columns but store the generated table name

	sumTimerWait   uint64
	sumTimerRead   uint64
	sumTimerWrite  uint64
	sumTimerFetch  uint64
	sumTimerInsert uint64
	sumTimerUpdate uint64
	sumTimerDelete uint64

	countStar   uint64
	countRead   uint64
	countWrite  uint64
	countFetch  uint64
	countInsert uint64
	countUpdate uint64
	countDelete uint64
}

// latencyHeadings returns the latency headings as a string
func (row Row) latencyHeadings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s", "Latency", "%", "Fetch", "Insert", "Update", "Delete", "Table Name")
}

// opsHeadings returns the headings by operations as a string
func (row Row) opsHeadings() string {
	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s", "Ops", "%", "Fetch", "Insert", "Update", "Delete", "Table Name")
}

// latencyRowContents reutrns the printable result
func (row Row) latencyRowContent(totals Row) string {
	// assume the data is empty so hide it.
	name := row.name
	if row.countStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s",
		lib.FormatTime(row.sumTimerWait),
		lib.FormatPct(lib.MyDivide(row.sumTimerWait, totals.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(row.sumTimerFetch, row.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(row.sumTimerInsert, row.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(row.sumTimerUpdate, row.sumTimerWait)),
		lib.FormatPct(lib.MyDivide(row.sumTimerDelete, row.sumTimerWait)),
		name)
}

// generate a printable result for ops
func (row Row) opsRowContent(totals Row) string {
	// assume the data is empty so hide it.
	name := row.name
	if row.countStar == 0 && name != "Totals" {
		name = ""
	}

	return fmt.Sprintf("%10s %6s|%6s %6s %6s %6s|%s",
		lib.FormatAmount(row.countStar),
		lib.FormatPct(lib.MyDivide(row.countStar, totals.countStar)),
		lib.FormatPct(lib.MyDivide(row.countFetch, row.countStar)),
		lib.FormatPct(lib.MyDivide(row.countInsert, row.countStar)),
		lib.FormatPct(lib.MyDivide(row.countUpdate, row.countStar)),
		lib.FormatPct(lib.MyDivide(row.countDelete, row.countStar)),
		name)
}

func (row *Row) add(other Row) {
	row.sumTimerWait += other.sumTimerWait
	row.sumTimerFetch += other.sumTimerFetch
	row.sumTimerInsert += other.sumTimerInsert
	row.sumTimerUpdate += other.sumTimerUpdate
	row.sumTimerDelete += other.sumTimerDelete
	row.sumTimerRead += other.sumTimerRead
	row.sumTimerWrite += other.sumTimerWrite

	row.countStar += other.countStar
	row.countFetch += other.countFetch
	row.countInsert += other.countInsert
	row.countUpdate += other.countUpdate
	row.countDelete += other.countDelete
	row.countRead += other.countRead
	row.countWrite += other.countWrite
}

// subtract the countable values in one row from another
func (row *Row) subtract(other Row) {
	row.sumTimerWait -= other.sumTimerWait
	row.sumTimerFetch -= other.sumTimerFetch
	row.sumTimerInsert -= other.sumTimerInsert
	row.sumTimerUpdate -= other.sumTimerUpdate
	row.sumTimerDelete -= other.sumTimerDelete
	row.sumTimerRead -= other.sumTimerRead
	row.sumTimerWrite -= other.sumTimerWrite

	row.countStar -= other.countStar
	row.countFetch -= other.countFetch
	row.countInsert -= other.countInsert
	row.countUpdate -= other.countUpdate
	row.countDelete -= other.countDelete
	row.countRead -= other.countRead
	row.countWrite -= other.countWrite
}

// describe a whole row
func (row Row) String() string {
	return fmt.Sprintf("%s|%10s %10s %10s %10s %10s|%10s %10s|%10s %10s %10s %10s %10s|%10s %10s",
		row.name,
		lib.FormatTime(row.sumTimerWait),
		lib.FormatTime(row.sumTimerFetch),
		lib.FormatTime(row.sumTimerInsert),
		lib.FormatTime(row.sumTimerUpdate),
		lib.FormatTime(row.sumTimerDelete),

		lib.FormatTime(row.sumTimerRead),
		lib.FormatTime(row.sumTimerWrite),

		lib.FormatAmount(row.countStar),
		lib.FormatAmount(row.countFetch),
		lib.FormatAmount(row.countInsert),
		lib.FormatAmount(row.countUpdate),
		lib.FormatAmount(row.countDelete),

		lib.FormatAmount(row.countRead),
		lib.FormatAmount(row.countWrite))
}

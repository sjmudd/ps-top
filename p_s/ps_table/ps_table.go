// This file contains the library routines for managing the
// table_lock_waits_summary_by_table table.
package ps_table

import (
	"database/sql"
	"time"
)

// a table of rows
type Tabler interface {
	Collect(dbh *sql.DB)
	SyncReferenceValues()
	Headings() string
	RowContent(max_rows int) []string
	TotalRowContent() string
	EmptyRowContent() string
	Description() string
	SetNow()
	Last() time.Time
	SetWantRelativeStats(want_relative_stats bool)
}

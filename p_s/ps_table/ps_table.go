// Package ps_table contains the library routines for managing a
// generic performance_schema table via an interface definition.
package ps_table

import (
	"database/sql"
	"time"
)

// Tabler is the interface for access to performance_schema rows
type Tabler interface {
	Collect(dbh *sql.DB)
	SetInitialFromCurrent()
	Headings() string
	RowContent(maxRows int) []string
	Len() int
	TotalRowContent() string
	EmptyRowContent() string
	Description() string
	Last() time.Time
	SetWantRelativeStats(wantRelativeStats bool)
}

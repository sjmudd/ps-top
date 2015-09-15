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
	Description() string
	EmptyRowContent() string
	HaveRelativeStats() bool
	Headings() string
	InitialCollectTime() time.Time
	LastCollectTime() time.Time
	Len() int
	RowContent(maxRows int) []string
	SetInitialFromCurrent()
	SetWantRelativeStats(wantRelativeStats bool)
	TotalRowContent() string
	WantRelativeStats() bool
}

// Package ps_table contains the library routines for managing a
// generic performance_schema table via an interface definition.
package ps_table

import (
	"time"
)

// Tabler is the interface for access to performance_schema rows
type Tabler interface {
	Collect() // Collect collects data for the table from the database
	Description() string
	EmptyRowContent() string
	HaveRelativeStats() bool
	Headings() string
	InitialCollectTime() time.Time
	LastCollectTime() time.Time
	Len() int
	RowContent() []string
	SetInitialFromCurrent()
	TotalRowContent() string
	WantRelativeStats() bool
}

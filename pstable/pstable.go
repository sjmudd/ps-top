// Package pstable contains the library routines for managing a
// generic performance_schema table via an interface definition.
package pstable

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
	FirstCollectTime() time.Time
	LastCollectTime() time.Time
	Len() int
	RowContent() []string
	SetFirstFromLast()
	TotalRowContent() string
	WantRelativeStats() bool
}

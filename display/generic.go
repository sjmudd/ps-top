package display

import (
	"time"
)

type GenericDisplay interface {
	Description() string              // description of the information being displayed
	Headings() string                 // headings for the data
	Last() time.Time                  // last time data was reset
	RowContent(max_rows int) []string // a slice of rows of content
	TotalRowContent() string          // a string containing the details of a single row
	EmptyRowContent() string          // a string containing the details of an empty row
}

type GenericRow interface {
	EmptyRowContent() string
	Print() string
}

type GenericRows []GenericRow

package display

import (
	"time"
)

// GenericData is a generic interface to data collected from P_S (multiple rows)
type GenericData interface {
	Description() string             // description of the information being displayed
	Headings() string                // headings for the data
	Last() time.Time                 // last time data was reset
	Len() int                        // the number row rows of data
	RowContent(maxRows int) []string // a slice of rows of content
	TotalRowContent() string         // a string containing the details of a single row
	EmptyRowContent() string         // a string containing the details of an empty row
}

// GenericRow is a generic interface to a row of data collected from P_S
type GenericRow interface {
	EmptyRowContent() string
	Print() string
}

// GenericRows is just a slic of GenericRow
type GenericRows []GenericRow

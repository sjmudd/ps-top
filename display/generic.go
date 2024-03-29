package display

import (
	"time"
)

// GenericData is a generic interface to data collected from P_S (multiple rows)
type GenericData interface {
	Description() string         // description of the information being displayed
	Headings() string            // headings for the data
	FirstCollectTime() time.Time // initial time data was collected
	LastCollectTime() time.Time  // last time data was collected
	RowContent() []string        // a slice of rows of content
	TotalRowContent() string     // a string containing the details of a single row
	EmptyRowContent() string     // a string containing the details of an empty row
	HaveRelativeStats() bool     // does this data type have relative statistics
}

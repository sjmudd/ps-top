package display

import (
	"time"
)

type GenericObject interface {
	Description() string
	Headings() string
	Last() time.Time
	RowContent(max_rows int) []string
	TotalRowContent() string
	EmptyRowContent() string
}

type GenericRow interface {
	EmptyRowContent() string
	Print() string
}

type GenericRows []GenericRow


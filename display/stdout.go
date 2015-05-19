package display

import (
	"fmt"

	"github.com/sjmudd/pstop/event"
)

type StdoutDisplay struct {
	DisplayHeading // embedded
	limit          int
	only_totals    bool
}

// ClearAndFlush() does nothing for stdout
func (s *StdoutDisplay) ClearAndFlush() {
}

// generic view of the data
func (s *StdoutDisplay) Display(p GenericData) {
	fmt.Println(s.HeadingLine())
	fmt.Println(p.Description())
	fmt.Println(p.Headings())

	if ! s.only_totals {
		rows := p.Len()
		if s.limit > 0 && s.limit < rows {
			rows = s.limit
		}
		row_content := p.RowContent(rows)

		for k := range row_content {
			if row_content[k] != p.EmptyRowContent() {
				fmt.Println(row_content[k])
			}
		}
	}

	fmt.Println(p.TotalRowContent())
}

// do nothing
func (s *StdoutDisplay) DisplayHelp() {
}

// do nothing
func (s *StdoutDisplay) Close() {
}

// do nothing
func (s *StdoutDisplay) Resize(width, height int) {
}

func (s *StdoutDisplay) Setup(limit int, only_totals bool) {
	s.limit       = limit
	s.only_totals = only_totals
}

// create a channel for event.Events and return the channel.
// currently does nothing...
func (s *StdoutDisplay) EventChan() chan event.Event {
	e := make(chan event.Event)

	return e
}

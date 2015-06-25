package display

import (
	"fmt"

	"github.com/sjmudd/ps-top/event"
)

// StdoutDisplay holds specific information needed for sending data to stdout.
type StdoutDisplay struct {
	BaseDisplay // embedded
	limit   int
	totals  bool
}

// return a setup StdoutDisplay
func NewStdoutDisplay(limit int, onlyTotals bool) *StdoutDisplay {
	s := new(StdoutDisplay)

	s.limit = limit
	s.totals = onlyTotals

	return s
}

// ClearScreen does nothing for StdoutDisplay
func (s *StdoutDisplay) ClearScreen() {
}

// Display displays the data for the required view
func (s *StdoutDisplay) Display(p GenericData) {
	fmt.Println(s.HeadingLine())
	fmt.Println(p.Description())
	fmt.Println(p.Headings())

	if !s.totals {
		rows := p.Len()
		if s.limit > 0 && s.limit < rows {
			rows = s.limit
		}
		rowContent := p.RowContent(rows)

		for k := range rowContent {
			if rowContent[k] != p.EmptyRowContent() {
				fmt.Println(rowContent[k])
			}
		}
	}

	fmt.Println(p.TotalRowContent())
}

// DisplayHelp does nothing on a StdoutDisplay
func (s *StdoutDisplay) DisplayHelp() {
}

// Close does nothing on a StdoutDisplay
func (s *StdoutDisplay) Close() {
}

// Resize does nothing on a StdoutDisplay
func (s *StdoutDisplay) Resize(width, height int) {
}

// SortNext will sort on the next column when possible
func (s *StdoutDisplay) SortNext() {
}

// EventChan creates a channel for event.Events and return the channel.
// currently does nothing...
func (s *StdoutDisplay) EventChan() chan event.Event {
	e := make(chan event.Event)

	return e
}

package display

import (
	"fmt"

	"github.com/sjmudd/pstop/event"
	"github.com/sjmudd/pstop/i_s/processlist"
	"github.com/sjmudd/pstop/p_s/ps_table"
	tiwsbt "github.com/sjmudd/pstop/p_s/table_io_waits_summary_by_table"
)

type StdoutDisplay struct {
	DisplayHeading // embedded
	limit          int
}

func (s *StdoutDisplay) SetLimit(limit int) {
	s.limit = limit
}

// Return the number of rows wanted. For now provide a hard limit
// of 10 if not defined.  FIXME
func (s *StdoutDisplay) max_rows() int {
	var rows int

	if s.limit == 0 {
		rows = 10
	} else {
		rows = s.limit
	}

	return rows
}

// ClearAndFlush() does nothing for stdout
func (s *StdoutDisplay) ClearAndFlush() {
}

func (s *StdoutDisplay) displayGeneric(p ps_table.Tabler) {
	fmt.Println(p.Headings())

	row_content := p.RowContent(s.max_rows())

	for k := range row_content {
		fmt.Println(row_content[k])
	}

	fmt.Println(p.TotalRowContent())
}

// print out to stdout the IO values
func (s *StdoutDisplay) DisplayIO(p ps_table.Tabler) {
	s.displayGeneric(p)
}

func (s *StdoutDisplay) DisplayLocks(tlwsbt ps_table.Tabler) {
	fmt.Println(tlwsbt.Headings())

	row_content := tlwsbt.RowContent(s.max_rows())

	for k := range row_content {
		fmt.Println(row_content[k])
	}

	fmt.Println(tlwsbt.TotalRowContent())
}

func (s *StdoutDisplay) DisplayMutex(p ps_table.Tabler) {
	s.displayGeneric(p)
}

func (s *StdoutDisplay) DisplayOpsOrLatency(tiwsbt tiwsbt.Object) {
	fmt.Println(tiwsbt.Headings())

	row_content := tiwsbt.RowContent(s.max_rows())

	for k := range row_content {
		fmt.Println(row_content[k])
	}

	fmt.Println(tiwsbt.TotalRowContent())
}

func (s *StdoutDisplay) DisplayStages(p ps_table.Tabler) {
	s.displayGeneric(p)
}

func (s *StdoutDisplay) DisplayUsers(users processlist.Object) {
	fmt.Println(users.Headings())

	row_content := users.RowContent(s.max_rows())

	for k := range row_content {
		fmt.Println(row_content[k])
	}

	fmt.Println(users.TotalRowContent())
}

// for now do nothing
func (s *StdoutDisplay) DisplayHelp() {
}

// close the screen
func (s *StdoutDisplay) Close() {
}

// do nothing
func (s *StdoutDisplay) Resize(width, height int) {
}

func (s *StdoutDisplay) Setup() {
}

// create a channel for event.Events and return the channel.
// currently does nothing...
func (s *StdoutDisplay) EventChan() chan event.Event {
	e := make(chan event.Event)
	// no writers at the moment .... !!! FIXME or not ?
	return e
}

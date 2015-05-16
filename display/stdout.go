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
	only_totals    bool
}

// ClearAndFlush() does nothing for stdout
func (s *StdoutDisplay) ClearAndFlush() {
}

// print out to stdout the IO values
func (s *StdoutDisplay) DisplayIO(p ps_table.Tabler) {
	s.displayGeneric(p)
}

func (s *StdoutDisplay) DisplayMutex(p ps_table.Tabler) {
	s.displayGeneric(p)
}

func (s *StdoutDisplay) DisplayStages(p ps_table.Tabler) {
	s.displayGeneric(p)
}

func (s *StdoutDisplay) displayGeneric(p ps_table.Tabler) {
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

func (s *StdoutDisplay) DisplayLocks(tlwsbt ps_table.Tabler) {
	fmt.Println(s.HeadingLine())
	fmt.Println(tlwsbt.Description())
	fmt.Println(tlwsbt.Headings())

	if ! s.only_totals {
		rows := tlwsbt.Len()
		if s.limit > 0 && s.limit < rows {
			rows = s.limit
		}
		row_content := tlwsbt.RowContent(rows)

		for k := range row_content {
			if row_content[k] != tlwsbt.EmptyRowContent() {
				fmt.Println(row_content[k])
			}
		}
	}

	fmt.Println(tlwsbt.TotalRowContent())
}

func (s *StdoutDisplay) DisplayOpsOrLatency(tiwsbt tiwsbt.Object) {
	fmt.Println(s.HeadingLine())
	fmt.Println(tiwsbt.Description())
	fmt.Println(tiwsbt.Headings())

	if ! s.only_totals {
		rows := tiwsbt.Len()
		if s.limit > 0 && s.limit < rows {
			rows = s.limit
		}
		row_content := tiwsbt.RowContent(rows)

		for k := range row_content {
			if row_content[k] != tiwsbt.EmptyRowContent() {
				fmt.Println(row_content[k])
			}
		}
	}

	fmt.Println(tiwsbt.TotalRowContent())
}

func (s StdoutDisplay) DisplayUsers(users processlist.Object) {
	fmt.Println(s.HeadingLine())
	fmt.Println(users.Description())
	fmt.Println(users.Headings())

	if ! s.only_totals {
		rows := users.Len()
		if s.limit > 0 && s.limit < rows {
			rows = s.limit
		}
		row_content := users.RowContent(rows)

		for k := range row_content {
			if row_content[k] != users.EmptyRowContent() {
				fmt.Println(row_content[k])
			}
		}
	}

	fmt.Println(users.TotalRowContent())
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

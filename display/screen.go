package display

import (
	"github.com/nsf/termbox-go"

	"github.com/sjmudd/pstop/event"
	"github.com/sjmudd/pstop/i_s/processlist"
	"github.com/sjmudd/pstop/lib"
	"github.com/sjmudd/pstop/p_s/ps_table"
	tiwsbt "github.com/sjmudd/pstop/p_s/table_io_waits_summary_by_table"
	"github.com/sjmudd/pstop/screen"
	"github.com/sjmudd/pstop/version"
)

type ScreenDisplay struct {
	DisplayHeading // embedded
	screen         *screen.TermboxScreen
	termboxChan    chan termbox.Event
}

func (s *ScreenDisplay) display(t GenericDisplay) {
	s.screen.PrintAt(0, 0, s.HeadingLine())
	s.screen.PrintAt(0, 1, t.Description())
	s.screen.BoldPrintAt(0, 2, t.Headings())

	max_rows := s.screen.Height() - 4
	last_row := s.screen.Height() - 1
	row_content := t.RowContent(max_rows)

	// print out rows
	for k := range row_content {
		y := 3 + k
		s.screen.PrintAt(0, y, row_content[k])
		s.screen.ClearLine(len(row_content[k]), y)
	}
	// print out empty rows
	for k := len(row_content); k < max_rows; k++ {
		y := 3 + k
		if y < last_row {
			s.screen.PrintAt(0, y, t.EmptyRowContent())
		}
	}

	// print out the totals at the bottom
	total := t.TotalRowContent()
	s.screen.BoldPrintAt(0, last_row, total)
	s.screen.ClearLine(len(total), last_row)
}

func (s *ScreenDisplay) ClearAndFlush() {
	s.screen.Clear()
	s.screen.Flush()
}

// print out to stdout the IO values
func (s *ScreenDisplay) DisplayIO(fsbi ps_table.Tabler) {
	s.display(fsbi)
}

func (s *ScreenDisplay) DisplayLocks(tlwsbt ps_table.Tabler) {
	s.display(tlwsbt)
}

func (s *ScreenDisplay) DisplayMutex(ewsgben ps_table.Tabler) {
	s.display(ewsgben)
}

func (s *ScreenDisplay) DisplayOpsOrLatency(tiwsbt tiwsbt.Object) {
	s.display(tiwsbt)
}

func (s *ScreenDisplay) DisplayStages(essgben ps_table.Tabler) {
	s.display(essgben)
}

func (s *ScreenDisplay) DisplayUsers(users processlist.Object) {
	s.display(users)
}

func (s *ScreenDisplay) DisplayHelp() {

	s.screen.PrintAt(0, 0, lib.MyName()+" version "+version.Version()+" "+lib.Copyright())

	s.screen.PrintAt(0, 2, "Program to show the top I/O information by accessing information from the")
	s.screen.PrintAt(0, 3, "performance_schema schema. Ideas based on mysql-sys.")

	s.screen.PrintAt(0, 5, "Keys:")
	s.screen.PrintAt(0, 6, "- - reduce the poll interval by 1 second (minimum 1 second)")
	s.screen.PrintAt(0, 7, "+ - increase the poll interval by 1 second")
	s.screen.PrintAt(0, 8, "h/? - this help screen")
	s.screen.PrintAt(0, 9, "q - quit")
	s.screen.PrintAt(0, 10, "t - toggle between showing time since resetting statistics or since P_S data was collected")
	s.screen.PrintAt(0, 11, "z - reset statistics")
	s.screen.PrintAt(0, 12, "<tab> or <right arrow> - change display modes between: latency, ops, file I/O, lock and user modes")
	s.screen.PrintAt(0, 13, "<left arrow> - change display modes to the previous screen (see above)")
	s.screen.PrintAt(0, 15, "Press h to return to main screen")
}

func (s *ScreenDisplay) Resize(width, height int) {
	s.screen.SetSize(width, height)
}

func (s *ScreenDisplay) Close() {
	s.screen.Close()
}

// limit not used in ScreenDisplay
func (s *ScreenDisplay) Setup(limit int) {

	s.screen = new(screen.TermboxScreen)
	s.screen.Initialise()
	s.termboxChan = s.screen.TermBoxChan()
}

// convert screen to app events
func (s *ScreenDisplay) poll_event() event.Event {
	e := event.Event{Type: event.EventUnknown}
	select {
	case tb_event := <-s.termboxChan:
		switch tb_event.Type {
		case termbox.EventKey:
			switch tb_event.Ch {
			case '-':
				e = event.Event{Type: event.EventDecreasePollTime}
			case '+':
				e = event.Event{Type: event.EventIncreasePollTime}
			case 'h', '?':
				e = event.Event{Type: event.EventHelp}
			case 'q':
				e = event.Event{Type: event.EventFinished}
			case 't':
				e = event.Event{Type: event.EventToggleWantRelative}
			case 'z':
				e = event.Event{Type: event.EventResetStatistics}
			}
			switch tb_event.Key {
			case termbox.KeyCtrlZ, termbox.KeyCtrlC, termbox.KeyEsc:
				e = event.Event{Type: event.EventFinished}
			case termbox.KeyArrowLeft:
				e = event.Event{Type: event.EventViewPrev}
			case termbox.KeyTab, termbox.KeyArrowRight:
				e = event.Event{Type: event.EventViewNext}
			}
		case termbox.EventResize:
			e = event.Event{Type: event.EventResizeScreen, Width: tb_event.Width, Height: tb_event.Height}
		case termbox.EventError:
			e = event.Event{Type: event.EventError}
		}
	}
	return e
}

// create a channel for termbox.Events and run a poller to send
// these events to the channel.  Return the channel.
func (s *ScreenDisplay) EventChan() chan event.Event {
	eventChan := make(chan event.Event)
	go func() {
		for {
			eventChan <- s.poll_event()
		}
	}()
	return eventChan
}

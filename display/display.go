package display

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gdamore/tcell/termbox"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/event"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/screen"
	"github.com/sjmudd/ps-top/version"
)

// Display contains screen specific display information
type Display struct {
	ctx         *context.Context
	screen      *screen.Screen
	termboxChan chan termbox.Event
}

// NewDisplay returns a Display
func NewDisplay(ctx *context.Context) *Display {
	d := &Display{
		ctx:    ctx,
		screen: screen.NewScreen(),
	}
	d.termboxChan = d.screen.TermBoxChan()

	return d
}

// uptime returns ctx.uptime() protecting against nil pointers
func (s *Display) uptime() int {
	if s == nil || s.ctx == nil {
		return 0
	}
	return s.ctx.Uptime()
}

// Display displays the wanted view to the screen
func (s *Display) Display(t GenericData) {
	s.screen.PrintAt(0, 0, s.HeadingLine(t.HaveRelativeStats(), s.ctx.WantRelativeStats(), t.FirstCollectTime(), t.LastCollectTime()))
	s.screen.InvertedPrintAt(0, 1, t.Description())
	s.screen.BoldPrintAt(0, 2, t.Headings())

	maxRows := s.screen.Height() - 4
	lastRow := s.screen.Height() - 2
	bottomRow := s.screen.Height() - 1
	content := t.RowContent()

	for k := 0; k < maxRows; k++ {
		y := 3 + k
		if k <= len(content)-1 && k < maxRows {
			// print out rows
			s.screen.PrintAt(0, y, content[k])
			s.screen.ClearLine(len(content[k]), y)
		} else {
			// print out empty rows
			if y < lastRow {
				s.screen.PrintAt(0, y, t.EmptyRowContent())
			}
		}
	}

	// print out the totals at the bottom
	total := t.TotalRowContent()
	s.screen.BoldPrintAt(0, lastRow, total)
	s.screen.ClearLine(len(total), lastRow)

	menu := "[+-] Delay  [<] Prev  [>] Next  [h]elp  [r] Abs/Rel  [q]uit  [z] Reset stats"
	s.screen.PrintAt(0, bottomRow, menu)
	s.screen.ClearLine(len(menu), bottomRow)
}

// ClearScreen clears the (internal) screen and flushes out the result to the real screen
func (s *Display) ClearScreen() {
	s.screen.Clear()
	s.screen.Flush()
}

// DisplayHelp displays a help page on the screen
func (s *Display) DisplayHelp() {
	s.screen.PrintAt(0, 0, lib.ProgName+" version "+version.Version+" "+lib.Copyright)

	s.screen.PrintAt(0, 2, "Program to show the top I/O information by accessing information from the")
	s.screen.PrintAt(0, 3, "performance_schema schema. Ideas based on mysql-sys.")

	s.screen.PrintAt(0, 5, "Keys:")
	s.screen.PrintAt(0, 6, "- - reduce the poll interval by 1 second (minimum 1 second)")
	s.screen.PrintAt(0, 7, "+ - increase the poll interval by 1 second")
	s.screen.PrintAt(0, 8, "h/? - this help screen")
	s.screen.PrintAt(0, 9, "q - quit")
	s.screen.PrintAt(0, 10, "s - sort differently (where enabled) - sorts on a different column")
	s.screen.PrintAt(0, 11, "t - toggle between showing time since resetting statistics or since P_S data was collected")
	s.screen.PrintAt(0, 12, "z - reset statistics")
	s.screen.PrintAt(0, 13, "<tab> or <right arrow> - change display modes between: latency, ops, file I/O, lock and user modes")
	s.screen.PrintAt(0, 14, "<left arrow> - change display modes to the previous screen (see above)")
	s.screen.PrintAt(0, 16, "Press h to return to main screen")
}

// Resize records the new size of the screen and resizes it
func (s *Display) Resize(width, height int) {
	s.screen.SetSize(width, height)
}

// Close is called prior to closing the screen
func (s *Display) Close() {
	s.screen.Close()
}

// convert screen to app events
func (s *Display) pollEvent() event.Event {
	e := event.Event{Type: event.EventUnknown}
	select {
	case tbEvent := <-s.termboxChan:
		switch tbEvent.Type {
		case termbox.EventKey:
			switch tbEvent.Ch {
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
			switch tbEvent.Key {
			case termbox.KeyCtrlZ, termbox.KeyCtrlC, termbox.KeyEsc:
				e = event.Event{Type: event.EventFinished}
			case termbox.KeyArrowLeft:
				e = event.Event{Type: event.EventViewPrev}
			case termbox.KeyTab, termbox.KeyArrowRight:
				e = event.Event{Type: event.EventViewNext}
			}
		case termbox.EventResize:
			e = event.Event{Type: event.EventResizeScreen, Width: tbEvent.Width, Height: tbEvent.Height}
		case termbox.EventError:
			e = event.Event{Type: event.EventError}
		}
	}
	return e
}

// EventChan creates a channel of display events and run a poller to send
// these events to the channel.  Return the channel which the application can use
func (s *Display) EventChan() chan event.Event {
	eventChan := make(chan event.Event)
	go func() {
		for {
			eventChan <- s.pollEvent()
		}
	}()
	return eventChan
}

// Uptime provides a usable form of uptime.
// Note: this doesn't return a string of a fixed size!
// Minimum value: 1s.
// Maximum value: 100d 23h 59m 59s (sort of).
func uptime(uptime int) string {
	var result string

	days := uptime / 24 / 60 / 60
	hours := (uptime - days*86400) / 3600
	minutes := (uptime - days*86400 - hours*3600) / 60
	seconds := uptime - days*86400 - hours*3600 - minutes*60

	result = strconv.Itoa(seconds) + "s"

	if minutes > 0 {
		result = strconv.Itoa(minutes) + "m " + result
	}
	if hours > 0 {
		result = strconv.Itoa(hours) + "h " + result
	}
	if days > 0 {
		result = strconv.Itoa(days) + "d " + result
	}

	return result
}

// HeadingLine returns the heading line as a string
func (s *Display) HeadingLine(haveRelativeStats, wantRelativeStats bool, initial, last time.Time) string {
	heading := lib.ProgName + " " + version.Version + " - " + now() + " " + s.ctx.Hostname() + " / " + s.ctx.MySQLVersion() + ", up " + fmt.Sprintf("%-16s", uptime(s.uptime()))

	if haveRelativeStats {
		if wantRelativeStats {
			heading += " [REL] " + fmt.Sprintf("%.0f seconds", time.Since(initial).Seconds())
		} else {
			heading += " [ABS]             "
		}
	}
	return heading
}

// now returns the time in format hh:mm:ss
func now() string {
	t := time.Now()
	return fmt.Sprintf("%2d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

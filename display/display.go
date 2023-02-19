package display

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gdamore/tcell/termbox"

	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/event"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/screen"
	"github.com/sjmudd/ps-top/version"
)

// Display contains screen specific display information
type Display struct {
	cfg         *config.Config
	screen      *screen.Screen
	termboxChan chan termbox.Event
}

// NewDisplay returns a Display
func NewDisplay(cfg *config.Config) *Display {
	display := &Display{
		cfg:    cfg,
		screen: screen.NewScreen(lib.ProgName),
	}
	display.termboxChan = display.screen.TermBoxChan()

	return display
}

// uptime returns cfg.uptime() protecting against nil pointers
func (display *Display) uptime() int {
	if display == nil || display.cfg == nil {
		return 0
	}
	return display.cfg.Uptime()
}

// Display displays the wanted view to the screen
func (display *Display) Display(t GenericData) {
	heading := display.HeadingLine(t.HaveRelativeStats(), display.cfg.WantRelativeStats(), t.FirstCollectTime(), t.LastCollectTime())
	description := t.Description()
	headings := t.Headings()

	display.screen.PrintAt(0, 0, heading)
	display.screen.ClearLine(len(heading), 0)

	display.screen.InvertedPrintAt(0, 1, description)
	display.screen.ClearLine(len(description), 1)

	display.screen.BoldPrintAt(0, 2, headings)
	display.screen.ClearLine(len(headings), 2)

	maxRows := display.screen.Height() - 4
	lastRow := display.screen.Height() - 2
	bottomRow := display.screen.Height() - 1
	content := t.RowContent()

	for k := 0; k < maxRows; k++ {
		y := 3 + k
		if k <= len(content)-1 && k < maxRows {
			// print out rows
			display.screen.PrintAt(0, y, content[k])
			display.screen.ClearLine(len(content[k]), y)
		} else {
			// print out empty rows
			if y < lastRow {
				display.screen.PrintAt(0, y, t.EmptyRowContent())
			}
		}
	}

	// print out the totals at the bottom
	total := t.TotalRowContent()
	display.screen.BoldPrintAt(0, lastRow, total)
	display.screen.ClearLine(len(total), lastRow)

	menu := "[+-] Delay  [<] Prev  [>] Next  [h]elp  [r] Abs/Rel  [q]uit  [z] Reset stats"
	display.screen.PrintAt(0, bottomRow, menu)
	display.screen.ClearLine(len(menu), bottomRow)
}

// ClearScreen clears the (internal) screen and flushes out the result to the real screen
func (display *Display) ClearScreen() {
	display.screen.Clear()
	display.screen.Flush()
}

// DisplayHelp displays a help page on the screen
func (display *Display) DisplayHelp() {
	display.screen.PrintAt(0, 0, lib.ProgName+" version "+version.Version+" "+lib.Copyright)

	display.screen.PrintAt(0, 2, "Program to show the top I/O information by accessing information from the")
	display.screen.PrintAt(0, 3, "performance_schema schema. Ideas based on mysql-sys.")

	display.screen.PrintAt(0, 5, "Keys:")
	display.screen.PrintAt(0, 6, "- - reduce the poll interval by 1 second (minimum 1 second)")
	display.screen.PrintAt(0, 7, "+ - increase the poll interval by 1 second")
	display.screen.PrintAt(0, 8, "h/? - this help screen")
	display.screen.PrintAt(0, 9, "q - quit")
	display.screen.PrintAt(0, 10, "s - sort differently (where enabled) - sorts on a different column")
	display.screen.PrintAt(0, 11, "t - toggle between showing time since resetting statistics or since P_S data was collected")
	display.screen.PrintAt(0, 12, "z - reset statistics")
	display.screen.PrintAt(0, 13, "<tab> or <right arrow> - change display modes between: latency, ops, file I/O, lock and user modes")
	display.screen.PrintAt(0, 14, "<left arrow> - change display modes to the previous screen (see above)")
	display.screen.PrintAt(0, 16, "Press h to return to main screen")
}

// Resize records the new size of the screen and resizes it
func (display *Display) Resize(width, height int) {
	display.screen.SetSize(width, height)
}

// Close is called prior to closing the screen
func (display *Display) Close() {
	display.screen.Close()
}

// convert screen to app events
func (display *Display) pollEvent() event.Event {
	e := event.Event{Type: event.EventUnknown}
	tbEvent := <-display.termboxChan
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
	return e
}

// EventChan creates a channel of display events and run a poller to send
// these events to the channel.  Return the channel which the application can use
func (display *Display) EventChan() chan event.Event {
	eventChan := make(chan event.Event)
	go func() {
		for {
			eventChan <- display.pollEvent()
		}
	}()
	return eventChan
}

// Uptime provides a usable form of uptime.
// Note: this doesn't return a string of a fixed size!
// Minimum value: 1s.
// Maximum value: 999d 23h 59m 59s (sort of).
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
func (display *Display) HeadingLine(haveRelativeStats, wantRelativeStats bool, initial, last time.Time) string {
	heading := lib.ProgName + " " + version.Version + " - " + now() + " " + display.cfg.Hostname() + " / " + display.cfg.MySQLVersion() + ", up " + fmt.Sprintf("%-16s", uptime(display.uptime()))

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

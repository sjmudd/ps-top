package display

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tcell "github.com/gdamore/tcell/v2"

	"github.com/sjmudd/ps-top/event"
	"github.com/sjmudd/ps-top/log"
	"github.com/sjmudd/ps-top/utils"
)

const endOfLineFiller = rune(' ')

var (
	whiteOnBlackStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	topLineStyle      = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGrey)
	descriptionStyle  = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorTeal)
	headingStyle      = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	tableStyle        = tcell.StyleDefault.Foreground(tcell.ColorGrey).Background(tcell.ColorBlack)
	menuStyle         = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGrey)
	menuTextStyle     = tcell.StyleDefault.Foreground(tcell.ColorDarkRed).Background(tcell.ColorGrey)
	bracketStyle      = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGrey)
	defaultStyle      = whiteOnBlackStyle
)

// Config provides the interfce to some required configuration settings needed by Display
type Config interface {
	Hostname() string
	MySQLVersion() string
	WantRelativeStats() bool
	Uptime() int
}

// Display contains screen specific display information
type Display struct {
	config    Config
	screen    tcell.Screen
	tcellChan chan tcell.Event
	height    int // display height
	width     int // display width
}

// NewDisplay returns a Display with an empty terminal
func NewDisplay(config Config) *Display {
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("tcell.NewScreen() failed: %+v", err)
	}

	if err := screen.Init(); err != nil {
		log.Fatalf("tcell.Init() failed: %+v", err)
	}
	screen.SetStyle(defaultStyle)
	screen.Clear()
	screen.Sync()

	// record the initial screen size
	width, height := screen.Size()

	return &Display{
		config:    config,
		screen:    screen,
		tcellChan: tcellPoller(screen),
		height:    height,
		width:     width,
	}
}

// uptime returns config.uptime() protecting against nil pointers
func (display *Display) uptime() int {
	if display == nil || display.config == nil {
		return 0
	}
	return display.config.Uptime()
}

// printLine prints the characters on the line and fills out to the
// right of the screen with a space character
func (display *Display) printLine(y int, text string, style tcell.Style) {
	log.Printf("printLine(y: %v, text: %q (%d), style: %v): width: %d", y, text, len([]rune(text)), style, display.width)

	// extend text to screen width with spaces
	L := len([]rune(text))
	if L < display.width {
		// extend string length to display width
		text = text + strings.Repeat(" ", display.width-L)
	}

	x := 0
	for _, r := range text {
		if x < display.width {
			display.screen.SetContent(x, y, r, nil, style)
			x++
		}
	}
}

// printTableData displays the provided content, filling lines with an empty row if needed
func (display *Display) printTableData(content []string, lastRow, maxRows int, emptyRow string, style tcell.Style) {
	for k := 0; k < maxRows; k++ {
		y := 3 + k
		if k <= len(content)-1 && k < maxRows {
			display.printLine(y, content[k], style)
		} else {
			if y < lastRow {
				display.printLine(y, emptyRow, style)
			}
		}
	}
}

// printMenu prints the menu bar at the bottom
// - styling - normally inverted style (black on grey), except between [ ] where we use tcell.ColorBlue
func (display *Display) printMenu(bottomRow int) {
	const (
		menu         = "[+-] Delay  [<] Prev  [>] Next  [h]elp  [r] Abs/Rel  [q]uit  [z] Reset stats"
		openBracket  = rune('[')
		closeBracket = rune(']')
	)

	style := menuStyle
	x := 0
	for _, r := range menu {
		nextStyle := style
		if r == openBracket {
			style = bracketStyle
			nextStyle = menuTextStyle
		}
		if r == closeBracket {
			style = bracketStyle
			nextStyle = menuStyle
		}

		if x < display.width {
			display.screen.SetContent(x, bottomRow, r, nil, style)
		}

		style = nextStyle
		x++
	}
	// inverted to end of line
	for x2 := x; x2 < display.width; x2++ {
		display.screen.SetContent(x2, bottomRow, endOfLineFiller, nil, menuStyle)
	}
}

// Clear clears the screen and flushes out the result to the terminal
func (display *Display) Clear() {
	display.screen.Clear()
	display.screen.Sync()
}

// Display displays the wanted view to the screen
func (display *Display) Display(gd GenericData) {
	maxRows := display.height - 5   // maximum number of rows we can show (taking into account headers/footers)
	lastRow := display.height - 2   // last row where we can print things
	bottomRow := display.height - 1 // the bottom row where the menu goes

	display.printLine(0,
		display.generateTopLine(
			gd.HaveRelativeStats(),
			display.config.WantRelativeStats(),
			gd.FirstCollectTime(),
			gd.LastCollectTime(),
			display.width,
		),
		topLineStyle)
	display.printLine(1, gd.Description(), descriptionStyle) // display table description
	display.printLine(2, gd.Headings(), headingStyle)
	// display table headings, data and totals
	display.printTableData(gd.RowContent(), lastRow, maxRows, gd.EmptyRowContent(), tableStyle)
	display.printLine(lastRow, gd.TotalRowContent(), defaultStyle)
	display.printMenu(bottomRow)

	display.screen.Show()
}

// Resize records the new size of the screen and clears it
func (display *Display) Resize(width, height int) {
	log.Printf("Display.Resize(width: %v, height: %v), previous values: (width: %v, height: %v)", width, height, display.width, display.height)
	display.screen.Clear()
	display.screen.Sync()

	display.width = width
	display.height = height
}

// Fini is called prior to finishing using the display
func (display *Display) Fini() {
	display.screen.Fini()
}

// convert screen to app events
func (display *Display) poll() event.Event {
	e := event.Event{Type: event.EventUnknown}

	tcellEvent := <-display.tcellChan
	switch tcellEvent.(type) {
	case *tcell.EventKey:
		log.Printf("tcell.EventKey: %+v", tcellEvent)
		ev := tcellEvent.(*tcell.EventKey)
		switch ev.Key() {
		case tcell.KeyCtrlZ, tcell.KeyCtrlC, tcell.KeyEsc:
			e = event.Event{Type: event.EventFinished}
		case tcell.KeyLeft:
			e = event.Event{Type: event.EventViewPrev}
		case tcell.KeyTab, tcell.KeyRight:
			e = event.Event{Type: event.EventViewNext}
		case tcell.KeyRune:
			switch ev.Rune() {
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
			}
		}
	case *tcell.EventResize:
		ev := tcellEvent.(*tcell.EventResize)
		width, height := ev.Size()
		log.Printf("poll: EventResize: width: %v, height: %v", width, height)
		e = event.Event{Type: event.EventResizeScreen, Width: width, Height: height}
	case *tcell.EventError:
		e = event.Event{Type: event.EventError}
	default:
		log.Printf("poll: received unexpected event: %+v", tcellEvent)
	}

	return e
}

// EventChan creates a channel of display events and run a poller to send
// these events to the channel.  Return the channel which the application can use
func (display *Display) EventChan() chan event.Event {
	eventChan := make(chan event.Event)
	go func() {
		for {
			eventChan <- display.poll()
		}
	}()
	return eventChan
}

// generateTopLine returns the heading line as a string
func (display *Display) generateTopLine(haveRelativeStats, wantRelativeStats bool, initial, last time.Time, width int) string {
	heading := utils.ProgName + " " +
		utils.Version + " - " +
		now() + " " +
		display.config.Hostname() + " / " +
		display.config.MySQLVersion() + ", up " +
		fmt.Sprintf("%-16s", uptime(display.uptime()))

	if haveRelativeStats {
		var suffix string
		if wantRelativeStats {
			suffix = " [REL] " + fmt.Sprintf("%.0f seconds", time.Since(initial).Seconds())
		} else {
			suffix = " [ABS]             "
		}
		if len(heading)+len(suffix) < width {
			heading += strings.Repeat(" ", width-len(heading)-len(suffix)) + suffix
		}
	}
	return heading
}

// now returns the current time in format hh:mm:ss
func now() string {
	t := time.Now()
	return fmt.Sprintf("%2d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

// Uptime provides a usable form of the MySQL uptime status variable provided in seconds.
//
// Note: this doesn't return a string of a fixed size!
// Minimum value: 0s.
// Maximum value: 999d 23h 59m 59s (sort of).
func uptime(uptime int) string {
	days := uptime / 24 / 60 / 60
	hours := (uptime - days*86400) / 3600
	minutes := (uptime - days*86400 - hours*3600) / 60
	seconds := uptime - days*86400 - hours*3600 - minutes*60

	result := strconv.Itoa(seconds) + "s"

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

// TCellChan creates a channel for tcell.Events and runs a poller to send
// these events to the channel.  Return the channel to the caller..
// FIXME:: Provide a way to stop the go routine on shutdown.
func tcellPoller(screen tcell.Screen) chan tcell.Event {
	eventChannel := make(chan tcell.Event)
	go func() {
		for {
			eventChannel <- screen.PollEvent()
		}
	}()

	return eventChannel
}

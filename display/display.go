package display

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gdamore/tcell"

	"github.com/sjmudd/ps-top/event"
	"github.com/sjmudd/ps-top/log"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/version"
)

var (
	whiteOnBlackStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	blackOnGreyStyle  = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGrey)
	menuLetterStyle   = tcell.StyleDefault.Foreground(tcell.ColorMaroon).Background(tcell.ColorGrey)
	topLineStyle      = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGrey)
	headingLineStyle  = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorTeal)
	tableStyle        = tcell.StyleDefault.Foreground(tcell.ColorGrey).Background(tcell.ColorBlack)
	defaultStyle      = whiteOnBlackStyle
	invertedStyle     = blackOnGreyStyle
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
	config       Config
	screen       tcell.Screen
	tcellChannel chan tcell.Event
	height       int // display height
	width        int // display width
}

// NewDisplay returns a Display with an empty terminal
func NewDisplay(config Config) *Display {
	tcscreen, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("tcell.NewScreen() failed: %+v", err)
	}

	if err := tcscreen.Init(); err != nil {
		log.Fatalf("tcell.Init() failed: %+v", err)
	}
	tcscreen.SetStyle(defaultStyle)
	tcscreen.Clear()
	tcscreen.Sync()

	// record the initial screen size
	width, height := tcscreen.Size()

	return &Display{
		config:       config,
		screen:       tcscreen,
		tcellChannel: setupScreenPoller(tcscreen),
		height:       height,
		width:        width,
	}
}

// uptime returns config.uptime() protecting against nil pointers
func (display *Display) uptime() int {
	if display == nil || display.config == nil {
		return 0
	}
	return display.config.Uptime()
}

// Show pushes out the pending changes to the screen
func (display *Display) Show() {
	display.screen.Show()
}

// printAtUsingStyle prints the characters at the requested location using the given style
func (display *Display) printAtUsingStyle(x int, y int, text string, style tcell.Style) {
	log.Printf("Screen.printAtUsingStyle(x: %v, y: %v, text: %q, style: %v)", x, y, text, style)
	display.screen.SetCell(x, y, style, []rune(text)...)
	display.screen.Show()
}

// clearLineUsingStyle will clear to end of line with the given style
func (display *Display) clearLineUsingStyle(x, y int, style tcell.Style) {
	log.Printf("Screen.clearLineUsingStyle(x: %v, y: %v, style: %v, width: %v)", x, y, style, display.width)

	for i := x; i < display.width; i++ {
		display.screen.SetCell(i, y, style, ' ')
	}

	display.screen.Show()
}

// display table data
func (display *Display) displayTableData(content []string, lastRow, maxRows int, emptyRow string) {
	for k := 0; k < maxRows; k++ {
		y := 3 + k
		if k <= len(content)-1 && k < maxRows {
			// print out rows
			display.printAtUsingStyle(0, y, content[k], tableStyle)
			display.clearLineUsingStyle(len(content[k]), y, tableStyle)
		} else {
			// print out empty rows
			if y < lastRow {
				display.printAtUsingStyle(0, y, emptyRow, tableStyle)
				display.clearLineUsingStyle(len(emptyRow), y, tableStyle)
			}
		}
	}
}

// printBottomMenuAt prints the menu bar at the bottom
// - styling - normally inverted style, except between [ ] where we use tcell.ColorBlue
func (display *Display) printBottomMenuAt(x int, y int, menu string) {
	currentStyle := invertedStyle
	ch := ' '
	for x := 0; x < display.width; x++ {
		if x < len(menu) {
			ch = rune(menu[x])
			if ch == ']' {
				currentStyle = invertedStyle
			}
		} else {
			currentStyle = invertedStyle
			ch = ' '
		}

		display.screen.SetCell(x, y, currentStyle, ch)

		if x < len(menu) {
			if ch == '[' {
				currentStyle = menuLetterStyle
			}
		}
	}
}

func (display *Display) displayBottomRowMenu(bottomRow int) {
	const bottomRowMenu = "[+-] Delay  [<] Prev  [>] Next  [h]elp  [r] Abs/Rel  [q]uit  [z] Reset stats"

	display.printBottomMenuAt(0, bottomRow, bottomRowMenu)
	display.clearLineUsingStyle(len(bottomRowMenu), bottomRow, invertedStyle)
	display.Show()
}

// topLinePrintAt displays the top line to the end of the line
func (display *Display) topLinePrintAt(x int, y int, text string) {
	l := len(text)
	display.screen.SetCell(x, y, topLineStyle, []rune(text)...)
	display.clearLineUsingStyle(x+l, y, topLineStyle)
	display.Show()
}

// headingLinePrintAt displays the heading line to the end of the line
func (display *Display) headingLinePrintAt(x int, y int, text string) {
	l := len(text)
	display.screen.SetCell(x, y, headingLineStyle, []rune(text)...)
	display.clearLineUsingStyle(x+l, y, headingLineStyle)
	display.Show()
}

// tableHeadingsLinePrintAt displays the table headings line to the end of the line
func (display *Display) tableHeadingsLinePrintAt(x int, y int, text string) {
	l := len(text)
	display.screen.SetCell(x, y, defaultStyle, []rune(text)...)
	display.clearLineUsingStyle(x+l, y, defaultStyle)
	display.Show()
}

// totalsPrintLineAt displays the table totals line to the end of the line
func (display *Display) totalsPrintLineAt(x int, y int, text string) {
	l := len(text)
	display.screen.SetCell(x, y, defaultStyle, []rune(text)...)
	display.clearLineUsingStyle(x+l, y, defaultStyle)
	display.Show()
}

// invertedPrintAt displays text inverting background and foreground
// colours at the location specified, but does not try to display
// outside of the screen boundary.
func (display *Display) invertedPrintAt(x int, y int, text string) {
	display.screen.SetCell(x, y, invertedStyle, []rune(text)...)
	display.Show()
}

// Display displays the wanted view to the screen
func (display *Display) Display(gd GenericData) {
	var (
		maxRows   int // maximum number of rows we can show (taking into account headers/footers)
		lastRow   int // last row where we can print things
		bottomRow int // the bottom row where the menu goes
	)

	maxRows = display.height - 4
	lastRow = display.height - 2
	bottomRow = display.height - 1

	// display the top line
	display.topLinePrintAt(
		0,
		0,
		display.headingLine(
			gd.HaveRelativeStats(),
			display.config.WantRelativeStats(),
			gd.FirstCollectTime(),
			gd.LastCollectTime(),
		),
	)

	// display table description
	display.headingLinePrintAt(
		0,
		1,
		gd.Description(),
	)

	// display table headings
	display.tableHeadingsLinePrintAt(
		0,
		2,
		gd.Headings(),
	)

	display.displayTableData(gd.RowContent(), lastRow, maxRows, gd.EmptyRowContent())

	// display table totals
	display.totalsPrintLineAt(
		0,
		lastRow,
		gd.TotalRowContent(),
	)

	display.displayBottomRowMenu(bottomRow)
}

// Clear clears the screen and flushes out the result to the terminal
func (display *Display) Clear() {
	display.screen.Clear()
	display.screen.Sync()
}

// Resize records the new size of the screen and resizes it
// - if the terminal gets smaller assume that the larger areas are just truncated so we do nothing.
// - if the terminal gets longer then clear out the bottom line(s).
// - if the terminal gets wider it should be "blank" so no need to do anything.
func (display *Display) Resize(width, height int) {
	log.Printf("Display.Resize(width: %v, height: %v), previous values: (width: %v, height: %v)", width, height, display.width, display.height)
	if height > display.height {
		for x := 0; x < display.width; x++ {
			display.screen.SetCell(x, display.height-1, defaultStyle, ' ')
		}
		display.screen.Sync()
	}
	display.width = width
	display.height = height
}

// Fini is called prior to finishing using the display
func (display *Display) Fini() {
	display.screen.Fini()
}

// convert screen to app events
func (display *Display) pollEvent() event.Event {
	e := event.Event{Type: event.EventUnknown}
	tcellEvent := <-display.tcellChannel
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
		log.Printf("pollEvent: EventResize: width: %v, height: %v", width, height)
		e = event.Event{Type: event.EventResizeScreen, Width: width, Height: height}
	case *tcell.EventError:
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

// headingLine returns the heading line as a string
func (display *Display) headingLine(haveRelativeStats, wantRelativeStats bool, initial, last time.Time) string {
	heading := utils.ProgName + " " +
		version.Version + " - " +
		now() + " " +
		display.config.Hostname() + " / " +
		display.config.MySQLVersion() + ", up " +
		fmt.Sprintf("%-16s", uptime(display.uptime()))

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

// TCellChan creates a channel for tcell.Events and runs a poller to send
// these events to the channel.  Return the channel to the caller..
// FIXME:: Provide a way to stop the go routine on shutdown.
func setupScreenPoller(screen tcell.Screen) chan tcell.Event {
	eventChannel := make(chan tcell.Event)
	go func() {
		for {
			eventChannel <- screen.PollEvent()
		}
	}()

	return eventChannel
}

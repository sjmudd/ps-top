// Package screen configures the screen, basically remembering the size
// and foreground and background colours.
package screen

import (
	"fmt"
	"log"
	"os"

	"github.com/gdamore/tcell/termbox"

	"github.com/sjmudd/ps-top/lib"
)

// Screen is a wrapper around termbox
type Screen struct {
	height int
	width  int
	bg     termbox.Attribute
	fg     termbox.Attribute
}

// NewScreen initialises and returns a Screen
func NewScreen() *Screen {
	s := new(Screen)
	s.Initialise()

	return s
}

// BoldPrintAt displays bold text at the location specified, but
// does not try to display outside of the screen boundary.
func (s *Screen) BoldPrintAt(x int, y int, text string) {
	offset := 0
	for c := range text {
		if (x + offset) < s.width {
			termbox.SetCell(x+offset, y, rune(text[c]), s.fg|termbox.AttrBold, s.bg)
			offset++
		}
	}
	s.Flush()
}

// InvertedPrintAt displays text inverting background and foreground
// colours at the location specified, but does not try to display
// outside of the screen boundary.
func (s *Screen) InvertedPrintAt(x int, y int, text string) {
	offset := 0
	for c := range text {
		if (x + offset) < s.width {
			termbox.SetCell(x+offset, y, rune(text[c]), s.bg, s.fg)
			offset++
		}
	}
	s.Flush()
}

// Clear clears the screen
func (s *Screen) Clear() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
}

// Close closes the screen prior to shutdown
func (s *Screen) Close() {
	termbox.Close()
}

// Flush pushes out the pending changes to the screen
func (s *Screen) Flush() {
	termbox.Flush()
}

// Height returns the current height of the screen
func (s *Screen) Height() int {
	return s.height
}

// Initialise initialises the screen and clears it on startup
func (s *Screen) Initialise() {
	if err := termbox.Init(); err != nil {
		fmt.Println("Could not start termbox for " + lib.ProgName + ". View ~/." + lib.ProgName + ".log for error messages.")
		log.Printf("Cannot start "+lib.ProgName+", termbox.Init() gave an error:\n%s\n", err)
		os.Exit(1)
	}

	s.Clear()
	s.fg = termbox.ColorWhite
	s.bg = termbox.ColorBlack

	s.SetSize(termbox.Size())
}

// PrintAt prints the characters at the requested location while they fit in the screen
func (s *Screen) PrintAt(x int, y int, text string) {
	offset := 0
	for c := range text {
		if (x + offset) < s.width {
			termbox.SetCell(x+offset, y, rune(text[c]), s.fg, s.bg)
			offset++
		}
	}
	s.Flush()
}

// ClearLine clears the line with spaces to the right hand side of the screen
func (s *Screen) ClearLine(x int, y int) {
	for i := x; i < s.width; i++ {
		termbox.SetCell(i, y, ' ', termbox.ColorDefault, termbox.ColorDefault)
	}
	s.Flush()
}

// SetSize records the size of the screen and if the terminal gets
// longer then clear out the bottom line.
func (s *Screen) SetSize(width, height int) {
	if height > s.height {
		for x := 0; x < s.width; x++ {
			termbox.SetCell(x, s.height-1, ' ', s.fg, s.bg)
		}
		s.Flush()
	}

	s.width = width
	s.height = height
}

// Size returns the current (width, height) of the screen
func (s *Screen) Size() (int, int) {
	return s.width, s.height
}

// TermBoxChan creates a channel for termbox.Events and run a poller to send
// these events to the channel.  Return the channel to the caller..
func (s Screen) TermBoxChan() chan termbox.Event {
	termboxChan := make(chan termbox.Event)
	go func() {
		for {
			termboxChan <- termbox.PollEvent()
		}
	}()
	return termboxChan
}

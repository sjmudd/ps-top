// Package screen configures the screen, basically remembering the size
// and foreground and background colours.
package screen

import (
	"fmt"
	"log"
	"os"

	"github.com/nsf/termbox-go"

	"github.com/sjmudd/ps-top/lib"
)

// TermboxScreen is a wrapper around termbox
type TermboxScreen struct {
	width, height int
	fg, bg        termbox.Attribute
}

// BoldPrintAt displays bold text at the location specified, but
// does not try to display outside of the screen boundary.
func (s *TermboxScreen) BoldPrintAt(x int, y int, text string) {
	offset := 0
	for c := range text {
		if (x + offset) < s.width {
			termbox.SetCell(x+offset, y, rune(text[c]), s.fg|termbox.AttrBold, s.bg)
			offset++
		}
	}
	s.Flush()
}

// Clear clears the screen
func (s *TermboxScreen) Clear() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
}

// Close closes the screen prior to shutdown
func (s *TermboxScreen) Close() {
	termbox.Close()
}

// Flush pushes out the pending changes to the screen
func (s *TermboxScreen) Flush() {
	termbox.Flush()
}

// Height returns the current height of the screen
func (s *TermboxScreen) Height() int {
	return s.height
}

// Initialise initialises the screen and clears it on startup
func (s *TermboxScreen) Initialise() {
	err := termbox.Init()
	if err != nil {
		fmt.Println("Could not start termbox for " + lib.ProgName + ". View ~/." + lib.ProgName + ".log for error messages.")
		log.Printf("Cannot start "+lib.ProgName+", termbox.Init() gave an error:\n%s\n", err)
		os.Exit(1)
	}

	s.Clear()
	s.fg = termbox.ColorDefault
	s.bg = termbox.ColorDefault

	s.SetSize(termbox.Size())
}

// PrintAt prints the characters at the requested location while they fit in the screen
func (s *TermboxScreen) PrintAt(x int, y int, text string) {
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
func (s *TermboxScreen) ClearLine(x int, y int) {
	for i := x; i < s.width; i++ {
		termbox.SetCell(i, y, ' ', termbox.ColorDefault, termbox.ColorDefault)
	}
	s.Flush()
}

// SetSize records the size of the screen
func (s *TermboxScreen) SetSize(width, height int) {
	// if we get bigger then clear out the bottom line
	for x := 0; x < s.width; x++ {
		termbox.SetCell(x, s.height-1, ' ', s.fg, s.bg)
	}
	s.Flush()

	s.width = width
	s.height = height
}

// Size returns the current (width, height) of the screen
func (s *TermboxScreen) Size() (int, int) {
	return s.width, s.height
}

// TermBoxChan creates a channel for termbox.Events and run a poller to send
// these events to the channel.  Return the channel to the caller..
func (s TermboxScreen) TermBoxChan() chan termbox.Event {
	termboxChan := make(chan termbox.Event)
	go func() {
		for {
			termboxChan <- termbox.PollEvent()
		}
	}()
	return termboxChan
}

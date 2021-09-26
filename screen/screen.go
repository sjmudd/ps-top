// Package screen configures the screen, basically remembering the size
// and foreground and background colours.
package screen

import (
	"fmt"
	"log"
	"os"

	"github.com/gdamore/tcell/termbox"
)

// Screen is a wrapper around termbox
type Screen struct {
	height int
	width  int
	bg     termbox.Attribute
	fg     termbox.Attribute
}

// NewScreen initialises a screen, clearing it, returning a *Screen
func NewScreen(program string) *Screen {
	s := new(Screen)

	if err := termbox.Init(); err != nil {
		fmt.Printf("Cannot start %v: %+v", program, err)
		log.Printf("Cannot start %v: %+v", program, err)
		os.Exit(1)
	}

	s.Clear()
	s.fg = termbox.ColorWhite
	s.bg = termbox.ColorBlack

	s.SetSize(termbox.Size())

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
func (screen *Screen) Clear() {
	termbox.Clear(screen.fg, screen.bg)
}

// Close closes the screen prior to shutdown
func (screen *Screen) Close() {
	termbox.Close()
}

// Flush pushes out the pending changes to the screen
func (screen *Screen) Flush() {
	termbox.Flush()
}

// Height returns the current height of the screen
func (screen *Screen) Height() int {
	return screen.height
}

// PrintAt prints the characters at the requested location while they fit in the screen
func (screen *Screen) PrintAt(x int, y int, text string) {
	offset := 0
	for c := range text {
		if (x + offset) < screen.width {
			termbox.SetCell(x+offset, y, rune(text[c]), screen.fg, screen.bg)
			offset++
		}
	}
	screen.Flush()
}

// ClearLine clears the line with spaces to the right hand side of the screen
func (screen *Screen) ClearLine(x int, y int) {
	for i := x; i < screen.width; i++ {
		termbox.SetCell(i, y, ' ', screen.fg, screen.bg)
	}
	screen.Flush()
}

// SetSize records the size of the screen and if the terminal gets
// longer then clear out the bottom line.
func (screen *Screen) SetSize(width, height int) {
	if height > screen.height {
		for x := 0; x < screen.width; x++ {
			termbox.SetCell(x, screen.height-1, ' ', screen.fg, screen.bg)
		}
		screen.Flush()
	}

	screen.width = width
	screen.height = height
}

// Size returns the current (width, height) of the screen
func (screen *Screen) Size() (int, int) {
	return screen.width, screen.height
}

// TermBoxChan creates a channel for termbox.Events and run a poller to send
// these events to the channel.  Return the channel to the caller..
func (screen Screen) TermBoxChan() chan termbox.Event {
	termboxChan := make(chan termbox.Event)
	go func() {
		for {
			termboxChan <- termbox.PollEvent()
		}
	}()
	return termboxChan
}

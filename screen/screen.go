// This file configures the screen, basically remembering the size
// and foreground and background colours.
package screen

import (
	"fmt"
	"log"
	"os"

	"github.com/nsf/termbox-go"

	"github.com/sjmudd/pstop/lib"
	"github.com/sjmudd/pstop/version"
)

// this just allows me to use stuff with it
type TermboxScreen struct {
	width, height int
	fg, bg        termbox.Attribute
}

// print the characters in bold (for headings) but don't print them outside the screen
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

// clear the screen
func (s *TermboxScreen) Clear() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
}

// close the screen
func (s *TermboxScreen) Close() {
	termbox.Close()
}

// display a help page
func (s *TermboxScreen) DisplayHelp() {
	s.PrintAt(0, 0, lib.MyName()+" version "+version.Version()+" "+lib.Copyright())

	s.PrintAt(0, 2, "Program to show the top I/O information by accessing information from the")
	s.PrintAt(0, 3, "performance_schema schema. Ideas based on mysql-sys.")

	s.PrintAt(0, 5, "Keys:")
	s.PrintAt(0, 6, "- - reduce the poll interval by 1 second (minimum 1 second)")
	s.PrintAt(0, 7, "+ - increase the poll interval by 1 second")
	s.PrintAt(0, 8, "h/? - this help screen")
	s.PrintAt(0, 9, "q - quit")
	s.PrintAt(0, 10, "t - toggle between showing time since resetting statistics or since P_S data was collected")
	s.PrintAt(0, 11, "z - reset statistics")
	s.PrintAt(0, 12, "<tab> or <right arrow> - change display modes between: latency, ops, file I/O, lock and user modes")
	s.PrintAt(0, 13, "<left arrow> - change display modes to the previous screen (see above)")
	s.PrintAt(0, 15, "Press h to return to main screen")
}

// flush changes to screen
func (s *TermboxScreen) Flush() {
	termbox.Flush()
}

// return the current height of the screen
func (s *TermboxScreen) Height() int {
	return s.height
}

// reset the termbox to a clear screen
func (s *TermboxScreen) Initialise() {
	err := termbox.Init()
	if err != nil {
		fmt.Println("Could not start termbox for " + lib.MyName() + ". View ~/." + lib.MyName() + ".log for error messages.")
		log.Printf("Cannot start "+lib.MyName()+", termbox.Init() gave an error:\n%s\n", err)
		os.Exit(1)
	}

	s.Clear()
	s.fg = termbox.ColorDefault
	s.bg = termbox.ColorDefault

	s.SetSize(termbox.Size())
}

// print the characters but don't print them outside the screen
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

// Clear EOL
func (s *TermboxScreen) ClearLine(x int, y int) {
	for i := x; i < s.width; i++ {
		termbox.SetCell(i, y, ' ', termbox.ColorDefault, termbox.ColorDefault)
	}
	s.Flush()
}

// set the screen size
func (s *TermboxScreen) SetSize(width, height int) {
	// if we get bigger then clear out the bottom line
	for x := 0; x < s.width; x++ {
		termbox.SetCell(x, s.height-1, ' ', s.fg, s.bg)
	}
	s.Flush()

	s.width = width
	s.height = height
}

// return the current (width, height) of the screen
func (s *TermboxScreen) Size() (int, int) {
	return s.width, s.height
}

// create a channel for termbox.Events and run a poller to send
// these events to the channel.  Return the channel.
func (s TermboxScreen) TermBoxChan() chan termbox.Event {
	termboxChan := make(chan termbox.Event)
	go func() {
		for {
			termboxChan <- termbox.PollEvent()
		}
	}()
	return termboxChan
}

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

type TermboxAttribute termbox.Attribute

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

	x, y := termbox.Size()
	s.SetSize(x, y)
}

// clear the screen
func (s *TermboxScreen) Clear() {
	termbox.Clear(termbox.ColorWhite, termbox.ColorBlack)
}

func (s *TermboxScreen) Flush() {
	termbox.Flush()
}

func (s *TermboxScreen) SetSize(width, height int) {
	// if we get bigger then clear out the bottom line
	for x := 0; x < s.width; x++ {
		termbox.SetCell(x, s.height-1, ' ', s.fg, s.bg)
	}
	s.Flush()

	s.width = width
	s.height = height
}

func (s *TermboxScreen) Size() (int, int) {
	return s.width, s.height
}

func (s *TermboxScreen) Height() int {
	return s.height
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

func (s *TermboxScreen) DisplayHelp() {
	s.PrintAt(0, 0, lib.MyName()+" version "+version.Version()+" (C) 2014 Simon J Mudd <sjmudd@pobox.com>")

	s.PrintAt(0, 2, "Program to show the top I/O information by accessing information from the")
	s.PrintAt(0, 3, "performance_schema schema. Ideas based on mysql-sys.")

	s.PrintAt(0, 5, "Keys:")
	s.PrintAt(0, 6, "- - reduce the poll interval by 1 second (minimum 1 second)")
	s.PrintAt(0, 7, "+ - increase the poll interval by 1 second")
	s.PrintAt(0, 8, "h - this help screen")
	s.PrintAt(0, 9, "q - quit")
	s.PrintAt(0, 10, "t - toggle between showing time since resetting statistics or since P_S data was collected")
	s.PrintAt(0, 11, "z - reset statistics")
	s.PrintAt(0, 12, "<tab> - change display modes between: latency, ops, file I/O and lock modes")
	s.PrintAt(0, 14, "Press h to return to main screen")
}

func (s *TermboxScreen) Close() {
	termbox.Close()
}

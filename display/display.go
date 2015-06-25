// Package display represents the information we display to the user.
package display

import (
	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/event"
)

// Display is a generic interface to what a display can do
type Display interface {
	// set values which are used later
	SetContext(ctx *context.Context)

	// stuff used by some of the objects
	ClearScreen()
	Close()
	EventChan() chan event.Event
	Resize(width, height int)
	SortNext() // if supported sort on the next column available

	// show verious things
	Display(p GenericData)
	DisplayHelp()
}

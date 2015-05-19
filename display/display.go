// Package to represent the information we display to the user.
package display

import (
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/event"
)

// interface to what a display can do
type Display interface {
	// set values which are used later
	SetHostname(hostname string)
	SetLast(last time.Time)
	SetMySQLVersion(version string)
	SetMyname(name string)
	SetUptime(uptime int)
	SetVersion(version string)
	SetWantRelativeStats(want bool)

	// stuff used by some of the objects
	ClearAndFlush()
	Close()
	EventChan() chan event.Event
	Resize(width, height int)
	Setup(limit int, only_totals bool)

	// show verious things
	Display(p GenericData)
	DisplayHelp()
}

// if there's a better way of doing this do it better ...
func now_hhmmss() string {
	t := time.Now()
	return fmt.Sprintf("%2d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

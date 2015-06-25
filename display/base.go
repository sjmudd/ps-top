// Package display provides information on headers to the views as shown to the user
package display

import (
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/lib"
)

// BaseDisplay holds the structure that is common for all types, somewhere
// to put what's needed in the header.  Make the internal members
// visible without functions for now.
type BaseDisplay struct {
	ctx    *context.Context
}

// SetContet sets the context from the given pointer
func (d *BaseDisplay) SetContext(ctx *context.Context) {
	d.ctx = ctx
}

// return ctx.Last() but protect against nil pointers
func (d BaseDisplay) Last() time.Time {
	if d.ctx == nil {
		return time.Time{}
	}
	return d.ctx.Last()
}

// return ctx.Uptime() but protect against nil pointers
func (d BaseDisplay) Uptime() int {
	if d.ctx == nil {
		return 0
	}
	return d.ctx.Uptime()
}

// MyName returns the application name (binary name)
func (d BaseDisplay) MyName() string {
        return lib.MyName()
}

// HeadingLine returns the heading line as a string
func (d *BaseDisplay) HeadingLine() string {
	var heading string

	headingStart := d.MyName() + " " + d.ctx.Version() + " - " + nowHHMMSS() + " " + d.ctx.Hostname() + " / " + d.ctx.MySQLVersion() + ", up " + fmt.Sprintf("%-16s", lib.Uptime(d.Uptime()))

	if d.ctx.WantRelativeStats() {
		heading = headingStart + " [REL] " + fmt.Sprintf("%.0f seconds", relativeTime(d.Last()))
	} else {
		heading = headingStart + " [ABS]             "
	}
	return heading
}

func relativeTime(last time.Time) float64 {
	now := time.Now()

	d := now.Sub(last)
	return d.Seconds()
}

// if there's a better way of doing this do it better ...
func nowHHMMSS() string {
	t := time.Now()
	return fmt.Sprintf("%2d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

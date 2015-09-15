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
	ctx *context.Context
}

// SetContet sets the context from the given pointer
func (d *BaseDisplay) SetContext(ctx *context.Context) {
	d.ctx = ctx
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
func (d *BaseDisplay) HeadingLine(haveRelativeStats, wantRelativeStats bool, initial, last time.Time) string {
	heading := d.MyName() + " " + d.ctx.Version() + " - " + nowHHMMSS() + " " + d.ctx.Hostname() + " / " + d.ctx.MySQLVersion() + ", up " + fmt.Sprintf("%-16s", lib.Uptime(d.Uptime()))

	if haveRelativeStats {
		if wantRelativeStats {
			heading += " [REL] " + fmt.Sprintf("%.0f seconds", time.Since(initial).Seconds())
		} else {
			heading += " [ABS]             "
		}
	}
	return heading
}

// if there's a better way of doing this do it better ...
func nowHHMMSS() string {
	t := time.Now()
	return fmt.Sprintf("%2d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

package display

import (
	"fmt"
	"time"

	"github.com/sjmudd/ps-top/heading"
	"github.com/sjmudd/ps-top/lib"
)

// Heading is common for all types, somewhere to put what's needed in the header
// make the internal members visible without functions for now.
type Heading struct {
	heading.Heading
}

// HeadingLine returns the heading line as a string
func (d *Heading) HeadingLine() string {
	var heading string

	headingStart := d.Myname + " " + d.Version + " - " + nowHHMMSS() + " " + d.Hostname + " / " + d.MysqlVersion + ", up " + fmt.Sprintf("%-16s", lib.Uptime(d.Uptime))

	if d.WantRelativeStats {
		heading = headingStart + " [REL] " + fmt.Sprintf("%.0f seconds", relativeTime(d.Last))
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

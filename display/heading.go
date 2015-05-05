package display

import (
	"fmt"
	"time"

	"github.com/sjmudd/pstop/lib"
	"github.com/sjmudd/pstop/heading"
)

// common for all types, somewhere to put what's needed in the header
// make the internal members visible without functions for now.
type DisplayHeading struct {
	heading.Heading
}

func (d *DisplayHeading) HeadingLine() string {
	var heading string

	heading_start := d.Myname + " " + d.Version + " - " + now_hhmmss() + " " + d.Hostname + " / " + d.MysqlVersion + ", up " + fmt.Sprintf("%-16s", lib.Uptime(d.Uptime))

	if d.WantRelativeStats {
		heading = heading_start + " [REL] " + fmt.Sprintf("%.0f seconds", rel_time(d.Last))
	} else {
		heading = heading_start + " [ABS]             "
	}
	return heading
}

func rel_time(last time.Time) float64 {
        now := time.Now()

        d := now.Sub(last)
        return d.Seconds()
}

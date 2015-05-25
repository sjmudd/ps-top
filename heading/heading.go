// Package heading provides information on headers to the views as shown to the user
package heading

import (
	"time"
)

// Heading holds the structure that is common for all types, somewhere
// to put what's needed in the header.  Make the internal members
// visible without functions for now.
type Heading struct {
	Hostname          string
	Myname            string
	MysqlVersion      string
	Last              time.Time
	Uptime            int
	Version           string
	WantRelativeStats bool
}

// SetHostname records the hostname of the server
func (d *Heading) SetHostname(hostname string) {
	d.Hostname = hostname
}

// SetLast records the Last time the data was collected
func (d *Heading) SetLast(last time.Time) {
	d.Last = last
}

// SetMyname records my program name
func (d *Heading) SetMyname(myname string) {
	d.Myname = myname
}

// SetVersion records the program version
func (d *Heading) SetVersion(version string) {
	d.Version = version
}

// SetMySQLVersion records the mysql version of the host
func (d *Heading) SetMySQLVersion(mysqlVersion string) {
	d.MysqlVersion = mysqlVersion
}

// SetWantRelativeStats records if we want to look at relative or absolute data
func (d *Heading) SetWantRelativeStats(wantRelativeStats bool) {
	d.WantRelativeStats = wantRelativeStats
}

// SetUptime records the mysql server uptime
func (d *Heading) SetUptime(uptime int) {
	d.Uptime = uptime
}

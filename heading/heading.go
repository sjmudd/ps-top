package heading

import (
	"time"
)

// common for all types, somewhere to put what's needed in the header
// make the internal members visible without functions for now.
type Heading struct {
	Hostname          string
	Myname            string
	MysqlVersion      string
	Last              time.Time
	Uptime            int
	Version           string
	WantRelativeStats bool
}

func (d *Heading) SetHostname(hostname string) {
	d.Hostname = hostname
}

func (d *Heading) SetLast(last time.Time) {
	d.Last = last
}

func (d *Heading) SetMyname(myname string) {
	d.Myname = myname
}

func (d *Heading) SetVersion(version string) {
	d.Version = version
}

func (d *Heading) SetMySQLVersion(mysql_version string) {
	d.MysqlVersion = mysql_version
}

func (d *Heading) SetWantRelativeStats(want_relative_stats bool) {
	d.WantRelativeStats = want_relative_stats
}

func (d *Heading) SetUptime(uptime int) {
	d.Uptime = uptime
}

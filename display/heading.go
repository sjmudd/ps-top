package display

// common for all types, somewhere to put what's needed in the header
// make the internal members visible without functions for now.
type DisplayHeading struct {
	Hostname          string
	Myname            string
	MysqlVersion      string
	Uptime            int
	Version           string
	WantRelativeStats bool
}

func (d *DisplayHeading) SetMyname(myname string) {
	d.Myname = myname
}

func (d *DisplayHeading) SetVersion(version string) {
	d.Version = version
}

func (d *DisplayHeading) SetHostname(hostname string) {
	d.Hostname = hostname
}

func (d *DisplayHeading) SetMySQLVersion(mysql_version string) {
	d.MysqlVersion = mysql_version
}

func (d *DisplayHeading) SetUptime(uptime int) {
	d.Uptime = uptime
}

func (d *DisplayHeading) SetWantRelativeStats(want_relative_stats bool) {
	d.WantRelativeStats = want_relative_stats
}

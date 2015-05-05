package display

import (
	"fmt"
	"time"

	"github.com/sjmudd/pstop/event"
	"github.com/sjmudd/pstop/i_s/processlist"
	"github.com/sjmudd/pstop/p_s/ps_table"
	tiwsbt "github.com/sjmudd/pstop/p_s/table_io_waits_summary_by_table"
)

// interface to what a display can do
type Display interface {
	// set values which are used later
	SetHostname(hostname string)
	SetLast(last time.Time)
	SetLimit(limit int)
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
	Setup()

	// show verious things
	DisplayIO(p ps_table.Tabler)
	DisplayLocks(p ps_table.Tabler)
	DisplayMutex(p ps_table.Tabler)
	DisplayOpsOrLatency(tiwsbt tiwsbt.Object)
	DisplayStages(p ps_table.Tabler)
	DisplayUsers(users processlist.Object)
	DisplayHelp()
}

// if there's a better way of doing this do it better ...
func now_hhmmss() string {
	t := time.Now()
	return fmt.Sprintf("%2d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

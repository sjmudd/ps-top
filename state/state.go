// lib - library routines for pstop.
//
// this file contains the library routines related to the stored state in pstop.
package state

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sjmudd/pstop/lib"
	fsbi "github.com/sjmudd/pstop/performance_schema/file_summary_by_instance"
	"github.com/sjmudd/pstop/performance_schema/ps_table"
	tiwsbt "github.com/sjmudd/pstop/performance_schema/table_io_waits_summary_by_table"
	tlwsbt "github.com/sjmudd/pstop/performance_schema/table_lock_waits_summary_by_table"
	"github.com/sjmudd/pstop/screen"
	"github.com/sjmudd/pstop/version"
)

// what information to show
type Show int

const (
	showLatency = iota
	showOps     = iota
	showIO      = iota
	showLocks   = iota
)

type State struct {
	datadir             string
	dbh                 *sql.DB
	help                bool
	hostname            string
	fsbi                ps_table.Tabler // ufsbi.File_summary_by_instance
	tiwsbt              tiwsbt.Table_io_waits_summary_by_table
	tlwsbt              ps_table.Tabler // tlwsbt.Table_lock_waits_summary_by_table
	screen              screen.TermboxScreen
	show                Show
	mysql_version       string
	want_relative_stats bool
}

func (state *State) Setup(dbh *sql.DB) {
	state.dbh = dbh

	state.screen.Initialise()

	_, variables := lib.SelectAllGlobalVariablesByVariableName(state.dbh)
	// setup to their initial types/values
	state.fsbi = fsbi.NewFileSummaryByInstance(variables)
	state.tlwsbt = new(tlwsbt.Table_lock_waits_summary_by_table)

	state.want_relative_stats = true // we show info from the point we start collecting data
	state.fsbi.SetWantRelativeStats(state.want_relative_stats)
	state.fsbi.SetNow()
	state.tlwsbt.SetWantRelativeStats(state.want_relative_stats)
	state.tlwsbt.SetNow()
	state.tiwsbt.SetWantRelativeStats(state.want_relative_stats)
	state.tiwsbt.SetNow()

	state.ResetDBStatistics()

	state.SetHelp(false)
	state.show = showLatency
	state.tiwsbt.SetWantsLatency(true)

	_, hostname := lib.SelectGlobalVariableByVariableName(state.dbh, "HOSTNAME")
	_, mysql_version := lib.SelectGlobalVariableByVariableName(state.dbh, "VERSION")
	_, datadir := lib.SelectGlobalVariableByVariableName(state.dbh, "DATADIR")
	state.SetHostname(hostname)
	state.SetMySQLVersion(mysql_version)
	state.SetDatadir(datadir)
}

// do a fresh collection of data and then update the initial values based on that.
func (state *State) ResetDBStatistics() {
	state.CollectAll()
	state.SyncReferenceValues()
}

func (state *State) SyncReferenceValues() {
	start := time.Now()
	state.fsbi.SyncReferenceValues()
	state.tlwsbt.SyncReferenceValues()
	state.tiwsbt.SyncReferenceValues()
	lib.Logger.Println("state.SyncReferenceValues() took", time.Duration(time.Since(start)).String())
}

// collect all initial values on startup / reset
func (state *State) CollectAll() {
	state.fsbi.Collect(state.dbh)
	state.tlwsbt.Collect(state.dbh)
	state.tiwsbt.Collect(state.dbh)
}

// Only collect the data we are looking at.
func (state *State) Collect() {
	start := time.Now()

	switch state.show {
	case showLatency, showOps:
		state.tiwsbt.Collect(state.dbh)
	case showIO:
		state.fsbi.Collect(state.dbh)
	case showLocks:
		state.tlwsbt.Collect(state.dbh)
	}
	lib.Logger.Println("state.Collect() took", time.Duration(time.Since(start)).String())
}

func (state State) MySQLVersion() string {
	return state.mysql_version
}

func (state State) Datadir() string {
	return state.datadir
}

func (state *State) SetHelp(newHelp bool) {
	state.help = newHelp

	state.screen.Clear()
	state.screen.Flush()
}

func (state *State) SetDatadir(datadir string) {
	state.datadir = datadir
}

func (state *State) SetMySQLVersion(mysql_version string) {
	state.mysql_version = mysql_version
}

func (state *State) SetHostname(hostname string) {
	state.hostname = hostname
}

func (state State) Help() bool {
	return state.help
}

// display the output according to the mode we are in
func (state *State) Display() {
	if state.help {
		state.screen.DisplayHelp()
	} else {
		state.displayHeading()
		switch state.show {
		case showLatency, showOps:
			state.displayOpsOrLatency()
		case showIO:
			state.displayIO()
		case showLocks:
			state.displayLocks()
		}
	}
}

// change to the next display mode
func (state *State) DisplayNext() {
	if state.show == showLocks {
		state.show = showLatency
	} else {
		state.show++
	}
	// this needs to be done more cleanly
	if state.show == showLatency {
		state.tiwsbt.SetWantsLatency(true)
	}
	if state.show == showOps {
		state.tiwsbt.SetWantsLatency(false)
	}
	state.screen.Clear()
	state.screen.Flush()
}

func (state State) displayHeading() {
	state.displayLine0()
	state.displayDescription()
}

func (state State) displayLine0() {
	_, uptime := lib.SelectGlobalStatusByVariableName(state.dbh, "UPTIME")
	top_line := lib.MyName() + " " + version.Version() + " - " + now_hhmmss() + " " + state.hostname + " / " + state.mysql_version + ", up " + fmt.Sprintf("%-16s", lib.Uptime(uptime))
	if state.want_relative_stats {
		now := time.Now()

		var initial time.Time

		switch state.show {
		case showLatency, showOps:
			initial = state.tiwsbt.Last()
		case showIO:
			initial = state.fsbi.Last()
		case showLocks:
			initial = state.tlwsbt.Last()
		default:
			initial = time.Now() // THIS IS WRONG !!!
		}

		d := now.Sub(initial)

		top_line = top_line + " [REL] " + fmt.Sprintf("%.0f seconds", d.Seconds())
	} else {
		top_line = top_line + " [ABS]             "
	}
	state.screen.PrintAt(0, 0, top_line)
}

func (state State) displayDescription() {
	description := "UNKNOWN"

	switch state.show {
	case showLatency, showOps:
		description = state.tiwsbt.Description()
	case showIO:
		description = state.fsbi.Description()
	case showLocks:
		description = state.tlwsbt.Description()
	}

	state.screen.PrintAt(0, 1, description)
}

func (state *State) displayOpsOrLatency() {
	state.screen.PrintAt(0, 2, state.tiwsbt.Headings())

	max_rows := state.screen.Height() - 3
	row_content := state.tiwsbt.RowContent(max_rows)

	// print out rows
	for k := range row_content {
		y := 3 + k
		state.screen.PrintAt(0, y, row_content[k])
	}
	// print out empty rows
	for k := len(row_content); k < (state.screen.Height() - 3); k++ {
		y := 3 + k
		if y < state.screen.Height()-1 {
			state.screen.PrintAt(0, y, state.tiwsbt.EmptyRowContent())
		}
	}

	// print out the totals at the bottom
	state.screen.PrintAt(0, state.screen.Height()-1, state.tiwsbt.TotalRowContent())
}

// show actual I/O latency values
func (state State) displayIO() {
	state.screen.PrintAt(0, 2, state.fsbi.Headings())

	// print out the data
	max_rows := state.screen.Height() - 3
	row_content := state.fsbi.RowContent(max_rows)

	// print out rows
	for k := range row_content {
		y := 3 + k
		state.screen.PrintAt(0, y, row_content[k])
	}
	// print out empty rows
	for k := len(row_content); k < (state.screen.Height() - 3); k++ {
		y := 3 + k
		if y < state.screen.Height()-1 {
			state.screen.PrintAt(0, y, state.fsbi.EmptyRowContent())
		}
	}

	// print out the totals at the bottom
	state.screen.PrintAt(0, state.screen.Height()-1, state.fsbi.TotalRowContent())
}

func (state *State) displayLocks() {
	state.screen.PrintAt(0, 2, state.tlwsbt.Headings())

	// print out the data
	max_rows := state.screen.Height() - 3
	row_content := state.tlwsbt.RowContent(max_rows)

	// print out rows
	for k := range row_content {
		y := 3 + k
		state.screen.PrintAt(0, y, row_content[k])
	}
	// print out empty rows
	for k := len(row_content); k < (state.screen.Height() - 3); k++ {
		y := 3 + k
		if y < state.screen.Height()-1 {
			state.screen.PrintAt(0, y, state.tlwsbt.EmptyRowContent())
		}
	}

	// print out the totals at the bottom
	state.screen.PrintAt(0, state.screen.Height()-1, state.tlwsbt.TotalRowContent())
}

// do we want to show all p_s data?
func (state State) WantRelativeStats() bool {
	return state.want_relative_stats
}

// set if we want data from when we started/reset stats.
func (state *State) SetWantRelativeStats(want_relative_stats bool) {
	state.want_relative_stats = want_relative_stats

	state.fsbi.SetWantRelativeStats(want_relative_stats)
	state.tlwsbt.SetWantRelativeStats(state.want_relative_stats)
	state.tiwsbt.SetWantRelativeStats(state.want_relative_stats)
}

// if there's a better way of doing this do it better ...
func now_hhmmss() string {
	t := time.Now()
	return fmt.Sprintf("%2d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

// record the latest screen size
func (state *State) ScreenSetSize(width, height int) {
	state.screen.SetSize(width, height)
}

// clean up screen and disconnect database
func (state *State) Cleanup() {
	state.screen.Close()
	if state.dbh != nil {
		_ = state.dbh.Close()
	}
}

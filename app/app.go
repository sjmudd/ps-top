// app - pstop application package
//
// This file contains the library routines related to running the app.
package app

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/nsf/termbox-go"

	"github.com/sjmudd/pstop/i_s/processlist"
	"github.com/sjmudd/pstop/lib"
	essgben "github.com/sjmudd/pstop/p_s/events_stages_summary_global_by_event_name"
	ewsgben "github.com/sjmudd/pstop/p_s/events_waits_summary_global_by_event_name"
	fsbi "github.com/sjmudd/pstop/p_s/file_summary_by_instance"
	"github.com/sjmudd/pstop/p_s/ps_table"
	"github.com/sjmudd/pstop/p_s/setup_instruments"
	tiwsbt "github.com/sjmudd/pstop/p_s/table_io_waits_summary_by_table"
	tlwsbt "github.com/sjmudd/pstop/p_s/table_lock_waits_summary_by_table"
	"github.com/sjmudd/pstop/screen"
	"github.com/sjmudd/pstop/version"
	"github.com/sjmudd/pstop/wait_info"
)

// what information to show
type Show int

const (
	showLatency = iota
	showOps     = iota
	showIO      = iota
	showLocks   = iota
	showUsers   = iota
	showMutex   = iota
	showStages  = iota
)

var (
	re_valid_version = regexp.MustCompile(`^(5\.[67]\.|10\.[01])`)
)

type App struct {
	count               int
	done                chan struct{}
	sigChan             chan os.Signal
	wi                  wait_info.WaitInfo
	finished            bool
	stdout              bool
	dbh                 *sql.DB
	help                bool
	hostname            string
	fsbi                ps_table.Tabler // ufsbi.File_summary_by_instance
	tiwsbt              tiwsbt.Object
	tlwsbt              ps_table.Tabler // tlwsbt.Table_lock_waits_summary_by_table
	ewsgben             ps_table.Tabler // ewsgben.Events_waits_summary_global_by_event_name
	essgben             ps_table.Tabler // essgben.Events_stages_summary_global_by_event_name
	users               processlist.Object
	screen              screen.TermboxScreen
	show                Show
	mysql_version       string
	want_relative_stats bool
	wait_info.WaitInfo  // embedded
	setup_instruments   setup_instruments.SetupInstruments
}

func (app *App) Setup(dbh *sql.DB, interval int, count int, stdout bool) {
	lib.Logger.Println("app.Setup()")
	app.dbh = dbh

	if err := app.validate_mysql_version(); err != nil {
		log.Fatal(err)
	}

	app.count = count
	app.finished = false
	app.stdout = stdout

	app.screen.Initialise()
	app.setup_instruments = setup_instruments.NewSetupInstruments(dbh)
	app.setup_instruments.EnableMonitoring()

	app.wi.SetWaitInterval( time.Second * time.Duration( interval ) )

	_, variables := lib.SelectAllGlobalVariablesByVariableName(app.dbh)
	// setup to their initial types/values
	app.fsbi = fsbi.NewFileSummaryByInstance(variables)
	app.tlwsbt = new(tlwsbt.Object)
	app.ewsgben = new(ewsgben.Object)
	app.essgben = new(essgben.Object)

	app.want_relative_stats = true // we show info from the point we start collecting data
	app.fsbi.SetWantRelativeStats(app.want_relative_stats)
	app.fsbi.SetNow()
	app.tlwsbt.SetWantRelativeStats(app.want_relative_stats)
	app.tlwsbt.SetNow()
	app.tiwsbt.SetWantRelativeStats(app.want_relative_stats)
	app.tiwsbt.SetNow()
	app.users.SetWantRelativeStats(app.want_relative_stats) // ignored
	app.users.SetNow()                                      // ignored
	app.essgben.SetWantRelativeStats(app.want_relative_stats)
	app.essgben.SetNow()
	app.ewsgben.SetWantRelativeStats(app.want_relative_stats) // ignored
	app.ewsgben.SetNow()                                      // ignored

	app.ResetDBStatistics()

	app.SetHelp(false)
	app.show = showLatency
	app.tiwsbt.SetWantsLatency(true)

	// get short name (to save space)
	_, hostname := lib.SelectGlobalVariableByVariableName(app.dbh, "HOSTNAME")
	if index := strings.Index(hostname, "."); index >= 0 {
		hostname = hostname[0:index]
	}
	_, mysql_version := lib.SelectGlobalVariableByVariableName(app.dbh, "VERSION")
	app.SetHostname(hostname)
	app.SetMySQLVersion(mysql_version)
}

// have we finished ?
func (app App) Finished() bool {
	return app.finished
}

// indicate we have finished
func (app *App) SetFinished() {
	app.finished = true
}

// do a fresh collection of data and then update the initial values based on that.
func (app *App) ResetDBStatistics() {
	app.CollectAll()
	app.SyncReferenceValues()
}

func (app *App) SyncReferenceValues() {
	start := time.Now()
	app.fsbi.SyncReferenceValues()
	app.tlwsbt.SyncReferenceValues()
	app.tiwsbt.SyncReferenceValues()
	app.essgben.SyncReferenceValues()
	lib.Logger.Println("app.SyncReferenceValues() took", time.Duration(time.Since(start)).String())
}

// collect all initial values on startup / reset
func (app *App) CollectAll() {
	app.fsbi.Collect(app.dbh)
	app.tlwsbt.Collect(app.dbh)
	app.tiwsbt.Collect(app.dbh)
}

// Only collect the data we are looking at.
func (app *App) Collect() {
	start := time.Now()

	switch app.show {
	case showLatency, showOps:
		app.tiwsbt.Collect(app.dbh)
	case showIO:
		app.fsbi.Collect(app.dbh)
	case showLocks:
		app.tlwsbt.Collect(app.dbh)
	case showUsers:
		app.users.Collect(app.dbh)
	case showMutex:
		app.ewsgben.Collect(app.dbh)
	case showStages:
		app.essgben.Collect(app.dbh)
	}
	app.wi.CollectedNow()
	lib.Logger.Println("app.Collect() took", time.Duration(time.Since(start)).String())
}

func (app App) MySQLVersion() string {
	return app.mysql_version
}

func (app *App) SetHelp(newHelp bool) {
	app.help = newHelp

	app.screen.Clear()
	app.screen.Flush()
}

func (app *App) SetMySQLVersion(mysql_version string) {
	app.mysql_version = mysql_version
}

func (app *App) SetHostname(hostname string) {
	lib.Logger.Println("app.SetHostname(",hostname,")")
	app.hostname = hostname
}

func (app App) Help() bool {
	return app.help
}

// apps go: showLatency -> showOps -> showIO -> showLocks -> showUsers -> showMutex -> showStages

// display the output according to the mode we are in
func (app *App) Display() {
	if app.help {
		app.screen.DisplayHelp()
	} else {
		app.displayHeading()
		switch app.show {
		case showLatency, showOps:
			app.displayOpsOrLatency()
		case showIO:
			app.displayIO()
		case showLocks:
			app.displayLocks()
		case showUsers:
			app.displayUsers()
		case showMutex:
			app.displayMutex()
		case showStages:
			app.displayStages()
		}
	}
}

// fix_latency_setting() ensures the SetWantsLatency() value is
// correct. This needs to be done more cleanly.
func (app *App) fix_latency_setting() {
	if app.show == showLatency {
		app.tiwsbt.SetWantsLatency(true)
	}
	if app.show == showOps {
		app.tiwsbt.SetWantsLatency(false)
	}
}

// change to the previous display mode
func (app *App) DisplayPrevious() {
	if app.show == showLatency {
		app.show = showStages
	} else {
		app.show--
	}
	app.fix_latency_setting()
	app.screen.Clear()
	app.screen.Flush()
}

// change to the next display mode
func (app *App) DisplayNext() {
	if app.show == showStages {
		app.show = showLatency
	} else {
		app.show++
	}
	app.fix_latency_setting()
	app.screen.Clear()
	app.screen.Flush()
}

func (app App) displayHeading() {
	app.displayLine0()
	app.displayDescription()
}

func (app App) displayLine0() {
	_, uptime := lib.SelectGlobalStatusByVariableName(app.dbh, "UPTIME")
	top_line := lib.MyName() + " " + version.Version() + " - " + now_hhmmss() + " " + app.hostname + " / " + app.mysql_version + ", up " + fmt.Sprintf("%-16s", lib.Uptime(uptime))
	if app.want_relative_stats {
		now := time.Now()

		var initial time.Time

		switch app.show {
		case showLatency, showOps:
			initial = app.tiwsbt.Last()
		case showIO:
			initial = app.fsbi.Last()
		case showLocks:
			initial = app.tlwsbt.Last()
		case showUsers:
			initial = app.users.Last()
		case showStages:
			initial = app.essgben.Last()
		case showMutex:
			initial = app.ewsgben.Last()
		default:
			// should not get here !
		}

		d := now.Sub(initial)

		top_line = top_line + " [REL] " + fmt.Sprintf("%.0f seconds", d.Seconds())
	} else {
		top_line = top_line + " [ABS]             "
	}
	app.screen.PrintAt(0, 0, top_line)
}

func (app App) displayDescription() {
	description := "UNKNOWN"

	switch app.show {
	case showLatency, showOps:
		description = app.tiwsbt.Description()
	case showIO:
		description = app.fsbi.Description()
	case showLocks:
		description = app.tlwsbt.Description()
	case showUsers:
		description = app.users.Description()
	case showMutex:
		description = app.ewsgben.Description()
	case showStages:
		description = app.essgben.Description()
	}

	app.screen.PrintAt(0, 1, description)
}

func (app *App) displayOpsOrLatency() {
	app.screen.BoldPrintAt(0, 2, app.tiwsbt.Headings())

	max_rows := app.screen.Height() - 3
	last_row := app.screen.Height() - 1
	row_content := app.tiwsbt.RowContent(max_rows)

	// print out rows
	for k := range row_content {
		y := 3 + k
		app.screen.PrintAt(0, y, row_content[k])
		app.screen.ClearLine(len(row_content[k]), y)
	}
	// print out empty rows
	for k := len(row_content); k < max_rows; k++ {
		y := 3 + k
		if y < max_rows-1 {
			app.screen.PrintAt(0, y, app.tiwsbt.EmptyRowContent())
		}
	}

	// print out the totals at the bottom
	total := app.tiwsbt.TotalRowContent()
	app.screen.BoldPrintAt(0, last_row, total)
	app.screen.ClearLine(len(total), last_row)
}

// show actual I/O latency values
func (app App) displayIO() {
	app.screen.BoldPrintAt(0, 2, app.fsbi.Headings())

	// print out the data
	max_rows := app.screen.Height() - 4
	last_row := app.screen.Height() - 1
	row_content := app.fsbi.RowContent(max_rows)

	// print out rows
	for k := range row_content {
		y := 3 + k
		app.screen.PrintAt(0, y, row_content[k])
		app.screen.ClearLine(len(row_content[k]), y)
	}
	// print out empty rows
	for k := len(row_content); k < max_rows; k++ {
		y := 3 + k
		if y < last_row {
			app.screen.PrintAt(0, y, app.fsbi.EmptyRowContent())
		}
	}

	// print out the totals at the bottom
	total := app.fsbi.TotalRowContent()
	app.screen.BoldPrintAt(0, last_row, total)
	app.screen.ClearLine(len(total), last_row)
}

func (app *App) displayLocks() {
	app.screen.BoldPrintAt(0, 2, app.tlwsbt.Headings())

	// print out the data
	max_rows := app.screen.Height() - 4
	last_row := app.screen.Height() - 1
	row_content := app.tlwsbt.RowContent(max_rows)

	// print out rows
	for k := range row_content {
		y := 3 + k
		app.screen.PrintAt(0, y, row_content[k])
		app.screen.ClearLine(len(row_content[k]), y)
	}
	// print out empty rows
	for k := len(row_content); k < (app.screen.Height() - 3); k++ {
		y := 3 + k
		if y < last_row {
			app.screen.PrintAt(0, y, app.tlwsbt.EmptyRowContent())
		}
	}

	// print out the totals at the bottom
	total := app.tlwsbt.TotalRowContent()
	app.screen.BoldPrintAt(0, last_row, total)
	app.screen.ClearLine(len(total), last_row)
}

func (app *App) displayUsers() {
	app.screen.BoldPrintAt(0, 2, app.users.Headings())

	// print out the data
	max_rows := app.screen.Height() - 4
	last_row := app.screen.Height() - 1
	row_content := app.users.RowContent(max_rows)

	// print out rows
	for k := range row_content {
		y := 3 + k
		app.screen.PrintAt(0, y, row_content[k])
		app.screen.ClearLine(len(row_content[k]), y)
	}
	// print out empty rows
	for k := len(row_content); k < max_rows; k++ {
		y := 3 + k
		if y < last_row {
			app.screen.PrintAt(0, y, app.users.EmptyRowContent())
		}
	}

	// print out the totals at the bottom
	total := app.users.TotalRowContent()
	app.screen.BoldPrintAt(0, last_row, total)
	app.screen.ClearLine(len(total), last_row)
}

func (app *App) displayMutex() {
	app.screen.BoldPrintAt(0, 2, app.ewsgben.Headings())

	// print out the data
	max_rows := app.screen.Height() - 4
	last_row := app.screen.Height() - 1
	row_content := app.ewsgben.RowContent(max_rows)

	// print out rows
	for k := range row_content {
		y := 3 + k
		app.screen.PrintAt(0, y, row_content[k])
		app.screen.ClearLine(len(row_content[k]), y)
	}
	// print out empty rows
	for k := len(row_content); k < max_rows; k++ {
		y := 3 + k
		if y < last_row {
			app.screen.PrintAt(0, y, app.ewsgben.EmptyRowContent())
		}
	}

	// print out the totals at the bottom
	total := app.ewsgben.TotalRowContent()
	app.screen.BoldPrintAt(0, last_row, total)
	app.screen.ClearLine(len(total), last_row)
}

func (app *App) displayStages() {
	app.screen.BoldPrintAt(0, 2, app.essgben.Headings())

	// print out the data
	max_rows := app.screen.Height() - 4
	last_row := app.screen.Height() - 1
	row_content := app.essgben.RowContent(max_rows)

	// print out rows
	for k := range row_content {
		y := 3 + k
		app.screen.PrintAt(0, y, row_content[k])
		app.screen.ClearLine(len(row_content[k]), y)
	}
	// print out empty rows
	for k := len(row_content); k < max_rows; k++ {
		y := 3 + k
		if y < last_row {
			app.screen.PrintAt(0, y, app.essgben.EmptyRowContent())
		}
	}

	// print out the totals at the bottom
	total := app.essgben.TotalRowContent()
	app.screen.BoldPrintAt(0, last_row, total)
	app.screen.ClearLine(len(total), last_row)
}

// do we want to show all p_s data?
func (app App) WantRelativeStats() bool {
	return app.want_relative_stats
}

// set if we want data from when we started/reset stats.
func (app *App) SetWantRelativeStats(want_relative_stats bool) {
	app.want_relative_stats = want_relative_stats

	app.fsbi.SetWantRelativeStats(want_relative_stats)
	app.tlwsbt.SetWantRelativeStats(app.want_relative_stats)
	app.tiwsbt.SetWantRelativeStats(app.want_relative_stats)
	app.ewsgben.SetWantRelativeStats(app.want_relative_stats)
	app.essgben.SetWantRelativeStats(app.want_relative_stats)
}

// if there's a better way of doing this do it better ...
func now_hhmmss() string {
	t := time.Now()
	return fmt.Sprintf("%2d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

// record the latest screen size
func (app *App) ScreenSetSize(width, height int) {
	app.screen.SetSize(width, height)
}

// clean up screen and disconnect database
func (app *App) Cleanup() {
	app.screen.Close()
	if app.dbh != nil {
		app.setup_instruments.RestoreConfiguration()
		_ = app.dbh.Close()
	}
}

// get into a run loop
func (app *App) Run() {
	lib.Logger.Println("app.Run()")
	app.done = make(chan struct{})
	defer close(app.done)

	app.sigChan = make(chan os.Signal, 1)
	signal.Notify(app.sigChan, syscall.SIGINT, syscall.SIGTERM)

	termboxChan := app.screen.TermBoxChan()

	for !app.Finished() {
		select {
		case <-app.done:
			fmt.Println("app.done(): exiting")
			app.SetFinished()
		case sig := <-app.sigChan:
			fmt.Println("Caught a signal", sig)
			app.done <- struct{}{}
		case <-app.wi.WaitNextPeriod():
			app.Collect()
			app.Display()
		case event := <-termboxChan:
			// switch on event type
			switch event.Type {
			case termbox.EventKey: // actions depend on key
				switch event.Key {
				case termbox.KeyCtrlZ, termbox.KeyCtrlC, termbox.KeyEsc:
					app.SetFinished()
				case termbox.KeyArrowLeft: // left arrow change to previous display mode
					app.DisplayPrevious()
					app.Display()
				case termbox.KeyTab, termbox.KeyArrowRight: // tab or right arrow - change to next display mode
					app.DisplayNext()
					app.Display()
				}
				switch event.Ch {
				case '-': // decrease the interval if > 1
					if app.wi.WaitInterval() > time.Second {
						app.wi.SetWaitInterval(app.wi.WaitInterval() - time.Second)
					}
				case '+': // increase interval by creating a new ticker
					app.wi.SetWaitInterval(app.wi.WaitInterval() + time.Second)
				case 'h', '?': // help
					app.SetHelp(!app.Help())
				case 'q': // quit
					app.SetFinished()
				case 't': // toggle between absolute/relative statistics
					app.SetWantRelativeStats(!app.WantRelativeStats())
					app.Display()
				case 'z': // reset the statistics to now by taking a query of current values
					app.ResetDBStatistics()
					app.Display()
				}
			case termbox.EventResize: // set sizes
				app.ScreenSetSize(event.Width, event.Height)
				app.Display()
			case termbox.EventError: // quit
				log.Fatalf("Quitting because of termbox error: \n%s\n", event.Err)
			}
		}
		// provide a hook to stop the application if the counter goes down to zero
		if app.stdout && app.count > 0 {
			app.count--
			if app.count == 0 {
				app.SetFinished()
			}
		}
	}
}

// pstop requires MySQL 5.6+ or MariaDB 10.0+. Check the version
// rather than giving an error message if the requires P_S tables can't
// be found.
func (app *App) validate_mysql_version() error {
	var tables = [...]string{
		"performance_schema.events_stages_summary_global_by_event_name",
		"performance_schema.events_waits_summary_global_by_event_name",
		"performance_schema.file_summary_by_instance",
		"performance_schema.table_io_waits_summary_by_table",
		"performance_schema.table_lock_waits_summary_by_table",
	}

	lib.Logger.Println("validate_mysql_version()")

	lib.Logger.Println("- Getting MySQL version")
	err, mysql_version := lib.SelectGlobalVariableByVariableName(app.dbh, "VERSION")
	if err != nil {
		return err
	}
	lib.Logger.Println("- mysql_version: '" + mysql_version + "'")

	if !re_valid_version.MatchString(mysql_version) {
		return errors.New(lib.MyName() + " does not work with MySQL version " + mysql_version)
	}
	lib.Logger.Println("OK: MySQL version is valid, continuing")

	lib.Logger.Println("Checking access to required tables:")
	for i := range tables {
		if err := lib.CheckTableAccess(app.dbh, tables[i]); err == nil {
			lib.Logger.Println("OK: " + tables[i] + " found")
		} else {
			return err
		}
	}
	lib.Logger.Println("OK: all table checks passed")

	return nil
}

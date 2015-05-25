// Package app is the "runtime" for the ps-top / ps-stats application packages
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

	"github.com/sjmudd/ps-top/display"
	"github.com/sjmudd/ps-top/event"
	"github.com/sjmudd/ps-top/i_s/processlist"
	"github.com/sjmudd/ps-top/lib"
	essgben "github.com/sjmudd/ps-top/p_s/events_stages_summary_global_by_event_name"
	ewsgben "github.com/sjmudd/ps-top/p_s/events_waits_summary_global_by_event_name"
	fsbi "github.com/sjmudd/ps-top/p_s/file_summary_by_instance"
	"github.com/sjmudd/ps-top/p_s/ps_table"
	"github.com/sjmudd/ps-top/p_s/setup_instruments"
	tiwsbt "github.com/sjmudd/ps-top/p_s/table_io_waits_summary_by_table"
	tlwsbt "github.com/sjmudd/ps-top/p_s/table_lock_waits_summary_by_table"
	"github.com/sjmudd/ps-top/version"
	"github.com/sjmudd/ps-top/view"
	"github.com/sjmudd/ps-top/wait_info"
)

// App holds the data needed by an application
type App struct {
	count              int
	display            display.Display
	done               chan struct{}
	sigChan            chan os.Signal
	wi                 wait_info.WaitInfo
	finished           bool
	stdout             bool
	dbh                *sql.DB
	help               bool
	hostname           string
	fsbi               ps_table.Tabler // ufsbi.File_summary_by_instance
	tiwsbt             tiwsbt.Object
	tlwsbt             ps_table.Tabler // tlwsbt.Table_lock_waits_summary_by_table
	ewsgben            ps_table.Tabler // ewsgben.Events_waits_summary_global_by_event_name
	essgben            ps_table.Tabler // essgben.Events_stages_summary_global_by_event_name
	users              processlist.Object
	view               view.View
	mysqlVersion       string
	wantRelativeStats  bool
	wait_info.WaitInfo // embedded
	setupInstruments   setup_instruments.SetupInstruments
}

// Setup initialises the application given various parameters.
func (app *App) Setup(dbh *sql.DB, interval int, count int, stdout bool, limit int, defaultView string, onlyTotals bool) {
	lib.Logger.Println("app.Setup()")

	app.count = count
	app.dbh = dbh
	app.finished = false
	app.stdout = stdout

	if stdout {
		app.display = new(display.StdoutDisplay)
	} else {
		app.display = new(display.ScreenDisplay)
	}
	app.display.Setup(limit, onlyTotals)
	app.SetHelp(false)
	app.view.SetByName(defaultView) // if empty will use the default

	if err := app.validateMysqlVersion(); err != nil {
		log.Fatal(err)
	}

	app.setupInstruments = setup_instruments.NewSetupInstruments(dbh)
	app.setupInstruments.EnableMonitoring()

	app.wi.SetWaitInterval(time.Second * time.Duration(interval))

	variables, _ := lib.SelectAllGlobalVariablesByVariableName(app.dbh)
	// setup to their initial types/values
	app.fsbi = fsbi.NewFileSummaryByInstance(variables)
	app.tlwsbt = new(tlwsbt.Object)
	app.ewsgben = new(ewsgben.Object)
	app.essgben = new(essgben.Object)

	app.wantRelativeStats = true // we show info from the point we start collecting data
	app.fsbi.SetWantRelativeStats(app.wantRelativeStats)
	app.fsbi.SetCollected()
	app.tlwsbt.SetWantRelativeStats(app.wantRelativeStats)
	app.tlwsbt.SetCollected()
	app.tiwsbt.SetWantRelativeStats(app.wantRelativeStats)
	app.tiwsbt.SetCollected()
	app.users.SetWantRelativeStats(app.wantRelativeStats) // ignored
	app.users.SetCollected()                              // ignored
	app.essgben.SetWantRelativeStats(app.wantRelativeStats)
	app.essgben.SetCollected()
	app.ewsgben.SetWantRelativeStats(app.wantRelativeStats) // ignored
	app.ewsgben.SetCollected()                              // ignored

	app.fixLatencySetting() // adjust to see ops/latency

	app.resetDBStatistics()

	// get short name (to save space)
	hostname, _ := lib.SelectGlobalVariableByVariableName(app.dbh, "HOSTNAME")
	if index := strings.Index(hostname, "."); index >= 0 {
		hostname = hostname[0:index]
	}
	mysqlVersion, _ := lib.SelectGlobalVariableByVariableName(app.dbh, "VERSION")

	// setup display with base data
	app.display.SetHostname(hostname)
	app.display.SetMySQLVersion(mysqlVersion)
	app.display.SetVersion(version.Version())
	app.display.SetMyname(lib.MyName())
	app.display.SetWantRelativeStats(app.wantRelativeStats)
}

// Finished tells us if we have finished
func (app App) Finished() bool {
	return app.finished
}

// CollectAll collects all the stats together in one go
func (app *App) collectAll() {
	app.fsbi.Collect(app.dbh)
	app.tlwsbt.Collect(app.dbh)
	app.tiwsbt.Collect(app.dbh)
	app.essgben.Collect(app.dbh)
	app.ewsgben.Collect(app.dbh)
}

// do a fresh collection of data and then update the initial values based on that.
func (app *App) resetDBStatistics() {
	app.collectAll()
	app.setInitialFromCurrent()
}

func (app *App) setInitialFromCurrent() {
	start := time.Now()
	app.fsbi.SetInitialFromCurrent()
	app.tlwsbt.SetInitialFromCurrent()
	app.tiwsbt.SetInitialFromCurrent()
	app.essgben.SetInitialFromCurrent()
	app.ewsgben.SetInitialFromCurrent()
	app.updateLast()
	lib.Logger.Println("app.setInitialFromCurrent() took", time.Duration(time.Since(start)).String())
}

// update the last time that have relative data for
func (app *App) updateLast() {
	switch app.view.Get() {
	case view.ViewLatency, view.ViewOps:
		app.display.SetLast(app.tiwsbt.Last())
	case view.ViewIO:
		app.display.SetLast(app.fsbi.Last())
	case view.ViewLocks:
		app.display.SetLast(app.tlwsbt.Last())
	case view.ViewUsers:
		app.display.SetLast(app.users.Last())
	case view.ViewMutex:
		app.display.SetLast(app.ewsgben.Last())
	case view.ViewStages:
		app.display.SetLast(app.essgben.Last())
	}
}

// Collect the data we are looking at.
func (app *App) Collect() {
	start := time.Now()

	switch app.view.Get() {
	case view.ViewLatency, view.ViewOps:
		app.tiwsbt.Collect(app.dbh)
	case view.ViewIO:
		app.fsbi.Collect(app.dbh)
	case view.ViewLocks:
		app.tlwsbt.Collect(app.dbh)
	case view.ViewUsers:
		app.users.Collect(app.dbh)
	case view.ViewMutex:
		app.ewsgben.Collect(app.dbh)
	case view.ViewStages:
		app.essgben.Collect(app.dbh)
	}
	app.updateLast()
	app.wi.CollectedNow()
	lib.Logger.Println("app.Collect() took", time.Duration(time.Since(start)).String())
}

// SetHelp determines if we need to display help
func (app *App) SetHelp(newHelp bool) {
	app.help = newHelp

	app.display.ClearScreen()
}

// SetMySQLVersion saves the current MySQL version we're using
func (app *App) SetMySQLVersion(mysqlVersion string) {
	app.mysqlVersion = mysqlVersion
}

// SetHostname records the current hostname
func (app *App) SetHostname(hostname string) {
	lib.Logger.Println("app.SetHostname(", hostname, ")")
	app.hostname = hostname
}

// Help returns the internal help variable
func (app App) Help() bool {
	return app.help
}

// Display shows the output appropriate to the corresponding view and device
func (app *App) Display() {
	if app.help {
		app.display.DisplayHelp() // shouldn't get here if in --stdout mode
	} else {
		uptime, _ := lib.SelectGlobalStatusByVariableName(app.dbh, "UPTIME")
		app.display.SetUptime(uptime)

		switch app.view.Get() {
		case view.ViewLatency, view.ViewOps:
			app.display.Display(app.tiwsbt)
		case view.ViewIO:
			app.display.Display(app.fsbi)
		case view.ViewLocks:
			app.display.Display(app.tlwsbt)
		case view.ViewUsers:
			app.display.Display(app.users)
		case view.ViewMutex:
			app.display.Display(app.ewsgben)
		case view.ViewStages:
			app.display.Display(app.essgben)
		}
	}
}

// fixLatencySetting() ensures the SetWantsLatency() value is
// correct. This needs to be done more cleanly.
func (app *App) fixLatencySetting() {
	if app.view.Get() == view.ViewLatency {
		app.tiwsbt.SetWantsLatency(true)
	}
	if app.view.Get() == view.ViewOps {
		app.tiwsbt.SetWantsLatency(false)
	}
}

// change to the previous display mode
func (app *App) displayPrevious() {
	app.view.SetPrev()
	app.fixLatencySetting()
	app.display.ClearScreen()
	app.Display()
}

// change to the next display mode
func (app *App) displayNext() {
	app.view.SetNext()
	app.fixLatencySetting()
	app.display.ClearScreen()
	app.Display()
}

// WantRelativeStats returns whether we want to see data that's relative to the start of the program (or reset point)
func (app App) WantRelativeStats() bool {
	return app.wantRelativeStats
}

// SetWantRelativeStats sets whether we want to see data that's relative or absolute
func (app *App) SetWantRelativeStats(wantRelativeStats bool) {
	app.wantRelativeStats = wantRelativeStats

	app.fsbi.SetWantRelativeStats(wantRelativeStats)
	app.tlwsbt.SetWantRelativeStats(app.wantRelativeStats)
	app.tiwsbt.SetWantRelativeStats(app.wantRelativeStats)
	app.ewsgben.SetWantRelativeStats(app.wantRelativeStats)
	app.essgben.SetWantRelativeStats(app.wantRelativeStats)
	app.display.SetWantRelativeStats(app.wantRelativeStats)
}

// Cleanup prepares  the application prior to shutting down
func (app *App) Cleanup() {
	app.display.Close()
	if app.dbh != nil {
		app.setupInstruments.RestoreConfiguration()
		_ = app.dbh.Close()
	}
}

// Run runs the application in a loop until we're ready to finish
func (app *App) Run() {
	lib.Logger.Println("app.Run()")

	app.sigChan = make(chan os.Signal, 10) // 10 entries
	signal.Notify(app.sigChan, syscall.SIGINT, syscall.SIGTERM)

	eventChan := app.display.EventChan()

	for !app.Finished() {
		select {
		case sig := <-app.sigChan:
			fmt.Println("Caught signal: ", sig)
			app.finished = true
		case <-app.wi.WaitNextPeriod():
			app.Collect()
			app.Display()
			if app.stdout {
				app.setInitialFromCurrent()
			}
		case inputEvent := <-eventChan:
			switch inputEvent.Type {
			case event.EventFinished:
				app.finished = true
			case event.EventViewNext:
				app.displayNext()
			case event.EventViewPrev:
				app.displayPrevious()
			case event.EventDecreasePollTime:
				if app.wi.WaitInterval() > time.Second {
					app.wi.SetWaitInterval(app.wi.WaitInterval() - time.Second)
				}
			case event.EventIncreasePollTime:
				app.wi.SetWaitInterval(app.wi.WaitInterval() + time.Second)
			case event.EventHelp:
				app.SetHelp(!app.Help())
			case event.EventToggleWantRelative:
				app.SetWantRelativeStats(!app.WantRelativeStats())
				app.Display()
			case event.EventResetStatistics:
				app.resetDBStatistics()
				app.Display()
			case event.EventResizeScreen:
				width, height := inputEvent.Width, inputEvent.Height
				app.display.Resize(width, height)
				app.Display()
			case event.EventError:
				log.Fatalf("Quitting because of EventError error")
			}
		}
		// provide a hook to stop the application if the counter goes down to zero
		if app.stdout && app.count > 0 {
			app.count--
			if app.count == 0 {
				app.finished = true
			}
		}
	}
}

// pstop requires MySQL 5.6+ or MariaDB 10.0+. Check the version
// rather than giving an error message if the requires P_S tables can't
// be found.
func (app *App) validateMysqlVersion() error {
	var reValidVersion = regexp.MustCompile(`^(5\.[67]\.|10\.[01])`)

	var tables = [...]string{
		"performance_schema.events_stages_summary_global_by_event_name",
		"performance_schema.events_waits_summary_global_by_event_name",
		"performance_schema.file_summary_by_instance",
		"performance_schema.table_io_waits_summary_by_table",
		"performance_schema.table_lock_waits_summary_by_table",
	}

	lib.Logger.Println("validateMysqlVersion()")

	lib.Logger.Println("- Getting MySQL version")
	mysqlVersion, err := lib.SelectGlobalVariableByVariableName(app.dbh, "VERSION")
	if err != nil {
		return err
	}
	lib.Logger.Println("- mysqlVersion: '" + mysqlVersion + "'")

	if !reValidVersion.MatchString(mysqlVersion) {
		return errors.New(lib.MyName() + " does not work with MySQL version " + mysqlVersion)
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

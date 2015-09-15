// Package app is the "runtime" for the ps-top / ps-stats application packages
//
// This file contains the library routines related to running the app.
package app

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sjmudd/anonymiser"
	"github.com/sjmudd/ps-top/connector"
	"github.com/sjmudd/ps-top/context"
	"github.com/sjmudd/ps-top/display"
	"github.com/sjmudd/ps-top/event"
	fsbi "github.com/sjmudd/ps-top/file_io_latency"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/logger"
	"github.com/sjmudd/ps-top/memory_usage"
	ewsgben "github.com/sjmudd/ps-top/mutex_latency"
	"github.com/sjmudd/ps-top/p_s/ps_table"
	"github.com/sjmudd/ps-top/setup_instruments"
	essgben "github.com/sjmudd/ps-top/stages_latency"
	tiwsbt "github.com/sjmudd/ps-top/table_io_latency"
	tlwsbt "github.com/sjmudd/ps-top/table_lock_latency"
	"github.com/sjmudd/ps-top/user_latency"
	"github.com/sjmudd/ps-top/view"
	"github.com/sjmudd/ps-top/wait_info"
)

const (
	hostname = "HOSTNAME"
	uptime   = "UPTIME"
	version  = "VERSION"
)

// Flags for initialising the app
type Flags struct {
	Anonymise bool
	Conn      *connector.Connector
	Interval  int
	Count     int
	Stdout    bool
	View      string
	Disp      display.Display
}

// App holds the data needed by an application
type App struct {
	ctx                *context.Context
	count              int
	display            display.Display
	done               chan struct{}
	sigChan            chan os.Signal
	wi                 wait_info.WaitInfo
	finished           bool
	stdout             bool
	dbh                *sql.DB
	help               bool
	fsbi               ps_table.Tabler // ufsbi.File_summary_by_instance
	tiwsbt             tiwsbt.Object
	tlwsbt             ps_table.Tabler // tlwsbt.Table_lock_waits_summary_by_table
	ewsgben            ps_table.Tabler // ewsgben.Events_waits_summary_global_by_event_name
	essgben            ps_table.Tabler // essgben.Events_stages_summary_global_by_event_name
	memory             memory_usage.Object
	users              user_latency.Object
	view               view.View
	wait_info.WaitInfo // embedded
	setupInstruments   setup_instruments.SetupInstruments
	wantRelativeStats  bool
}

// Ignore any errors. Perhaps not ideal but makes the code easier to read
// further down.
func selectGlobalVariableByVariableName(dbh *sql.DB, variable string) string {
	result, _ := lib.SelectGlobalVariableByVariableName(dbh, variable)
	return result
}

// Ignore any errors. Perhaps not ideal but makes the code easier to read
// further down.
func selectGlobalStatusByVariableName(dbh *sql.DB, variable string) int {
	result, _ := lib.SelectGlobalStatusByVariableName(dbh, variable)
	return result
}

// NewApp sets up the application given various parameters.
func NewApp(flags Flags) *App {
	logger.Println("app.NewApp()")
	app := new(App)

	anonymiser.Enable(flags.Anonymise) // not dynamic at the moment
	app.ctx = new(context.Context)
	app.count = flags.Count
	app.dbh = flags.Conn.Handle()
	app.finished = false
	app.stdout = flags.Stdout
	app.display = flags.Disp
	app.display.SetContext(app.ctx)
	app.SetHelp(false)

	if err := view.ValidateViews(app.dbh); err != nil {
		log.Fatal(err)
	}

	logger.Println("app.Setup() Setting the default view to:", flags.View)
	app.view.SetByName(flags.View) // if empty will use the default

	app.setupInstruments = setup_instruments.NewSetupInstruments(app.dbh)
	app.setupInstruments.EnableMonitoring()

	app.wi.SetWaitInterval(time.Second * time.Duration(flags.Interval))

	variables, _ := lib.SelectAllGlobalVariablesByVariableName(app.dbh)

	// setup to their initial types/values
	app.fsbi = fsbi.NewFileSummaryByInstance(variables)
	app.tlwsbt = new(tlwsbt.Object)
	app.ewsgben = new(ewsgben.Object)
	app.essgben = new(essgben.Object)

	app.SetWantRelativeStats(true)
	app.fixLatencySetting() // adjust to see ops/latency

	app.resetDBStatistics()

	app.ctx.SetHostname(anonymiser.Anonymise("host", selectGlobalVariableByVariableName(app.dbh, hostname)))
	app.ctx.SetMySQLVersion(selectGlobalVariableByVariableName(app.dbh, version))

	return app
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
	app.users.Collect(app.dbh)
	app.essgben.Collect(app.dbh)
	app.ewsgben.Collect(app.dbh)
	app.memory.Collect(app.dbh)
}

func (app *App) WantRelativeStats() bool {
	return app.wantRelativeStats
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
	app.memory.SetInitialFromCurrent()
	logger.Println("app.setInitialFromCurrent() took", time.Duration(time.Since(start)).String())
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
	case view.ViewMemory:
		app.memory.Collect(app.dbh)
	}
	app.wi.CollectedNow()
	logger.Println("app.Collect() took", time.Duration(time.Since(start)).String())
}

// SetHelp determines if we need to display help
func (app *App) SetHelp(newHelp bool) {
	app.help = newHelp

	app.display.ClearScreen()
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
		app.ctx.SetUptime(selectGlobalStatusByVariableName(app.dbh, uptime))

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
		case view.ViewMemory:
			app.display.Display(app.memory)
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

// SetWantRelativeStats sets whether we want to see data that's relative or absolute
func (app *App) SetWantRelativeStats(want bool) {
	app.wantRelativeStats = want

	app.fsbi.SetWantRelativeStats(want)
	app.tlwsbt.SetWantRelativeStats(want)
	app.tiwsbt.SetWantRelativeStats(want)
	app.users.SetWantRelativeStats(want) // ignored
	app.essgben.SetWantRelativeStats(want)
	app.ewsgben.SetWantRelativeStats(want) // ignored
	app.memory.SetWantRelativeStats(want)
}

// Cleanup prepares  the application prior to shutting down
func (app *App) Cleanup() {
	app.display.Close()
	if app.dbh != nil {
		app.setupInstruments.RestoreConfiguration()
		_ = app.dbh.Close()
	}
	logger.Println("App.Cleanup completed")
}

// Run runs the application in a loop until we're ready to finish
func (app *App) Run() {
	logger.Println("app.Run()")

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
			case event.EventAnonymise:
				anonymiser.Enable(!anonymiser.Enabled()) // toggle current behaviour
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

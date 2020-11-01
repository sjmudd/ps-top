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
	"github.com/sjmudd/ps-top/global"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/logger"
	"github.com/sjmudd/ps-top/memory_usage"
	ewsgben "github.com/sjmudd/ps-top/mutex_latency"
	"github.com/sjmudd/ps-top/ps_table"
	"github.com/sjmudd/ps-top/setup_instruments"
	essgben "github.com/sjmudd/ps-top/stages_latency"
	tiwsbt "github.com/sjmudd/ps-top/table_io_latency"
	tlwsbt "github.com/sjmudd/ps-top/table_lock_latency"
	"github.com/sjmudd/ps-top/user_latency"
	"github.com/sjmudd/ps-top/view"
	"github.com/sjmudd/ps-top/wait_info"
)

// Flags for initialising the app
type Settings struct {
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
	ctx      *context.Context
	count    int
	display  display.Display
	done     chan struct{}
	sigChan  chan os.Signal
	wi       wait_info.WaitInfo
	finished bool
	stdout   bool
	db       *sql.DB
	help     bool
	fsbi     ps_table.Tabler // *ufsbi.File_summary_by_instance
	tiwsbt/* ps_table.Tabler */ *tiwsbt.Object
	tlwsbt             ps_table.Tabler // tlwsbt.Table_lock_waits_summary_by_table
	ewsgben            ps_table.Tabler // ewsgben.Events_waits_summary_global_by_event_name
	essgben            ps_table.Tabler // essgben.Events_stages_summary_global_by_event_name
	memory             ps_table.Tabler // memory_usage.Object
	users              ps_table.Tabler // user_latency.Object
	currentView        view.View
	wait_info.WaitInfo // embedded
	setupInstruments   setup_instruments.SetupInstruments
}

// ensure performance_schema is enabled
// - if not will not return and will exit
func ensurePerformanceSchemaEnabled(variables *global.Variables) {
	if variables == nil {
		log.Fatal("ensurePerformanceSchemaEnabled() variables is nil")
	}

	// check that performance_schema = ON
	if value := variables.Get("performance_schema"); value != "ON" {
		log.Fatal(fmt.Sprintf("ensurePerformanceSchemaEnabled(): performance_schema = '%s'. Please configure performance_schema = 1 in /etc/my.cnf (or equivalent) and restart mysqld to use %s.",
			value, lib.MyName()))
	} else {
		logger.Println("performance_schema = ON check succeeds")
	}
}

// NewApp sets up the application given various parameters.
func NewApp(settings Settings) *App {
	logger.Println("app.NewApp()")
	app := new(App)

	anonymiser.Enable(settings.Anonymise) // not dynamic at the moment
	app.db = settings.Conn.Handle()

	status := global.NewStatus(app.db)
	variables := global.NewVariables(app.db)
	// Prior to setting up screen check that performance_schema is enabled.
	// On MariaDB this is not the default setting so it will confuse people.
	ensurePerformanceSchemaEnabled(variables)

	app.ctx = context.NewContext(status, variables)
	app.ctx.SetWantRelativeStats(true)
	app.count = settings.Count
	app.finished = false

	app.stdout = settings.Stdout
	app.display = settings.Disp
	app.display.SetContext(app.ctx)
	app.SetHelp(false)

	if err := view.ValidateViews(app.db); err != nil {
		log.Fatal(err)
	}

	logger.Println("app.Setup() Setting the default view to:", settings.View)
	app.currentView.SetByName(settings.View) // if empty will use the default

	app.setupInstruments = setup_instruments.NewSetupInstruments(app.db)
	app.setupInstruments.EnableMonitoring()

	app.wi.SetWaitInterval(time.Second * time.Duration(settings.Interval))

	// setup to their initial types/values
	logger.Println("app.NewApp() Setup models")
	app.fsbi = fsbi.NewFileSummaryByInstance(app.ctx, app.db)
	app.tiwsbt = tiwsbt.NewTableIoLatency(app.ctx, app.db)
	app.tlwsbt = tlwsbt.NewTableLockLatency(app.ctx, app.db)
	app.ewsgben = ewsgben.NewMutexLatency(app.ctx, app.db)
	app.essgben = essgben.NewStagesLatency(app.ctx, app.db)
	app.memory = memory_usage.NewMemoryUsage(app.ctx, app.db)
	app.users = user_latency.NewUserLatency(app.ctx, app.db)
	logger.Println("app.NewApp() Finished initialising models")

	logger.Println("app.NewApp() fixLatencySetting()")
	app.fixLatencySetting() // adjust to see ops/latency

	logger.Println("app.NewApp() resetDBStatistics()")
	app.resetDBStatistics()

	logger.Println("app.NewApp() finishes")
	return app
}

// Finished tells us if we have finished
func (app App) Finished() bool {
	return app.finished
}

// CollectAll collects all the stats together in one go
func (app *App) collectAll() {
	logger.Println("app.collectAll() start")
	app.fsbi.Collect()
	app.tlwsbt.Collect()
	app.tiwsbt.Collect()
	app.users.Collect()
	app.essgben.Collect()
	app.ewsgben.Collect()
	app.memory.Collect()
	logger.Println("app.collectAll() finished")
}

// do a fresh collection of data and then update the initial values based on that.
func (app *App) resetDBStatistics() {
	logger.Println("app.resetDBStatistcs()")
	app.collectAll()
	app.setInitialFromCurrent()
}

func (app *App) setInitialFromCurrent() {
	start := time.Now()
	app.fsbi.SetInitialFromCurrent()
	app.tlwsbt.SetInitialFromCurrent()
	app.tiwsbt.SetInitialFromCurrent()
	app.users.SetInitialFromCurrent()
	app.essgben.SetInitialFromCurrent()
	app.ewsgben.SetInitialFromCurrent()
	app.memory.SetInitialFromCurrent()
	logger.Println("app.setInitialFromCurrent() took", time.Duration(time.Since(start)).String())
}

// Collect the data we are looking at.
func (app *App) Collect() {
	logger.Println("app.Collect()")
	start := time.Now()

	switch app.currentView.Get() {
	case view.ViewLatency, view.ViewOps:
		app.tiwsbt.Collect()
	case view.ViewIO:
		app.fsbi.Collect()
	case view.ViewLocks:
		app.tlwsbt.Collect()
	case view.ViewUsers:
		app.users.Collect()
	case view.ViewMutex:
		app.ewsgben.Collect()
	case view.ViewStages:
		app.essgben.Collect()
	case view.ViewMemory:
		app.memory.Collect()
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
		switch app.currentView.Get() {
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
	if app.currentView.Get() == view.ViewLatency {
		app.tiwsbt.SetWantsLatency(true)
	}
	if app.currentView.Get() == view.ViewOps {
		app.tiwsbt.SetWantsLatency(false)
	}
}

// change to the previous display mode
func (app *App) displayPrevious() {
	app.currentView.SetPrev()
	app.fixLatencySetting()
	app.display.ClearScreen()
	app.Display()
}

// change to the next display mode
func (app *App) displayNext() {
	app.currentView.SetNext()
	app.fixLatencySetting()
	app.display.ClearScreen()
	app.Display()
}

// Cleanup prepares  the application prior to shutting down
func (app *App) Cleanup() {
	app.display.Close()
	if app.db != nil {
		app.setupInstruments.RestoreConfiguration()
		_ = app.db.Close()
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
				app.ctx.SetWantRelativeStats(!app.ctx.WantRelativeStats())
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

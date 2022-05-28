// Package app is the "runtime" for the ps-top application.
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
	"github.com/sjmudd/ps-top/global"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/model/filter"
	"github.com/sjmudd/ps-top/mylog"
	"github.com/sjmudd/ps-top/pstable"
	"github.com/sjmudd/ps-top/setupinstruments"
	"github.com/sjmudd/ps-top/view"
	"github.com/sjmudd/ps-top/wait"
	"github.com/sjmudd/ps-top/wrapper/fileinfolatency"
	"github.com/sjmudd/ps-top/wrapper/memoryusage"
	"github.com/sjmudd/ps-top/wrapper/mutexlatency"
	"github.com/sjmudd/ps-top/wrapper/stageslatency"
	"github.com/sjmudd/ps-top/wrapper/tableiolatency"
	"github.com/sjmudd/ps-top/wrapper/tableioops"
	"github.com/sjmudd/ps-top/wrapper/tablelocklatency"
	"github.com/sjmudd/ps-top/wrapper/userlatency"
)

// Settings holds the application configuration settingss from the command line.
type Settings struct {
	Anonymise bool                   // Do we want to anonymise data shown?
	Filter    *filter.DatabaseFilter // optional names of databases to filter on
	Interval  int                    // default interval to poll information
	ViewName  string                 // name of the view to start with
}

// App holds the data needed by an application
type App struct {
	ctx              *context.Context                   // some context needed by the display
	display          *display.Display                   // display displays the information to the screen
	sigChan          chan os.Signal                     // signal handler channel
	waitHandler      wait.Handler                       // for handling waits
	Finished         bool                               // has the app finished?
	db               *sql.DB                            // connection to MySQL
	Help             bool                               // show help (during runtime)
	fileinfolatency  pstable.Tabler                     // file i/o latency information
	tableiolatency   pstable.Tabler                     // table i/o latency information
	tableioops       pstable.Tabler                     // table i/o operations information
	tablelocklatency pstable.Tabler                     // table lock information
	mutexlatency     pstable.Tabler                     // mutex latency information
	stageslatency    pstable.Tabler                     // stages latency information
	memory           pstable.Tabler                     // memory usage information
	users            pstable.Tabler                     // user information
	currentView      view.View                          // holds the view we are currently using
	setupInstruments *setupinstruments.SetupInstruments // for setting up and restoring performance_schema configuration.
}

// ensure performance_schema is enabled
// - if not will not return and will exit
func ensurePerformanceSchemaEnabled(variables *global.Variables) {
	if variables == nil {
		mylog.Fatal("ensurePerformanceSchemaEnabled() variables is nil")
	}

	// check that performance_schema = ON
	if value := variables.Get("performance_schema"); value != "ON" {
		mylog.Fatal(fmt.Sprintf("ensurePerformanceSchemaEnabled(): performance_schema = '%s'. Please configure performance_schema = 1 in /etc/my.cnf (or equivalent) and restart mysqld to use %s.",
			value, lib.ProgName))
	} else {
		log.Println("performance_schema = ON check succeeds")
	}
}

// NewApp sets up the application given various parameters.
func NewApp(
	connectorFlags connector.Config,
	settings Settings) *App {
	log.Println("app.NewApp()")
	app := new(App)

	anonymiser.Enable(settings.Anonymise)
	app.db = connector.NewConnector(connectorFlags).DB

	status := global.NewStatus(app.db)
	variables := global.NewVariables(app.db).SelectAll()
	// Prior to setting up screen check that performance_schema is enabled.
	// On MariaDB this is not the default setting so it will confuse people.
	ensurePerformanceSchemaEnabled(variables)

	app.ctx = context.NewContext(status, variables, settings.Filter, true)
	app.Finished = false
	app.display = display.NewDisplay(app.ctx)
	app.SetHelp(false)

	app.currentView = view.SetupAndValidate(settings.ViewName, app.db) // if empty will use the default

	app.setupInstruments = setupinstruments.NewSetupInstruments(app.db)
	app.setupInstruments.EnableMonitoring()
	app.waitHandler.SetWaitInterval(time.Second * time.Duration(settings.Interval))

	// setup to their initial types/values
	log.Println("app.NewApp() Setup models")
	app.fileinfolatency = fileinfolatency.NewFileSummaryByInstance(app.ctx, app.db)
	temptableiolatency := tableiolatency.NewTableIoLatency(app.ctx, app.db) // shared backend/metrics
	app.tableiolatency = temptableiolatency
	app.tableioops = tableioops.NewTableIoOps(temptableiolatency)
	app.tablelocklatency = tablelocklatency.NewTableLockLatency(app.ctx, app.db)
	app.mutexlatency = mutexlatency.NewMutexLatency(app.ctx, app.db)
	app.stageslatency = stageslatency.NewStagesLatency(app.ctx, app.db)
	app.memory = memoryusage.NewMemoryUsage(app.ctx, app.db)
	app.users = userlatency.NewUserLatency(app.ctx, app.db)
	log.Println("app.NewApp() Finished initialising models")

	app.resetDBStatistics()

	log.Println("app.NewApp() finishes")
	return app
}

// CollectAll collects all the stats together in one go
func (app *App) collectAll() {
	log.Println("app.collectAll() start")
	app.fileinfolatency.Collect()
	app.tablelocklatency.Collect()
	app.tableiolatency.Collect()
	app.users.Collect()
	app.stageslatency.Collect()
	app.mutexlatency.Collect()
	app.memory.Collect()
	log.Println("app.collectAll() finished")
}

// resetDBStatistics does a fresh collection of data and then updates the initial values based on that.
func (app *App) resetDBStatistics() {
	log.Println("app.resetDBStatistcs()")
	app.collectAll()
	app.resetStatistics()
}

func (app *App) resetStatistics() {
	start := time.Now()
	app.fileinfolatency.ResetStatistics()
	app.tablelocklatency.ResetStatistics()
	app.tableiolatency.ResetStatistics()
	app.users.ResetStatistics()
	app.stageslatency.ResetStatistics()
	app.mutexlatency.ResetStatistics()
	app.memory.ResetStatistics()

	log.Println("app.resetStatistics() took", time.Duration(time.Since(start)).String())
}

// Collect the data we are looking at.
func (app *App) Collect() {
	log.Println("app.Collect()")
	start := time.Now()

	switch app.currentView.Get() {
	case view.ViewLatency, view.ViewOps:
		app.tableiolatency.Collect()
	case view.ViewIO:
		app.fileinfolatency.Collect()
	case view.ViewLocks:
		app.tablelocklatency.Collect()
	case view.ViewUsers:
		app.users.Collect()
	case view.ViewMutex:
		app.mutexlatency.Collect()
	case view.ViewStages:
		app.stageslatency.Collect()
	case view.ViewMemory:
		app.memory.Collect()
	}
	app.waitHandler.CollectedNow()
	log.Println("app.Collect() took", time.Duration(time.Since(start)).String())
}

// SetHelp determines if we need to display help
func (app *App) SetHelp(help bool) {
	app.Help = help

	app.display.ClearScreen()
}

// Display shows the output appropriate to the corresponding view and device
func (app *App) Display() {
	if app.Help {
		app.display.DisplayHelp()
	} else {
		switch app.currentView.Get() {
		case view.ViewLatency:
			app.display.Display(app.tableiolatency)
		case view.ViewOps:
			app.display.Display(app.tableioops)
		case view.ViewIO:
			app.display.Display(app.fileinfolatency)
		case view.ViewLocks:
			app.display.Display(app.tablelocklatency)
		case view.ViewUsers:
			app.display.Display(app.users)
		case view.ViewMutex:
			app.display.Display(app.mutexlatency)
		case view.ViewStages:
			app.display.Display(app.stageslatency)
		case view.ViewMemory:
			app.display.Display(app.memory)
		}
	}
}

// change to the previous display mode
func (app *App) displayPrevious() {
	app.currentView.SetPrev()
	app.display.ClearScreen()
	app.Display()
}

// change to the next display mode
func (app *App) displayNext() {
	app.currentView.SetNext()
	app.display.ClearScreen()
	app.Display()
}

// Cleanup prepares the application prior to shutting down
func (app *App) Cleanup() {
	app.display.Close()
	if app.db != nil {
		app.setupInstruments.RestoreConfiguration()
		_ = app.db.Close()
	}
	log.Println("App.Cleanup completed")
}

// Run runs the application in a loop until we're ready to finish
func (app *App) Run() {
	log.Println("app.Run()")

	app.sigChan = make(chan os.Signal, 10) // 10 entries
	signal.Notify(app.sigChan, syscall.SIGINT, syscall.SIGTERM)

	eventChan := app.display.EventChan()

	for !app.Finished {
		select {
		case sig := <-app.sigChan:
			fmt.Println("Caught signal: ", sig)
			app.Finished = true
		case <-app.waitHandler.WaitUntilNextPeriod():
			app.Collect()
			app.Display()
		case inputEvent := <-eventChan:
			switch inputEvent.Type {
			case event.EventAnonymise:
				anonymiser.Enable(!anonymiser.Enabled()) // toggle current behaviour
			case event.EventFinished:
				app.Finished = true
			case event.EventViewNext:
				app.displayNext()
			case event.EventViewPrev:
				app.displayPrevious()
			case event.EventDecreasePollTime:
				if app.waitHandler.WaitInterval() > time.Second {
					app.waitHandler.SetWaitInterval(app.waitHandler.WaitInterval() - time.Second)
				}
			case event.EventIncreasePollTime:
				app.waitHandler.SetWaitInterval(app.waitHandler.WaitInterval() + time.Second)
			case event.EventHelp:
				app.SetHelp(!app.Help)
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
				mylog.Fatalf("Quitting because of EventError error")
			}
		}
	}
}

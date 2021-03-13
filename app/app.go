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
	"github.com/sjmudd/ps-top/logger"
	"github.com/sjmudd/ps-top/model/filter"
	"github.com/sjmudd/ps-top/ps_table"
	"github.com/sjmudd/ps-top/setup_instruments"
	"github.com/sjmudd/ps-top/view"
	"github.com/sjmudd/ps-top/wait"
	"github.com/sjmudd/ps-top/wrapper/file_io_latency"
	"github.com/sjmudd/ps-top/wrapper/memory_usage"
	"github.com/sjmudd/ps-top/wrapper/mutex_latency"
	"github.com/sjmudd/ps-top/wrapper/stages_latency"
	"github.com/sjmudd/ps-top/wrapper/table_io_latency"
	"github.com/sjmudd/ps-top/wrapper/table_io_ops"
	"github.com/sjmudd/ps-top/wrapper/table_lock_latency"
	"github.com/sjmudd/ps-top/wrapper/user_latency"
)

// Settings holds the application configuration settingss from the command line.
type Settings struct {
	Anonymise bool                   // Do we want to anonymise data shown?
	ConnFlags connector.Flags        // database connection flags
	Filter    *filter.DatabaseFilter // optional names of databases to filter on
	Interval  int                    // default interval to poll information
	View      string                 // which view to start with
}

// App holds the data needed by an application
type App struct {
	ctx                *context.Context                    // some context needed by the display
	display            *display.Display                    // display displays the information to the screen
	sigChan            chan os.Signal                      // signal handler channel
	waitHandler        wait.Handler                        // for handling waits
	Finished           bool                                // has the app finished?
	db                 *sql.DB                             // connection to MySQL
	Help               bool                                // show help (during runtime)
	file_io_latency    ps_table.Tabler                     // file i/o latency information
	table_io_latency   ps_table.Tabler                     // table i/o latency information
	table_io_ops       ps_table.Tabler                     // table i/o operations information
	table_lock_latency ps_table.Tabler                     // table lock information
	mutex_latency      ps_table.Tabler                     // mutex latency information
	stages_latency     ps_table.Tabler                     // stages latency information
	memory             ps_table.Tabler                     // memory usage information
	users              ps_table.Tabler                     // user information
	currentView        view.View                           // holds the view we are currently using
	setupInstruments   *setup_instruments.SetupInstruments // for setting up and restoring performance_schema configuration.
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
			value, lib.ProgName))
	} else {
		logger.Println("performance_schema = ON check succeeds")
	}
}

// NewApp sets up the application given various parameters.
func NewApp(settings Settings) *App {
	logger.Println("app.NewApp()")
	app := new(App)

	anonymiser.Enable(settings.Anonymise)
	app.db = connector.NewConnector(settings.ConnFlags).Handle()

	status := global.NewStatus(app.db)
	variables := global.NewVariables(app.db)
	// Prior to setting up screen check that performance_schema is enabled.
	// On MariaDB this is not the default setting so it will confuse people.
	ensurePerformanceSchemaEnabled(variables)

	app.ctx = context.NewContext(status, variables, settings.Filter, true)
	app.Finished = false
	app.display = display.NewDisplay(app.ctx)
	app.SetHelp(false)

	if err := view.ValidateViews(app.db); err != nil {
		log.Fatal(err)
	}

	logger.Println("app.Setup() Setting the default view to:", settings.View)
	app.currentView.SetByName(settings.View) // if empty will use the default

	app.setupInstruments = setup_instruments.NewSetupInstruments(app.db)
	app.setupInstruments.EnableMonitoring()
	app.waitHandler.SetWaitInterval(time.Second * time.Duration(settings.Interval))

	// setup to their initial types/values
	logger.Println("app.NewApp() Setup models")
	app.file_io_latency = file_io_latency.NewFileSummaryByInstance(app.ctx, app.db)
	temp_table_io_latency := table_io_latency.NewTableIoLatency(app.ctx, app.db) // shared backend/metrics
	app.table_io_latency = temp_table_io_latency
	app.table_io_ops = table_io_ops.NewTableIoOps(temp_table_io_latency)
	app.table_lock_latency = table_lock_latency.NewTableLockLatency(app.ctx, app.db)
	app.mutex_latency = mutex_latency.NewMutexLatency(app.ctx, app.db)
	app.stages_latency = stages_latency.NewStagesLatency(app.ctx, app.db)
	app.memory = memory_usage.NewMemoryUsage(app.ctx, app.db)
	app.users = user_latency.NewUserLatency(app.ctx, app.db)
	logger.Println("app.NewApp() Finished initialising models")

	app.resetDBStatistics()

	logger.Println("app.NewApp() finishes")
	return app
}

// CollectAll collects all the stats together in one go
func (app *App) collectAll() {
	logger.Println("app.collectAll() start")
	app.file_io_latency.Collect()
	app.table_lock_latency.Collect()
	app.table_io_latency.Collect()
	app.users.Collect()
	app.stages_latency.Collect()
	app.mutex_latency.Collect()
	app.memory.Collect()
	logger.Println("app.collectAll() finished")
}

// resetDBStatistics does a fresh collection of data and then updates the initial values based on that.
func (app *App) resetDBStatistics() {
	logger.Println("app.resetDBStatistcs()")
	app.collectAll()
	app.setFirstFromLast()
}

func (app *App) setFirstFromLast() {
	start := time.Now()
	app.file_io_latency.SetFirstFromLast()
	app.table_lock_latency.SetFirstFromLast()
	app.table_io_latency.SetFirstFromLast()
	app.users.SetFirstFromLast()
	app.stages_latency.SetFirstFromLast()
	app.mutex_latency.SetFirstFromLast()
	app.memory.SetFirstFromLast()

	logger.Println("app.setFirstFromLast() took", time.Duration(time.Since(start)).String())
}

// Collect the data we are looking at.
func (app *App) Collect() {
	logger.Println("app.Collect()")
	start := time.Now()

	switch app.currentView.Get() {
	case view.ViewLatency, view.ViewOps:
		app.table_io_latency.Collect()
	case view.ViewIO:
		app.file_io_latency.Collect()
	case view.ViewLocks:
		app.table_lock_latency.Collect()
	case view.ViewUsers:
		app.users.Collect()
	case view.ViewMutex:
		app.mutex_latency.Collect()
	case view.ViewStages:
		app.stages_latency.Collect()
	case view.ViewMemory:
		app.memory.Collect()
	}
	app.waitHandler.CollectedNow()
	logger.Println("app.Collect() took", time.Duration(time.Since(start)).String())
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
			app.display.Display(app.table_io_latency)
		case view.ViewOps:
			app.display.Display(app.table_io_ops)
		case view.ViewIO:
			app.display.Display(app.file_io_latency)
		case view.ViewLocks:
			app.display.Display(app.table_lock_latency)
		case view.ViewUsers:
			app.display.Display(app.users)
		case view.ViewMutex:
			app.display.Display(app.mutex_latency)
		case view.ViewStages:
			app.display.Display(app.stages_latency)
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
	logger.Println("App.Cleanup completed")
}

// Run runs the application in a loop until we're ready to finish
func (app *App) Run() {
	logger.Println("app.Run()")

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
				log.Fatalf("Quitting because of EventError error")
			}
		}
	}
}

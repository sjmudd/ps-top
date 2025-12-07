// Package app is the "runtime" for the ps-top application.
package app

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sjmudd/anonymiser"
	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/connector"
	"github.com/sjmudd/ps-top/display"
	"github.com/sjmudd/ps-top/event"
	"github.com/sjmudd/ps-top/global"
	"github.com/sjmudd/ps-top/log"
	"github.com/sjmudd/ps-top/model/filter"
	"github.com/sjmudd/ps-top/pstable"
	"github.com/sjmudd/ps-top/setupinstruments"
	"github.com/sjmudd/ps-top/utils"
	"github.com/sjmudd/ps-top/view"
	"github.com/sjmudd/ps-top/wait"
)

// Settings holds the application configuration settings from the command line.
type Settings struct {
	Anonymise bool                   // Do we want to anonymise data shown?
	Filter    *filter.DatabaseFilter // optional names of databases to filter on
	Interval  int                    // default interval to poll information
	ViewName  string                 // name of the view to start with
}

// App holds the data needed by an application
type App struct {
	config           *config.Config                     // some config needed by the display
	db               *sql.DB                            // connection to MySQL
	display          *display.Display                   // display displays the information to the screen
	finished         bool                               // has the app finished?
	sigChan          chan os.Signal                     // signal handler channel
	waiter           *wait.Waiter                       // for handling waits between collecting metrics
	help             bool                               // show help (during runtime)
	fileinfolatency  pstable.Tabler                     // file i/o latency information
	tableiolatency   pstable.Tabler                     // table i/o latency information
	tableioops       pstable.Tabler                     // table i/o operations information
	tablelocklatency pstable.Tabler                     // table lock information
	mutexlatency     pstable.Tabler                     // mutex latency information
	stageslatency    pstable.Tabler                     // stages latency information
	memory           pstable.Tabler                     // memory usage information
	users            pstable.Tabler                     // user information
	currentTabler    pstable.Tabler                     // current data being collected
	currentView      view.View                          // holds the view we are currently using
	setupInstruments *setupinstruments.SetupInstruments // for setting up and restoring performance_schema configuration.
}

var (
	errPeformanceSchemaEnabledVariables = errors.New("ensurePerformanceSchemaEnabled() variables is nil")
)

// return an error if performance_schema is not enabled
func performanceSchemaEnabled(variables *global.Variables) error {
	if variables == nil {
		return errPeformanceSchemaEnabledVariables
	}

	// check that performance_schema = ON
	if value := variables.Get("performance_schema"); value != "ON" {
		return fmt.Errorf("performanceSchemaEnabled: performance_schema = '%s'. Please configure performance_schema = 1 in /etc/my.cnf (or equivalent) and restart mysqld to use %s",
			value, utils.ProgName)
	}

	log.Println("performance_schema = ON check succeeds")

	return nil
}

// NewApp sets up the application given various parameters returning a possible if initialisation fails.
func NewApp(
	connectorFlags connector.Config,
	settings Settings) (*App, error) {
	log.Println("app.NewApp()")
	app := new(App)

	anonymiser.Enable(settings.Anonymise)
	app.db = connector.NewConnector(connectorFlags).DB

	status := global.NewStatus(app.db)
	variables := global.NewVariables(app.db)

	// Prior to setting up screen check that performance_schema is enabled.
	// On MariaDB this is not the default setting so it will confuse people.
	if err := performanceSchemaEnabled(variables); err != nil {
		return nil, err
	}

	app.config = config.NewConfig(status, variables, settings.Filter, true)
	app.display = display.NewDisplay(app.config)
	app.finished = false
	app.help = false
	app.display.Clear()

	app.setupInstruments = setupinstruments.NewSetupInstruments(app.db)
	app.setupInstruments.EnableMonitoring()

	app.waiter = wait.NewWaiter()
	app.waiter.SetWaitInterval(time.Second * time.Duration(settings.Interval))

	// setup to their initial types/values
	log.Println("app.NewApp: Setting up models")

	app.fileinfolatency = pstable.NewTabler(pstable.FileIoLatency, app.config, app.db)
	temptableiolatency := pstable.NewTabler(pstable.TableIoLatency, app.config, app.db) // shared backend/metrics
	app.tableiolatency = temptableiolatency
	app.tableioops = pstable.NewTableIoOps(temptableiolatency)
	app.tablelocklatency = pstable.NewTabler(pstable.TableLockLatency, app.config, app.db)
	app.mutexlatency = pstable.NewTabler(pstable.MutexLatency, app.config, app.db)
	app.stageslatency = pstable.NewTabler(pstable.StagesLatency, app.config, app.db)
	app.memory = pstable.NewTabler(pstable.MemoryUsage, app.config, app.db)
	app.users = pstable.NewTabler(pstable.UserLatency, app.config, app.db)

	log.Println("app.NewApp: model setup complete")

	app.resetDBStatistics()

	app.currentView = view.SetupAndValidate(settings.ViewName, app.db) // if empty will use the default
	app.UpdateCurrentTabler()

	log.Println("app.NewApp() finishes")
	return app, nil
}

// UpdateCurrentTabler updates the current tabler to use
func (app *App) UpdateCurrentTabler() {
	switch app.currentView.Get() {
	case view.ViewLatency:
		app.currentTabler = app.tableiolatency
	case view.ViewOps:
		app.currentTabler = app.tableioops
	case view.ViewIO:
		app.currentTabler = app.fileinfolatency
	case view.ViewLocks:
		app.currentTabler = app.tablelocklatency
	case view.ViewUsers:
		app.currentTabler = app.users
	case view.ViewMutex:
		app.currentTabler = app.mutexlatency
	case view.ViewStages:
		app.currentTabler = app.stageslatency
	case view.ViewMemory:
		app.currentTabler = app.memory
	}
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

	app.currentTabler.Collect()
	app.waiter.CollectedNow()
	log.Println("app.Collect() took", time.Duration(time.Since(start)).String())
}

// Display shows the output appropriate to the corresponding view and device
func (app *App) Display() {
	if app.help {
		app.display.Display(display.Help)
	} else {
		app.display.Display(app.currentTabler)
	}
}

// change to the previous display mode
func (app *App) displayPrevious() {
	app.currentView.SetPrev()
	app.UpdateCurrentTabler()
	app.display.Clear()
	app.Display()
}

// change to the next display mode
func (app *App) displayNext() {
	app.currentView.SetNext()
	app.UpdateCurrentTabler()
	app.display.Clear()
	app.Display()
}

// Cleanup prepares the application prior to shutting down
func (app *App) Cleanup() {
	app.display.Fini()
	if app.db != nil {
		app.setupInstruments.RestoreConfiguration()
		_ = app.db.Close()
	}
	log.Println("App.Cleanup completed")
}

// Run runs the application in a loop until we're ready to finish
func (app *App) Run() {
	defer app.Cleanup()

	log.Println("app.Run()")

	app.sigChan = make(chan os.Signal, 10) // 10 entries
	signal.Notify(app.sigChan, syscall.SIGINT, syscall.SIGTERM)

	eventChan := app.display.EventChan()

	for !app.finished {
		select {
		case sig := <-app.sigChan:
			log.Println("Caught signal: ", sig)
			app.finished = true
		case <-app.waiter.WaitUntilNextPeriod():
			app.Collect()
			app.Display()
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
				if app.waiter.WaitInterval() > time.Second {
					app.waiter.SetWaitInterval(app.waiter.WaitInterval() - time.Second)
				}
			case event.EventIncreasePollTime:
				app.waiter.SetWaitInterval(app.waiter.WaitInterval() + time.Second)
			case event.EventHelp:
				app.help = !app.help
				app.display.Clear()
			case event.EventToggleWantRelative:
				app.config.SetWantRelativeStats(!app.config.WantRelativeStats())
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

// Package app is the "runtime" for the ps-top application.
package app

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sjmudd/anonymiser"
	"github.com/sjmudd/ps-top/config"
	"github.com/sjmudd/ps-top/connector"
	"github.com/sjmudd/ps-top/display"
	"github.com/sjmudd/ps-top/event"
	"github.com/sjmudd/ps-top/global"
	"github.com/sjmudd/ps-top/log"
	"github.com/sjmudd/ps-top/model/filter"
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
	help             bool                               // show help (during runtime)
	collector        *DBCollector                       // owns all tablers and collection logic
	signalHandler    *SignalHandler                     // handles signals
	waiter           *wait.Waiter                       // for handling waits between collecting metrics
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
	conn, err := connector.NewConnector(connectorFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	app.db = conn.DB

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

	// Create DBCollector to manage all data collection (replaces individual tabler fields)
	log.Println("app.NewApp: Setting up models via DBCollector")
	app.collector = NewDBCollector(app.config, app.db)

	// Create signal handler
	app.signalHandler = NewSignalHandler()

	// Setup view (using collector's currentView field)
	var viewErr error
	app.collector.currentView, viewErr = view.SetupAndValidate(settings.ViewName, app.db) // if empty will use the default
	if viewErr != nil {
		return nil, fmt.Errorf("app.NewApp: %w", viewErr)
	}
	app.collector.UpdateCurrentTabler()

	// Initial collection and reset to establish baseline
	log.Println("app.NewApp: Initial collection and reset")
	app.collector.CollectAll()
	app.collector.ResetAll()

	log.Println("app.NewApp() finishes")
	return app, nil
}

// Collect the data we are looking at.
func (app *App) Collect() {
	log.Println("app.Collect()")
	start := time.Now()

	app.collector.Collect()
	app.waiter.CollectedNow()
	log.Println("app.Collect() took", time.Since(start))
}

// Display shows the output appropriate to the corresponding view and device
func (app *App) Display() {
	if app.help {
		app.display.Display(display.Help)
	} else {
		app.display.Display(app.collector.CurrentTabler())
	}
}

// change to the previous display mode
func (app *App) displayPrevious() {
	app.collector.currentView.SetPrev()
	app.collector.UpdateCurrentTabler()
	app.display.Clear()
	app.Display()
}

// change to the next display mode
func (app *App) displayNext() {
	app.collector.currentView.SetNext()
	app.collector.UpdateCurrentTabler()
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

	eventChan := app.display.EventChan()

	for !app.finished {
		select {
		case sig := <-app.signalHandler.Channel():
			log.Println("Caught signal: ", sig)
			app.finished = true
		case <-app.waiter.WaitUntilNextPeriod():
			app.collectAndDisplay()
		case inputEvent := <-eventChan:
			if app.handleInputEvent(inputEvent) {
				return
			}
		}
	}
}

// collectAndDisplay runs a collection and then updates the display.
// Extracted to keep the Run loop concise.
func (app *App) collectAndDisplay() {
	app.Collect()
	app.Display()
}

// handleInputEvent processes a single input event. It returns true if the
// caller should return immediately (used for EventError path so deferred
// Cleanup() runs).
func (app *App) handleInputEvent(inputEvent event.Event) bool {
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
		app.collector.ResetAll()
		app.Display()
	case event.EventResizeScreen:
		width, height := inputEvent.Width, inputEvent.Height
		app.display.Resize(width, height)
		app.Display()
	case event.EventError:
		// Avoid calling Fatalf while there is a defer (Cleanup) in Run();
		// set finished and return true so the caller returns and deferred
		// Cleanup() runs.
		log.Println("Quitting because of EventError error")
		app.finished = true
		return true
	}

	return false
}

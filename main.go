// pstop - Top like progream which collects information from MySQL's
// performance_schema database.
package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"runtime/pprof"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nsf/termbox-go"

	"github.com/sjmudd/mysql_defaults_file"
	"github.com/sjmudd/pstop/lib"
	"github.com/sjmudd/pstop/state"
	"github.com/sjmudd/pstop/version"
	"github.com/sjmudd/pstop/wait_info"
)

const (
	sql_driver = "mysql"
	db         = "performance_schema"
)

var (
	flag_debug         = flag.Bool("debug", false, "Enabling debug logging")
	flag_defaults_file = flag.String("defaults-file", "", "Provide a defaults-file to use to connect to MySQL")
	flag_help          = flag.Bool("help", false, "Provide some help for "+lib.MyName())
	flag_version       = flag.Bool("version", false, "Show the version of "+lib.MyName())
	cpuprofile         = flag.String("cpuprofile", "", "write cpu profile to file")

	re_valid_version = regexp.MustCompile(`^(5\.[67]\.|10\.[01])`)
)

// Connect to the database with the given defaults-file, or ~/.my.cnf if not provided.
func get_db_handle( defaults_file string ) *sql.DB {
	var err error
	var dbh *sql.DB
	lib.Logger.Println("get_db_handle() connecting to database")

	dbh, err = mysql_defaults_file.OpenUsingDefaultsFile(sql_driver, defaults_file, "performance_schema")
	if err != nil {
		log.Fatal(err)
	}
	if err = dbh.Ping(); err != nil {
		log.Fatal(err)
	}

	return dbh
}

// make chan for termbox events and run a poller to send events to the channel
// - return the channel
func new_tb_chan() chan termbox.Event {
	termboxChan := make(chan termbox.Event)
	go func() {
		for {
			termboxChan <- termbox.PollEvent()
		}
	}()
	return termboxChan
}

func usage() {
	fmt.Println(lib.MyName() + " - " + lib.Copyright())
	fmt.Println("")
	fmt.Println("Top-like program to show MySQL activity by using information collected")
	fmt.Println("from performance_schema.")
	fmt.Println("")
	fmt.Println("Usage: " + lib.MyName() + " <options>")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("-defaults-file=/path/to/defaults.file   Connect to MySQL using given defaults-file" )
	fmt.Println("-help                                   show this help message")
	fmt.Println("-version                                show the version")
}

// pstop requires MySQL 5.6+ or MariaDB 10.0+. Check the version
// rather than giving an error message if the requires P_S tables can't
// be found.
func validate_mysql_version(dbh *sql.DB) error {
	var tables = [...]string{
		"performance_schema.file_summary_by_instance",
		"performance_schema.table_io_waits_summary_by_table",
		"performance_schema.table_lock_waits_summary_by_table",
	}

	lib.Logger.Println("validate_mysql_version()")

	lib.Logger.Println("- Getting MySQL version")
	err, mysql_version := lib.SelectGlobalVariableByVariableName(dbh, "VERSION")
	if err != nil {
		return err
	}
	lib.Logger.Println("- mysql_version: '" + mysql_version + "'")

	if !re_valid_version.MatchString(mysql_version) {
		err := errors.New(lib.MyName() + " does not work with MySQL version " + mysql_version)
		return err
	}
	lib.Logger.Println("OK: MySQL version is valid, continuing")

	lib.Logger.Println("Checking access to required tables:")
	for i := range tables {
		if err := lib.CheckTableAccess(dbh, tables[i]); err == nil {
			lib.Logger.Println("OK: " + tables[i] + " found")
		} else {
			return err
		}
	}
	lib.Logger.Println("OK: all table checks passed")

	return nil
}

func main() {
	var defaults_file string = ""
	flag.Parse()

	// clean me up
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *flag_debug {
		lib.Logger.EnableLogging(true)
	}
	if *flag_version {
		fmt.Println(lib.MyName() + " version " + version.Version())
		return
	}
	if *flag_help {
		usage()
		return
	}

	lib.Logger.Println("Starting " + lib.MyName())

	if flag_defaults_file != nil && *flag_defaults_file != "" {
		defaults_file = *flag_defaults_file
	}

	dbh := get_db_handle( defaults_file )
	if err := validate_mysql_version(dbh); err != nil {
		log.Fatal(err)
	}

	var state state.State
	var wi wait_info.WaitInfo
	wi.SetWaitInterval(time.Second)

	sigChan := make(chan os.Signal, 1)
	done := make(chan struct{})
	defer close(done)
	termboxChan := new_tb_chan()

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	state.Setup(dbh)

	finished := false
	for !finished {
		select {
		case <-done:
			fmt.Println("exiting")
			finished = true
		case sig := <-sigChan:
			fmt.Println("Caught a signal", sig)
			done <- struct{}{}
		case <-wi.WaitNextPeriod():
			state.Collect()
			wi.CollectedNow()
			state.Display()
		case event := <-termboxChan:
			// switch on event type
			switch event.Type {
			case termbox.EventKey: // actions depend on key
				switch event.Key {
				case termbox.KeyCtrlZ, termbox.KeyCtrlC, termbox.KeyEsc:
					finished = true
				case termbox.KeyArrowLeft: // left arrow change to previous display mode
					state.DisplayPrevious()
					state.Display()
				case termbox.KeyTab, termbox.KeyArrowRight: // tab or right arrow - change to next display mode
					state.DisplayNext()
					state.Display()
				}
				switch event.Ch {
				case '-': // decrease the interval if > 1
					if wi.WaitInterval() > time.Second {
						wi.SetWaitInterval(wi.WaitInterval() - time.Second)
					}
				case '+': // increase interval by creating a new ticker
					wi.SetWaitInterval(wi.WaitInterval() + time.Second)
				case 'h', '?': // help
					state.SetHelp(!state.Help())
				case 'q': // quit
					finished = true
				case 't': // toggle between absolute/relative statistics
					state.SetWantRelativeStats(!state.WantRelativeStats())
					state.Display()
				case 'z': // reset the statistics to now by taking a query of current values
					state.ResetDBStatistics()
					state.Display()
				}
			case termbox.EventResize: // set sizes
				state.ScreenSetSize(event.Width, event.Height)
				state.Display()
			case termbox.EventError: // quit
				log.Fatalf("Quitting because of termbox error: \n%s\n", event.Err)
			}
		}
	}
	state.Cleanup()
	lib.Logger.Println("Terminating " + lib.MyName())
}

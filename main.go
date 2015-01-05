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
	flag_host          = flag.String("host", "", "Provide the hostname of the MySQL to connect to")
	flag_port          = flag.Int("port", 0 , "Provide the port number of the MySQL to connect to (default: 3306)") /* deliberately 0 here, defaults to 3306 elsewhere */
	flag_socket        = flag.String("socket", "", "Provide the path to the local MySQL server to connect to")
	flag_password      = flag.String("password", "", "Provide the password when connecting to the MySQL server")
	flag_user          = flag.String("user", "", "Provide the username to connect with to MySQL (default: $USER)")
	flag_version       = flag.Bool("version", false, "Show the version of "+lib.MyName())
	cpuprofile         = flag.String("cpuprofile", "", "write cpu profile to file")

	re_valid_version = regexp.MustCompile(`^(5\.[67]\.|10\.[01])`)
)

// Connect to the database with the given defaults-file, or ~/.my.cnf if not provided.
func connect_by_defaults_file( defaults_file string ) *sql.DB {
	var err error
	var dbh *sql.DB
	lib.Logger.Println("connect_by_defaults_file() connecting to database")

	dbh, err = mysql_defaults_file.OpenUsingDefaultsFile(sql_driver, defaults_file, "performance_schema")
	if err != nil {
		log.Fatal(err)
	}
	if err = dbh.Ping(); err != nil {
		log.Fatal(err)
	}

	return dbh
}

// connect to MySQL using various component parts needed to make the dsn
func connect_by_components( components map[string]string ) *sql.DB {
	var err error
	var dbh *sql.DB
	lib.Logger.Println("connect_by_components() connecting to database")

	new_dsn := mysql_defaults_file.BuildDSN(components, "performance_schema")
	dbh, err = sql.Open(sql_driver, new_dsn)
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
	fmt.Println("--defaults-file=/path/to/defaults.file   Connect to MySQL using given defaults-file" )
	fmt.Println("--help                                   Show this help message")
	fmt.Println("--version                                Show the version")
	fmt.Println("--host=<hostname>                        MySQL host to connect to")
	fmt.Println("--port=<port>                            MySQL port to connect to")
	fmt.Println("--socket=<path>                          MySQL path of the socket to connect to")
	fmt.Println("--user=<user>                            User to connect with")
	fmt.Println("--password=<password>                    Password to use when connecting")
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

	var dbh *sql.DB

	if *flag_host != "" || *flag_socket != "" {
		lib.Logger.Println("--host= or --socket= defined")
		var components = make(map[string]string)
		if *flag_host != "" && *flag_socket != "" {
			fmt.Println(lib.MyName() + ": Do not specify --host and --socket together" )
			os.Exit(1)
		}
		if *flag_host != "" {
			components["host"] = *flag_host
		}
		if *flag_port != 0 {
			if *flag_socket == "" {
				components["port"] = fmt.Sprintf("%d", *flag_port)
			} else {
				fmt.Println(lib.MyName() + ": Do not specify --socket and --port together" )
				os.Exit(1)
			}
		}
		if *flag_socket != "" {
			components["socket"] = *flag_socket
		}
		if *flag_user != "" {
			components["user"] = *flag_user
		}
		if *flag_password != "" {
			components["password"] = *flag_password
		}
		dbh = connect_by_components( components )
	} else {
		 if flag_defaults_file != nil && *flag_defaults_file != "" {
			lib.Logger.Println("--defaults-file defined")
			defaults_file = *flag_defaults_file
		} else {
			lib.Logger.Println("connecting by implicit defaults file")
		}
		dbh = connect_by_defaults_file( defaults_file )
	}

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
	for !state.Finished() {
		select {
		case <-done:
			fmt.Println("exiting")
			state.SetFinished()
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
					state.SetFinished()
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
					state.SetFinished()
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

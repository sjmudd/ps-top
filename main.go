// Top like progream which collects information from MySQL's
// performance_schema database.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nsf/termbox-go"

	"github.com/sjmudd/mysql_defaults_file"
	"github.com/sjmudd/pstop/lib"
	"github.com/sjmudd/pstop/state"
	"github.com/sjmudd/pstop/version"
)

const (
	sql_driver = "mysql"
	db         = "performance_schema"
)

var flag_version = flag.Bool("version", false, "Show the version of "+lib.MyName())
var flag_debug = flag.Bool("debug", false, "Enabling debug logging")
var flag_help = flag.Bool("help", false, "Provide some help for "+lib.MyName())
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func get_db_handle() *sql.DB {
	var err error
	var dbh *sql.DB

	dbh, err = mysql_defaults_file.OpenUsingDefaultsFile(sql_driver, "", "performance_schema")
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
	fmt.Println("-help      show this help message")
	fmt.Println("-version   show the version")
}

func main() {
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
	var state state.State

	interval := time.Second
	sigChan := make(chan os.Signal, 1)
	done := make(chan struct{})
	defer close(done)
	termboxChan := new_tb_chan()

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(interval) // generate a periodic signal

	state.Setup(get_db_handle())

	finished := false
	for !finished {
		select {
		case <-done:
			fmt.Println("exiting")
			finished = true
		case sig := <-sigChan:
			fmt.Println("Caught a signal", sig)
			done <- struct{}{}
		case <-ticker.C:
			state.Collect()
			state.Display()
		case event := <-termboxChan:
			// switch on event type
			switch event.Type {
			case termbox.EventKey: // actions depend on key
				switch event.Key {
				case termbox.KeyCtrlZ, termbox.KeyCtrlC, termbox.KeyEsc:
					finished = true
				case termbox.KeyTab: // tab - change display modes
					state.DisplayNext()
					state.Display()
				}
				switch event.Ch {
				case '-': // decrease the interval if > 1
					if interval > time.Second {
						ticker.Stop()
						interval -= time.Second
						ticker = time.NewTicker(interval)
					}
				case '+': // increase interval by creating a new ticker
					ticker.Stop()
					interval += time.Second
					ticker = time.NewTicker(interval)
				case 'h': // help
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
	ticker.Stop()
	lib.Logger.Println("Terminating " + lib.MyName())
}

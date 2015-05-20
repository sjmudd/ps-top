// ps-top - Top like program which collects information from MySQL's
// performance_schema database.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	_ "github.com/go-sql-driver/mysql"

	"github.com/sjmudd/ps-top/app"
	"github.com/sjmudd/ps-top/connector"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/version"
)

var (
	cpuprofile       = flag.String("cpuprofile", "", "write cpu profile to file")
	flagCount        = flag.Int("count", 0, "Provide the number of iterations to make (default: 0 is forever)")
	flagDebug        = flag.Bool("debug", false, "Enabling debug logging")
	flagDefaultsFile = flag.String("defaults-file", "", "Provide a defaults-file to use to connect to MySQL")
	flagHelp         = flag.Bool("help", false, "Provide some help for "+lib.MyName())
	flagHost         = flag.String("host", "", "Provide the hostname of the MySQL to connect to")
	flagInterval     = flag.Int("interval", 1, "Set the initial poll interval (default 1 second)")
	flagLimit        = flag.Int("limit", 0, "Show a maximum of limit entries (defaults to screen size if output to screen)")
	flagPassword     = flag.String("password", "", "Provide the password when connecting to the MySQL server")
	flagPort         = flag.Int("port", 0, "Provide the port number of the MySQL to connect to (default: 3306)") /* deliberately 0 here, defaults to 3306 elsewhere */
	flagSocket       = flag.String("socket", "", "Provide the path to the local MySQL server to connect to")
	flagUser         = flag.String("user", "", "Provide the username to connect with to MySQL (default: $USER)")
	flagVersion      = flag.Bool("version", false, "Show the version of "+lib.MyName())
	flagView         = flag.String("view", "", "Provide view to show when starting pstop (default: table_io_latency)")
)

func usage() {
	fmt.Println(lib.MyName() + " - " + lib.Copyright())
	fmt.Println("")
	fmt.Println("Top-like program to show MySQL activity by using information collected")
	fmt.Println("from performance_schema.")
	fmt.Println("")
	fmt.Println("Usage: " + lib.MyName() + " <options>")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("--count=<count>                          Set the number of times to watch")
	fmt.Println("--defaults-file=/path/to/defaults.file   Connect to MySQL using given defaults-file")
	fmt.Println("--help                                   Show this help message")
	fmt.Println("--host=<hostname>                        MySQL host to connect to")
	fmt.Println("--interval=<seconds>                     Set the default poll interval (in seconds)")
	fmt.Println("--limit=<rows>                           Limit the number of lines of output (excluding headers)")
	fmt.Println("--password=<password>                    Password to use when connecting")
	fmt.Println("--port=<port>                            MySQL port to connect to")
	fmt.Println("--socket=<path>                          MySQL path of the socket to connect to")
	fmt.Println("--user=<user>                            User to connect with")
	fmt.Println("--version                                Show the version")
	fmt.Println("--view=<view>                            Determine the view you want to see when pstop starts (default: table_io_latency")
	fmt.Println("                                         Possible values: table_io_latency table_io_ops file_io_latency table_lock_latency user_latency mutex_latency stages_latency")
}

func main() {
	var connector connector.Connector
	var defaultsFile string

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

	if *flagDebug {
		lib.Logger.EnableLogging(true)
	}
	if *flagVersion {
		fmt.Println(lib.MyName() + " version " + version.Version())
		return
	}
	if *flagHelp {
		usage()
		return
	}

	lib.Logger.Println("Starting " + lib.MyName())

	if *flagHost != "" || *flagSocket != "" {
		lib.Logger.Println("--host= or --socket= defined")
		var components = make(map[string]string)
		if *flagHost != "" && *flagSocket != "" {
			fmt.Println(lib.MyName() + ": Do not specify --host and --socket together")
			os.Exit(1)
		}
		if *flagHost != "" {
			components["host"] = *flagHost
		}
		if *flagPort != 0 {
			if *flagSocket == "" {
				components["port"] = fmt.Sprintf("%d", *flagPort)
			} else {
				fmt.Println(lib.MyName() + ": Do not specify --socket and --port together")
				os.Exit(1)
			}
		}
		if *flagSocket != "" {
			components["socket"] = *flagSocket
		}
		if *flagUser != "" {
			components["user"] = *flagUser
		}
		if *flagPassword != "" {
			components["password"] = *flagPassword
		}
		connector.ConnectByComponents(components)
	} else {
		if flagDefaultsFile != nil && *flagDefaultsFile != "" {
			lib.Logger.Println("--defaults-file defined")
			defaultsFile = *flagDefaultsFile
		} else {
			lib.Logger.Println("connecting by implicit defaults file")
		}
		connector.ConnectByDefaultsFile(defaultsFile)
	}

	var app app.App

	app.Setup(connector.Handle(), *flagInterval, *flagCount, false, *flagLimit, *flagView, false)
	app.Run()
	app.Cleanup()
	lib.Logger.Println("Terminating " + lib.MyName())
}

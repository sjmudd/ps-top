// ps-top - Top like program which collects information from MySQL's
// performance_schema database.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	"github.com/howeyc/gopass"

	"github.com/sjmudd/ps-top/app"
	"github.com/sjmudd/ps-top/connector"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/model/filter"
	"github.com/sjmudd/ps-top/mylog"
	"github.com/sjmudd/ps-top/version"
)

var (
	connectorFlags     connector.Flags
	cpuprofile         = flag.String("cpuprofile", "", "write cpu profile to file")
	flagAnonymise      = flag.Bool("anonymise", false, "Anonymise hostname, user, db and table names (default: false)")
	flagAskpass        = flag.Bool("askpass", false, "Ask for password interactively")
	flagDatabaseFilter = flag.String("database-filter", "", "Optional comma-separated filter of database names")
	flagDebug          = flag.Bool("debug", false, "Enabling debug logging")
	flagHelp           = flag.Bool("help", false, "Provide some help for "+lib.ProgName)
	flagInterval       = flag.Int("interval", 1, "Set the initial poll interval (default 1 second)")
	flagVersion        = flag.Bool("version", false, "Show the version of "+lib.ProgName)
	flagView           = flag.String("view", "", "Provide view to show when starting "+lib.ProgName+" (default: table_io_latency)")
)

func usage() {
	fmt.Println(lib.ProgName + " - " + lib.Copyright)
	fmt.Println("")
	fmt.Println("Top-like program to show MySQL activity by using information collected")
	fmt.Println("from performance_schema.")
	fmt.Println("")
	fmt.Println("Usage: " + lib.ProgName + " <options>")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("--anonymise=<true|false>                 Anonymise hostname, user, db and table names")
	fmt.Println("--askpass                                Request password to be provided interactively")
	fmt.Println("--database-filter=db1[,db2,db3,...]      Optional database names to filter on, default ''")
	fmt.Println("--defaults-file=/path/to/defaults.file   Connect to MySQL using given defaults-file, default ~/.my.cnf")
	fmt.Println("--help                                   Show this help message")
	fmt.Println("--host=<hostname>                        MySQL host to connect to")
	fmt.Println("--interval=<seconds>                     Set the default poll interval (in seconds)")
	fmt.Println("--password=<password>                    Password to use when connecting")
	fmt.Println("--port=<port>                            MySQL port to connect to")
	fmt.Println("--socket=<path>                          MySQL path of the socket to connect to")
	fmt.Println("--user=<user>                            User to connect with")
	fmt.Println("--use-environment                        Connect to MySQL using a go dsn collected from MYSQL_DSN e.g. MYSQL_DSN='test_user:test_pass@tcp(127.0.0.1:3306)/performance_schema'")
	fmt.Println("--version                                Show the version")
	fmt.Println("--view=<view>                            Determine the view you want to see when " + lib.ProgName + " starts (default: table_io_latency)")
	fmt.Println("                                         Possible values: table_io_latency table_io_ops file_io_latency table_lock_latency user_latency mutex_latency stages_latency")
}

// askPass asks for a password interactively from the user and returns it.
func askPass() (string, error) {
	fmt.Printf("Password: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		return "", err
	}
	stringPassword := string(pass) // converting from []char to string may not be perfect
	return stringPassword, nil
}

func main() {
	defaultsFile := flag.String("defaults-file", "", "Define the defaults file to read")
	host := flag.String("host", "", "Provide the hostname of the MySQL to connect to")
	password := flag.String("password", "", "Provide the password when connecting to the MySQL server")
	port := flag.Int("port", 0, "Provide the port number of the MySQL to connect to (default: 3306)") /* Port is deliberately 0 here, defaults to 3306 elsewhere */
	socket := flag.String("socket", "", "Provide the path to the local MySQL server to connect to")
	user := flag.String("user", "", "Provide the username to connect with to MySQL (default: $USER)")
	useEnvironment := flag.Bool("use-environment", false, "Use the environment variable MYSQL_DSN (go dsn) to connect with to MySQL")

	flag.Parse()

	// Enable logging if requested or PSTOP_DEBUG=1
	mylog.SetupLogging(*flagDebug || os.Getenv("PSTOP_DEBUG") == "1", lib.ProgName+".log")

	log.Printf("Starting %v version %v", lib.ProgName, version.Version)

	connectorFlags = connector.Flags{
		DefaultsFile:   defaultsFile,
		Host:           host,
		Password:       password,
		Port:           port,
		Socket:         socket,
		User:           user,
		UseEnvironment: useEnvironment,
	}

	if *flagAskpass {
		password, err := askPass()
		if err != nil {
			fmt.Printf("Failed to read password: %v\n", err)
			return
		}
		connectorFlags.Password = &password
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			mylog.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *flagHelp {
		usage()
		return
	}
	if *flagVersion {
		fmt.Println(lib.ProgName + " version " + version.Version)
		return
	}

	app := app.NewApp(app.Settings{
		Anonymise: *flagAnonymise,
		ConnFlags: connectorFlags,
		Filter:    filter.NewDatabaseFilter(*flagDatabaseFilter),
		Interval:  *flagInterval,
		ViewName:  *flagView,
	})
	defer app.Cleanup()
	app.Run()
}

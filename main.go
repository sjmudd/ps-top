// ps-top - Top like program which collects information from MySQL's
// performance_schema database.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/howeyc/gopass"

	"github.com/sjmudd/ps-top/app"
	"github.com/sjmudd/ps-top/connector"
	"github.com/sjmudd/ps-top/log"
	"github.com/sjmudd/ps-top/model/filter"
	"github.com/sjmudd/ps-top/utils"
)

var (
	// command line flags
	cpuprofile         = flag.String("cpuprofile", "", "write cpu profile to file")
	flagAnonymise      = flag.Bool("anonymise", false, "Anonymise hostname, user, db and table names (default: false)")
	flagAskpass        = flag.Bool("askpass", false, "Ask for password interactively")
	flagDatabaseFilter = flag.String("database-filter", "", "Optional comma-separated filter of database names")
	flagDebug          = flag.Bool("debug", false, "Enabling debug logging")
	flagHelp           = flag.Bool("help", false, "Provide some help for "+utils.ProgName)
	flagInterval       = flag.Int("interval", 1, "Set the initial poll interval (default 1 second)")
	flagVersion        = flag.Bool("version", false, "Show the version of "+utils.ProgName)
	flagView           = flag.String("view", "", "Provide view to show when starting "+utils.ProgName+" (default: table_io_latency)")

	getPasswdFunc = gopass.GetPasswd // to allow me to test
)

func usage() {
	lines := []string{
		utils.ProgName + " - " + utils.Copyright,
		"",
		"Top-like program to show MySQL activity by using information collected",
		"from performance_schema.",
		"",
		"Usage: " + utils.ProgName + " <options>",
		"",
		"Options:",
		"--anonymise=<true|false>                 Anonymise hostname, user, db and table names",
		"--askpass                                Request password to be provided interactively",
		"--database-filter=db1[,db2,db3,...]      Optional database names to filter on, default ''",
		"--defaults-file=/path/to/defaults.file   Connect to MySQL using given defaults-file, default ~/.my.cnf",
		"--help                                   Show this help message",
		"--host=<hostname>                        MySQL host to connect to",
		"--interval=<seconds>                     Set the default poll interval (in seconds)",
		"--password=<password>                    Password to use when connecting",
		"--port=<port>                            MySQL port to connect to",
		"--socket=<path>                          MySQL path of the socket to connect to",
		"--user=<user>                            User to connect with",
		"--use-environment                        Connect to MySQL using a go dsn collected from MYSQL_DSN e.g. MYSQL_DSN='test_user:test_pass@tcp(127.0.0.1:3306)/performance_schema'",
		"--version                                Show the version",
		"--view=<view>                            Determine the view you want to see when " + utils.ProgName + " starts (default: table_io_latency)",
		"                                         Possible values: table_io_latency table_io_ops file_io_latency table_lock_latency user_latency mutex_latency stages_latency",
	}

	for _, line := range lines {
		fmt.Println(line)
	}
}

// askPass asks for a password interactively from the user and returns it.
func askPass() (string, error) {
	fmt.Printf("Password: ")
	pass, err := getPasswdFunc()
	if err != nil {
		return "", err
	}
	stringPassword := string(pass) // converting from []char to string may not be perfect
	return stringPassword, nil
}

func main() {
	connectorConfig := connector.Config{
		DefaultsFile:   flag.String("defaults-file", "", "Define the defaults file to read"),
		Host:           flag.String("host", "", "Provide the hostname of the MySQL to connect to"),
		Password:       flag.String("password", "", "Provide the password when connecting to the MySQL server"),
		Port:           flag.Int("port", 0, "Provide the port number of the MySQL to connect to (default: 3306)"), /* Port is deliberately 0 here, defaults to 3306 elsewhere */
		Socket:         flag.String("socket", "", "Provide the path to the local MySQL server to connect to"),
		User:           flag.String("user", "", "Provide the username to connect with to MySQL (default: $USER)"),
		UseEnvironment: flag.Bool("use-environment", false, "Use the environment variable MYSQL_DSN (go dsn) to connect with to MySQL"),
	}
	flag.Parse()

	if *flagHelp {
		usage()
		return
	}

	// Enable logging if requested or PSTOP_DEBUG=1
	log.SetupLogging(*flagDebug || os.Getenv("PSTOP_DEBUG") == "1", utils.ProgName+".log")
	log.Printf("Starting %v version %v", utils.ProgName, utils.Version)

	if *flagAskpass {
		password, err := askPass()
		if err != nil {
			fmt.Printf("Failed to read password: %v\n", err)
			return
		}
		connectorConfig.Password = &password
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	if *flagVersion {
		fmt.Println(utils.ProgName + " version " + utils.Version)
		return
	}

	app, err := app.NewApp(
		connectorConfig,
		app.Settings{
			Anonymise: *flagAnonymise,
			Filter:    filter.NewDatabaseFilter(*flagDatabaseFilter),
			Interval:  *flagInterval,
			ViewName:  *flagView,
		},
	)

	if err != nil {
		log.Fatalf("Failed to start %s: %s", utils.ProgName, err)
	}
	app.Run()
}

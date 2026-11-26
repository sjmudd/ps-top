// Command ps-top collects performance_schema metrics on one interval and
// shows table views on another, full screen
// (the TUI, see internal/tui) or printed as text with --batch:
//
//	ps-top                                                 # the TUI: one view at a time, h for the keys
//	ps-top --interval=5s --window=30s                      # redraw every 5 s, differences over 30 s
//	ps-top --batch                                         # all views, every --interval, until interrupted
//	ps-top --batch --interval=10s --count=6                # every 10 s, 6 outputs
//	ps-top --batch --view=table_io_latency:current,file_io_latency
//	ps-top --batch --view=user_latency --limit=10          # top 10 lines
//	ps-top --batch --totals                                # totals lines only
//	ps-top --batch --anonymise                             # schema1.table2, user1: output to share
//	ps-top --batch --database-filter=shop,crm              # only the shop and crm databases
//
// The first output shows values up to now, as vmstat's first line does,
// the following ones each view in its style (by default the difference
// over --window, or current values).
//
// File I/O's table names are renamed by the [munge] section of ~/.pstoprc,
// if it exists (see internal/munge), so tables named alike show as one row:
//
//	[munge]
//	_[0-9]{8}$ = _YYYYMMDD                                 # shop.orders_20260101 shows as shop.orders_YYYYMMDD
//
// Connecting (see internal/connect):
//
//	ps-top --batch                                         # ~/.my.cnf if it exists, else $USER on 127.0.0.1:3306
//	ps-top --batch --defaults-file=/path/to/my.cnf
//	ps-top --batch --host=db1 --port=3307 --user=app --askpass
//	ps-top --batch --socket=/run/mysqld/mysqld.sock
//	ps-top --batch --dsn='app:secret@tcp(db1:3306)/performance_schema?interpolateParams=true'
//	MYSQL_DSN='app:secret@tcp(db1:3306)/performance_schema?interpolateParams=true' ps-top --batch --use-environment
//
// A DSN given with --dsn or MYSQL_DSN is used as given. The DSN built from
// the option file and flags sets interpolateParams=true, so a query with
// arguments is one round trip, not a prepared statement's three: add it to
// a DSN given.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/howeyc/gopass"

	"github.com/sjmudd/ps-top/v2/internal/app"
	"github.com/sjmudd/ps-top/v2/internal/connect"
	"github.com/sjmudd/ps-top/v2/internal/munge"
)

const version = "2.0.0"

func main() {
	cfg := app.Defaults
	var conn connect.Options
	flag.StringVar(&conn.DSN, "dsn", "", "connect with this go-sql-driver DSN, used as given; not with other connection options")
	flag.BoolVar(&conn.UseEnvironment, "use-environment", false, "connect with the DSN in MYSQL_DSN; not with other connection options")
	flag.StringVar(&conn.DefaultsFile, "defaults-file", "", "read the [client] group of this option file (default ~/.my.cnf, if it exists)")
	flag.StringVar(&conn.Host, "host", "", "MySQL host to connect to")
	flag.IntVar(&conn.Port, "port", 0, "MySQL port to connect to; 0 the default (3306)")
	flag.StringVar(&conn.Socket, "socket", "", "MySQL socket to connect to")
	flag.StringVar(&conn.User, "user", "", "user to connect with (default $USER)")
	flag.StringVar(&conn.Password, "password", "", "password to connect with")
	flag.BoolVar(&conn.Askpass, "askpass", false, "ask for the password")
	showVersion := flag.Bool("version", false, "show the version")
	durationVar(&cfg.CollectInterval, "collect-interval", cfg.CollectInterval, "collect interval")
	durationVar(&cfg.PresentInterval, "interval", cfg.PresentInterval, "time between outputs")
	durationVar(&cfg.Window, "window", cfg.Window, "difference over this long; a multiple of the collect interval")
	batch := flag.Bool("batch", false, "print the views as text, rather than the TUI")
	views := flag.String("view", "", "views to show, separated by \",\", each name[:current|difference]; default all")
	flag.IntVar(&cfg.Count, "count", 0, "outputs to print, the first included; 0 until interrupted")
	flag.IntVar(&cfg.Limit, "limit", 0, "at most this many lines per view; 0 no limit")
	flag.BoolVar(&cfg.TotalsOnly, "totals", false, "only each view's totals line, no detail lines")
	flag.IntVar(&cfg.Width, "width", 0, "cut lines at this many characters; 0 not cut")
	flag.BoolVar(&cfg.Anonymise, "anonymise", false, "show schema, table and user names anonymised, e.g. schema1.table2")
	databaseFilter := flag.String("database-filter", "", "show only these databases, separated by \",\", e.g. shop,crm; default all")
	var run time.Duration
	durationVar(&run, "for", 0, "run time; 0 runs until interrupted")
	cfg.MessageTime = time.Second
	durationVar(&cfg.MessageTime, "message-time", cfg.MessageTime, "how long the TUI shows a message")
	flag.Parse()

	if *showVersion {
		fmt.Println("ps-top version " + version)
		return
	}
	cfg.TUI = !*batch
	cfg.Program = filepath.Base(os.Args[0])
	cfg.Version = version
	var err error
	if cfg.Views, err = app.ParseViews(*views); err != nil {
		log.Fatal(err)
	}
	cfg.DatabaseFilter = app.ParseDatabaseFilter(*databaseFilter)
	rules, err := munge.Load(munge.DefaultPath())
	if err != nil {
		log.Fatal(err)
	}
	cfg.Munge = rules.Apply
	conn.ReadPassword = askPass
	if cfg.DSN, err = conn.BuildDSN(); err != nil {
		log.Fatal(err)
	}

	// Ctrl-C in the TUI is a key press: these are signals from outside.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()
	if run > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, run)
		defer cancel()
	}
	if err := app.Run(ctx, cfg, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

// durationVar defines a duration flag, as flag.DurationVar, that also takes
// a whole number of seconds: --interval=3 is --interval=3s. The usage ends
// with the forms taken, "duration" in backquotes so --help shows
// "-interval duration", as for flag.DurationVar.
func durationVar(p *time.Duration, name string, value time.Duration, usage string) {
	*p = value
	flag.Var((*durationValue)(p), name, usage+" (a `duration`, or whole seconds)")
}

// durationValue is a duration flag's value (see durationVar).
type durationValue time.Duration

func (d *durationValue) String() string { return time.Duration(*d).String() }

func (d *durationValue) Set(s string) error {
	v, err := parseDuration(s)
	if err != nil {
		return err
	}
	*d = durationValue(v)
	return nil
}

// parseDuration returns s as a duration: a whole number is seconds, e.g. "3"
// is 3s, anything else is as time.ParseDuration takes it, e.g. "500ms" or
// "1m". "1.5" and "-3" are errors: they need a unit, "1.5s", "-3s".
func parseDuration(s string) (time.Duration, error) {
	if n, err := strconv.ParseUint(s, 10, 64); err == nil {
		if n > math.MaxInt64/uint64(time.Second) {
			return 0, fmt.Errorf("%s seconds: too long", s)
		}
		return time.Duration(n) * time.Second, nil
	}
	return time.ParseDuration(s)
}

// askPass asks for the password on the terminal, the prompt on stderr
// so it is not mixed with output on stdout.
func askPass() (string, error) {
	fmt.Fprint(os.Stderr, "Password: ")
	pass, err := gopass.GetPasswd()
	return string(pass), err
}

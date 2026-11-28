// Package logging sets up slog's default logger from the --log and
// --log-file flags. Nothing is logged unless one of them is given:
//
//	ps-top                                     # no log
//	ps-top --log                               # info to ./ps-top.log
//	ps-top --log=debug                         # debug to ./ps-top.log
//	ps-top --log-file=/tmp/p.log               # info to /tmp/p.log
//	ps-top --log=false --log-file=/tmp/p.log   # no log: --log=false wins
//	ps-top --log=help                          # print LevelsHelp
//
// Other packages log with slog's top-level functions, e.g.
// slog.Info("chose connection method", "method", "dsn"). The message is a
// short description, the same for every record of its kind: the values
// are in the fields.
package logging

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
)

// DefaultFile is --log-file's default.
const DefaultFile = "ps-top.log"

// Field names used by records in more than one package, e.g.
// slog.Debug("ran metric query", logging.KeyMetric, "table_io", logging.KeyRows, 12).
const (
	KeyErr    = "err"
	KeyMetric = "metric"
	KeyRows   = "rows"
	KeySQL    = "sql"
	KeyTable  = "table"
	KeyTook   = "took"
)

// LevelsHelp is what --log=help prints.
const LevelsHelp = `--log levels, each logging its own and the ones below it:
  debug   per collection detail
  info    startup, settings and exit (--log, or --log-file alone)
  warn    ps-top carries on, with less or suspect data
  error   ps-top or a view stops
  false   no log (as no --log or --log-file)
`

// stderr reports fatal errors: after Setup, log's default logger writes to
// slog's handler, which may discard everything.
var stderr = log.New(os.Stderr, "", log.LstdFlags)

// Fatal prints v to stderr, as log.Fatal, and exits with status 1.
func Fatal(v ...any) { stderr.Fatal(v...) }

// Fatalf prints to stderr, as log.Fatalf, and exits with status 1.
func Fatalf(format string, v ...any) { stderr.Fatalf(format, v...) }

// Flags holds the --log and --log-file flags' values.
type Flags struct {
	level levelValue
	file  fileValue
}

// AddFlags defines --log and --log-file on fs.
func AddFlags(fs *flag.FlagSet) *Flags {
	f := &Flags{file: fileValue{name: DefaultFile}}
	fs.Var(&f.level, "log", "log at this level, debug, info, warn or error, to --log-file; --log is info, --log=help lists the levels")
	fs.Var(&f.file, "log-file", "`file` to log to, appended to; given without --log, logs at info")
	return f
}

// HelpWanted reports whether --log=help was given.
func (f *Flags) HelpWanted() bool { return f.level.help }

// Setup makes slog's default logger write to the --log-file at the --log
// level if either flag was given, and discard everything otherwise. It
// returns an error for an unknown --log value or a file it cannot open.
func (f *Flags) Setup() error {
	l := f.level
	if l.bad != "" {
		return fmt.Errorf("--log=%s: not one of debug, info, warn, error, false or help", l.bad)
	}
	if l.off || (!l.given && !f.file.given) {
		slog.SetDefault(slog.New(slog.DiscardHandler))
		return nil
	}
	level := slog.LevelInfo
	if l.given {
		level = l.level
	}
	file, err := os.OpenFile(f.file.name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("--log-file: %w", err)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{Level: level})))
	return nil
}

// levelValue is the --log flag: a boolean flag, so --log alone is
// --log=true (info), that also takes a level, e.g. --log=debug.
type levelValue struct {
	given bool   // --log given
	off   bool   // --log=false
	help  bool   // --log=help
	bad   string // none of these, reported by Setup
	level slog.Level
}

func (l *levelValue) IsBoolFlag() bool { return true }

func (l *levelValue) String() string {
	if l == nil || !l.given || l.off {
		return "false"
	}
	return strings.ToLower(l.level.String())
}

func (l *levelValue) Set(s string) error {
	l.given, l.off, l.bad = true, false, ""
	switch strings.ToLower(s) {
	case "true":
		l.level = slog.LevelInfo
	case "false":
		l.off = true
	case "help":
		l.help = true
	case "debug", "info", "warn", "error":
		return l.level.UnmarshalText([]byte(s))
	default:
		// Not an error here: flag would call it an "invalid boolean value".
		l.bad = s
	}
	return nil
}

// fileValue is the --log-file flag: a file name that records whether it
// was given, as giving it turns logging on.
type fileValue struct {
	given bool
	name  string
}

func (f *fileValue) String() string {
	if f == nil {
		return ""
	}
	return f.name
}

func (f *fileValue) Set(s string) error {
	f.given, f.name = true, s
	return nil
}

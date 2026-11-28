// Package connect works out the DSN ps-top connects with, from its
// command line connection options.
//
// One of three ways is used:
//
//   - --dsn: the DSN as given.
//   - --use-environment: the DSN in MYSQL_DSN.
//   - otherwise: the [client] group of the option file (--defaults-file,
//     else ~/.my.cnf if it exists), with --host, --port, --socket, --user,
//     --password or --askpass given on the command line in place of the
//     file's values. No database is selected: every query names its
//     schema, e.g. performance_schema.setup_instruments.
//
// --dsn and --use-environment cannot be used with any other connection
// option. --host and --socket cannot be used together, nor --socket and
// --port, nor --password and --askpass. The port is from 0 to 65535. Port 0 means the default (the
// file's port, else 3306), as in the MySQL client.
package connect

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/sjmudd/mysql_defaults_file"
)

// Options are the connection options from the command line. An option
// with its zero value was not given.
type Options struct {
	DSN            string
	UseEnvironment bool
	DefaultsFile   string // "" reads ~/.my.cnf if it exists
	Host           string
	Port           int
	Socket         string
	User           string
	Password       string
	Askpass        bool

	// ReadPassword asks for the password when Askpass is set.
	ReadPassword func() (string, error)
}

// given returns the names of the options given, other than --dsn and
// --use-environment.
func (o Options) given() []string {
	var names []string
	add := func(set bool, name string) {
		if set {
			names = append(names, name)
		}
	}
	add(o.DefaultsFile != "", "--defaults-file")
	add(o.Host != "", "--host")
	add(o.Port != 0, "--port")
	add(o.Socket != "", "--socket")
	add(o.User != "", "--user")
	add(o.Password != "", "--password")
	add(o.Askpass, "--askpass")
	return names
}

// set returns the settings the option file gave, e.g. [user password].
func set(c mysql_defaults_file.Config) []string {
	var names []string
	add := func(given bool, name string) {
		if given {
			names = append(names, name)
		}
	}
	add(c.Host != "", "host")
	add(c.Port != 0, "port")
	add(c.Socket != "", "socket")
	add(c.User != "", "user")
	add(c.Password != "", "password")
	return names
}

// BuildDSN returns the DSN to connect with, or an error if the options
// cannot be used together, a value is out of range, or the option file
// cannot be read. A DSN given with --dsn or MYSQL_DSN is returned as it is.
// A DSN built from the option file and flags sets interpolateParams=true:
// the driver puts a query's arguments into its text, escaped, and sends one
// query, rather than a prepared statement (prepare, execute and close, three
// round trips) each time.
func (o Options) BuildDSN() (string, error) {
	others := o.given()
	switch {
	case o.DSN != "":
		if o.UseEnvironment {
			others = append([]string{"--use-environment"}, others...)
		}
		if len(others) > 0 {
			return "", fmt.Errorf("--dsn cannot be used with %s", strings.Join(others, ", "))
		}
		return o.DSN, nil
	case o.UseEnvironment:
		if len(others) > 0 {
			return "", fmt.Errorf("--use-environment cannot be used with %s", strings.Join(others, ", "))
		}
		dsn := os.Getenv("MYSQL_DSN")
		if dsn == "" {
			return "", errors.New("--use-environment: MYSQL_DSN is not set or empty")
		}
		return dsn, nil
	}

	if o.Host != "" && o.Socket != "" {
		return "", errors.New("--host and --socket cannot be used together")
	}
	if o.Socket != "" && o.Port != 0 {
		return "", errors.New("--socket and --port cannot be used together")
	}
	if o.Password != "" && o.Askpass {
		return "", errors.New("--password and --askpass cannot be used together")
	}
	if o.Port < 0 || o.Port > 65535 {
		return "", fmt.Errorf("--port %d: not a number from 0 to 65535", o.Port)
	}
	var cfg mysql_defaults_file.Config
	var err error
	if o.DefaultsFile != "" {
		cfg, err = mysql_defaults_file.Load(o.DefaultsFile)
	} else {
		cfg, err = mysql_defaults_file.LoadOptional("")
	}
	if err != nil {
		return "", err
	}
	fromFile := set(cfg)

	// A host or socket given replaces the file's way of connecting.
	if o.Host != "" {
		cfg.Host, cfg.Socket = o.Host, ""
	}
	if o.Socket != "" {
		cfg.Socket, cfg.Host, cfg.Port = o.Socket, "", 0
	}
	if o.Port != 0 {
		cfg.Port = uint16(o.Port) // checked above to be in range
	}
	if o.User != "" {
		cfg.User = o.User
	}
	if o.Password != "" {
		cfg.Password = o.Password
	}
	if o.Askpass {
		if o.ReadPassword == nil {
			return "", errors.New("--askpass: no way to read the password")
		}
		if cfg.Password, err = o.ReadPassword(); err != nil {
			return "", fmt.Errorf("--askpass: %w", err)
		}
	}
	mc, err := cfg.MySQLConfig("")
	if err != nil {
		return "", err
	}
	slog.Info("determined appropriate connection settings", "file", cfg.Filename, "host", cfg.Host, "port", cfg.Port, "socket", cfg.Socket,
		"user", cfg.User, "password", cfg.Password != "",
		"from_file", strings.Join(fromFile, ","), "from_flags", strings.Join(o.given(), ","))
	// One round trip per query with arguments, not prepare, execute and close.
	mc.InterpolateParams = true
	return mc.FormatDSN(), nil
}

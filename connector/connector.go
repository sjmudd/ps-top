// Package connector is used to specify how to connect to MySQL.
// Then get a sql.*DB from it which is returned to the app.
package connector

import (
	"database/sql"
	"fmt"
	"math"

	"github.com/sjmudd/mysql_defaults_file"
	"github.com/sjmudd/ps-top/log"
)

// Method indicates how we want to connect to MySQL
type Method int

const (
	db           = "performance_schema" // database to connect to
	maxOpenConns = 5                    // maximum number of connections the go driver should keep open. Hard-coded value!
	sqlDriver    = "mysql"              // name of the go-sql-driver to use
	// ConnectByDefaultsFile indicates we want to connect using a MySQL defaults file
	ConnectByDefaultsFile Method = iota
	// ConnectByConfig indicates we want to connect by various components (fields)
	ConnectByConfig
	// ConnectByEnvironment indicates we want to connect by using MYSQL_DSN environment variable
	ConnectByEnvironment
)

// Config holds various command line configuration for connecting to the database
type Config struct {
	Host           *string // the host to connect to
	Socket         *string // the unix socket to connect with
	Port           *int    // the port to connect to
	User           *string // the user to connect with
	Password       *string // the password to use
	DefaultsFile   *string // name of the defaults file to use
	UseEnvironment *bool   // use the environment to set connection settings?
}

// Connector contains information on how to connect to MySQL
type Connector struct {
	method Method
	config mysql_defaults_file.Config
	DB     *sql.DB
}

// DefaultsFile returns the defaults file
func (c Connector) DefaultsFile() string {
	return c.config.Filename
}

// SetMethod records the method used to connect to the database
func (c *Connector) SetMethod(method Method) {
	c.method = method
}

// Connect makes a connection to the database using the configured settings
func (c *Connector) Connect() error {
	var err error

	switch c.method {
	case ConnectByConfig:
		log.Println("Connector.Connect: using ConnectByConfig...")
		c.DB, err = sql.Open(sqlDriver, mysql_defaults_file.BuildDSN(c.config, db))

	case ConnectByDefaultsFile:
		log.Println("Connector.Connect: ConnectByDefaultsFile...")
		c.DB, err = mysql_defaults_file.Open(c.config.Filename, db)

	case ConnectByEnvironment:
		/*********************************************************************************
		 *  WARNING             This functionality may be removed.              WARNING  *
		 *                                                                               *
		 *  Although I have implemented this it may not be good/safe to actually use it. *
		 *  See: http://dev.mysql.com/doc/refman/5.6/en/password-security-user.html      *
		 *  Store your password in the MYSQL_PWD environment variable. See Section       *
		 *  2.12, “Environment Variables”.                                               *
		 *********************************************************************************/
		log.Println("Connector.Connect: ConnectByEnvironment...")
		c.DB, err = mysql_defaults_file.OpenUsingEnvironment(sqlDriver)

	default:
		return fmt.Errorf("Connector.Connect: unexpected method %v", c.method)
	}

	// we catch Open...() errors here
	if err != nil {
		return fmt.Errorf("Connector.Connect: method: %v: %w", c.method, err)
	}

	// without calling Ping() we don't actually connect.
	if err = c.DB.Ping(); err != nil {
		return fmt.Errorf("Connector.Connect: Ping() failed: %w", err)
	}

	// Deliberately limit the pool size to 5 to avoid "problems" if any queries hang.
	c.DB.SetMaxOpenConns(maxOpenConns)

	return nil
}

// NewConnector returns a connected Connector given the provided configuration.
// It returns an error if connection fails instead of calling os.Exit.
func NewConnector(cfg Config) (*Connector, error) { // nolint:gocyclo
	var defaultsFile string
	connector := new(Connector)

	if *cfg.UseEnvironment {
		connector.method = ConnectByEnvironment
		if err := connector.Connect(); err != nil {
			return nil, fmt.Errorf("ConnectByEnvironment: %w", err)
		}
	} else {
		if *cfg.Host != "" || *cfg.Socket != "" {
			log.Println("--host= or --socket= defined")
			var config mysql_defaults_file.Config
			if *cfg.Host != "" && *cfg.Socket != "" {
				return nil, fmt.Errorf("do not specify --host and --socket together")
			}
			if *cfg.Host != "" {
				config.Host = *cfg.Host
			}
			if *cfg.Port != 0 {
				if *cfg.Socket == "" {
					// validate port number
					port := *cfg.Port
					if port < 0 || port > math.MaxUint16 {
						return nil, fmt.Errorf("invalid port value: %d", port)
					}
					config.Port = uint16(port) // nolint:gosec
				} else {
					return nil, fmt.Errorf("do not specify --socket and --port together")
				}
			}
			if *cfg.Socket != "" {
				config.Socket = *cfg.Socket
			}
			if *cfg.User != "" {
				config.User = *cfg.User
			}
			if *cfg.Password != "" {
				config.Password = *cfg.Password
			}
			connector.config = config
			connector.method = ConnectByConfig
			if err := connector.Connect(); err != nil {
				return nil, fmt.Errorf("ConnectByConfig: %w", err)
			}
		} else {
			// no host or socket provided so assume connecting by a defaults file.
			// - if an explicit defaults-file is provided use that.
			// - if no explicit defaults-file is provided
			//   we expect to IMPLICITLY use the default
			//   defaults-file, e.g. ~/.my.cnf.
			if cfg.DefaultsFile != nil && *cfg.DefaultsFile != "" {
				log.Println("--defaults-file defined")
				defaultsFile = *cfg.DefaultsFile
			} else {
				log.Println("NewConnector: connecting by implicit defaults file")
			}
			connector.config = mysql_defaults_file.NewConfig(defaultsFile)
			connector.method = ConnectByDefaultsFile
			if err := connector.Connect(); err != nil {
				return nil, fmt.Errorf("ConnectByDefaultsFile(%s): %w", defaultsFile, err)
			}
		}
	}

	return connector, nil
}

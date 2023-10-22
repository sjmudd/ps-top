// Package connector is used to specify how to connect to MySQL.
// Then get a sql.*DB from it which is returned to the app.
package connector

import (
	"database/sql"

	"github.com/sjmudd/mysql_defaults_file"
	"github.com/sjmudd/ps-top/log"
)

// ConnectMethod indicates how we want to connect to MySQL
type ConnectMethod int

const (
	db           = "performance_schema" // database to connect to
	maxOpenConns = 5                    // maximum number of connections the go driver should keep open. Hard-coded value!
	sqlDriver    = "mysql"              // name of the go-sql-driver to use
	// ConnectByDefaultsFile indicates we want to connect using a MySQL defaults file
	ConnectByDefaultsFile ConnectMethod = iota
	// ConnectByConfig indicates we want to connect by various components (fields)
	ConnectByConfig
	// ConnectByEnvironment indicates we want to connect by using MYSQL_DSN environment variable
	ConnectByEnvironment
)

// Connector contains information on how to connect to MySQL
type Connector struct {
	method ConnectMethod
	config mysql_defaults_file.Config
	DB     *sql.DB
}

// DefaultsFile returns the defaults file
func (c Connector) DefaultsFile() string {
	return c.config.Filename
}

// SetConnectBy records how we want to connect
func (c *Connector) SetConnectBy(method ConnectMethod) {
	c.method = method
}

// Connect makes a connection to the database using the previously defined settings
func (c *Connector) Connect() {
	var err error

	switch {
	case c.method == ConnectByConfig:
		log.Println("ConnectByConfig() Connecting...")
		c.DB, err = sql.Open(sqlDriver, mysql_defaults_file.BuildDSN(c.config, db))

	case c.method == ConnectByDefaultsFile:
		log.Println("ConnectByDefaults_file() Connecting...")
		c.DB, err = mysql_defaults_file.Open(c.config.Filename, db)

	case c.method == ConnectByEnvironment:
		/*********************************************************************************
		 *  WARNING             This functionality may be removed.              WARNING  *
		 *                                                                               *
		 *  Although I have implemented this it may not be good/safe to actually use it. *
		 *  See: http://dev.mysql.com/doc/refman/5.6/en/password-security-user.html      *
		 *  Store your password in the MYSQL_PWD environment variable. See Section       *
		 *  2.12, “Environment Variables”.                                               *
		 *********************************************************************************/
		log.Println("ConnectByEnvironment() Connecting...")
		c.DB, err = mysql_defaults_file.OpenUsingEnvironment(sqlDriver)

	default:
		log.Fatal("Connector.Connect() c.method not ConnectByDefaultsFile/ConnectByConfig/ConnectByEnvironment")
	}

	// we catch Open...() errors here
	if err != nil {
		log.Fatal(err)
	}

	// without calling Ping() we don't actually connect.
	if err = c.DB.Ping(); err != nil {
		log.Fatal(err)
	}

	// Deliberately limit the pool size to 5 to avoid "problems" if any queries hang.
	c.DB.SetMaxOpenConns(maxOpenConns)
}

// ConnectByConfig connects to MySQL using various configuration settings
// needed to create the DSN.
func (c *Connector) ConnectByConfig(config mysql_defaults_file.Config) {
	c.config = config
	c.SetConnectBy(ConnectByConfig)
	c.Connect()
}

// ConnectByDefaultsFile connects to the database with the given
// defaults-file, or ~/.my.cnf if not provided.
func (c *Connector) ConnectByDefaultsFile(defaultsFile string) {
	c.config = mysql_defaults_file.NewConfig(defaultsFile)
	c.SetConnectBy(ConnectByDefaultsFile)
	c.Connect()
}

// ConnectByEnvironment connects using environment variables
func (c *Connector) ConnectByEnvironment() {
	c.SetConnectBy(ConnectByEnvironment)
	c.Connect()
}

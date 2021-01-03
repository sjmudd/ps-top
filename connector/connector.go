// Package connector is used to specify how to connect to MySQL.
// Then get a sql.*DB from it which is returned to the app.
package connector

import (
	"database/sql"
	"log"

	"github.com/sjmudd/mysql_defaults_file"
	"github.com/sjmudd/ps-top/logger"
)

// ConnectMethod indicates how we want to connect to MySQL
type ConnectMethod int

const (
	db                                  = "performance_schema" // database to connect to
	maxOpenConns                        = 5                    // maximum number of connections the go driver should keep open. Hard-coded value!
	sqlDriver                           = "mysql"              // name of the go-sql-driver to use
	ConnectByDefaultsFile ConnectMethod = iota                 // ConnectByDefaultsFile indicates we want to connect using a MySQL defaults file
	ConnectByComponents                                        // ConnectByComponents indicates we want to connect by various components (fields)
	ConnectByEnvironment                                       // ConnectByEnvironment indicates we want to connect by using MYSQL_DSN environment variable
)

// Connector contains information on how to connect to MySQL
type Connector struct {
	method       ConnectMethod
	components   map[string]string
	defaultsFile string
	dbh          *sql.DB
}

// Handle returns the database handle
func (c Connector) Handle() *sql.DB {
	return c.dbh
}

// DefaultsFile returns the defaults file
func (c Connector) DefaultsFile() string {
	return c.defaultsFile
}

// SetDefaultsFile specifies the defaults file to use
func (c *Connector) SetDefaultsFile(defaultsFile string) {
	c.defaultsFile = defaultsFile
}

// SetComponents sets the component information needed to make the connection
func (c *Connector) SetComponents(components map[string]string) {
	c.components = components
}

// SetConnectBy records how we want to connect
func (c *Connector) SetConnectBy(method ConnectMethod) {
	c.method = method
}

// Connect makes a connection to the database using the previously defined settings
func (c *Connector) Connect() {
	var err error

	switch {
	case c.method == ConnectByComponents:
		logger.Println("ConnectByComponents() Connecting...")
		c.dbh, err = sql.Open(sqlDriver, mysql_defaults_file.BuildDSN(c.components, db))

	case c.method == ConnectByDefaultsFile:
		logger.Println("ConnectByDefaults_file() Connecting...")
		c.dbh, err = mysql_defaults_file.OpenUsingDefaultsFile(sqlDriver, c.defaultsFile, db)

	case c.method == ConnectByEnvironment:
		/*********************************************************************************
		 *  WARNING             This functionality may be removed.              WARNING  *
		 *                                                                               *
		 *  Although I have implemented this it may not be good/safe to actually use it. *
		 *  See: http://dev.mysql.com/doc/refman/5.6/en/password-security-user.html      *
		 *  Store your password in the MYSQL_PWD environment variable. See Section       *
		 *  2.12, “Environment Variables”.                                               *
		 *********************************************************************************/
		logger.Println("ConnectByEnvironment() Connecting...")
		c.dbh, err = mysql_defaults_file.OpenUsingEnvironment(sqlDriver)

	default:
		log.Fatal("Connector.Connect() c.method not ConnectByDefaultsFile/ConnectByComponents/ConnectByEnvironment")
	}

	// we catch Open...() errors here
	if err != nil {
		log.Fatal(err)
	}

	// without calling Ping() we don't actually connect.
	if err = c.dbh.Ping(); err != nil {
		log.Fatal(err)
	}

	// Deliberately limit the pool size to 5 to avoid "problems" if any queries hang.
	c.dbh.SetMaxOpenConns(maxOpenConns)
}

// ConnectByComponents connects to MySQL using various component
// parts needed to make the dsn.
func (c *Connector) ConnectByComponents(components map[string]string) {
	c.SetComponents(components)
	c.SetConnectBy(ConnectByComponents)
	c.Connect()
}

// ConnectByDefaultsFile connects to the database with the given
// defaults-file, or ~/.my.cnf if not provided.
func (c *Connector) ConnectByDefaultsFile(defaultsFile string) {
	c.SetDefaultsFile(defaultsFile)
	c.SetConnectBy(ConnectByDefaultsFile)
	c.Connect()
}

// ConnectByEnvironment connects using environment variables
func (c *Connector) ConnectByEnvironment() {
	c.SetConnectBy(ConnectByEnvironment)
	c.Connect()
}

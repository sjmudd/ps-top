// Package connector is used to specify how to connect to MySQL.
// Then get a sql.*DB from it which is returned to the app..
package connector

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql" // hide the actual driver used. Is this the right thing to do?

	"github.com/sjmudd/mysql_defaults_file"
	"github.com/sjmudd/ps-top/lib"
)

const (
	sqlDriver    = "mysql"
	db           = "performance_schema"
	// ConnectByDefaultsFile indicates we want to connect using a MySQL defaults file
	ConnectByDefaultsFile = iota
	// ConnectByComponents indicates we want to connect by various component (fields)
	ConnectByComponents   = iota
)

// Connector contains information on how you want to connect
type Connector struct {
	connectBy     int
	components    map[string]string
	defaultsFile string
	dbh           *sql.DB
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

// postConnectAction has things to do after connecting
func (c *Connector) postConnectAction() {
	if err := c.dbh.Ping(); err != nil {
		log.Fatal(err)
	}

	// deliberately limit the pool size to 5 to avoid "problems" if any queries hang.
	c.dbh.SetMaxOpenConns(5) // hard-coded value!
}

// SetConnectBy records how we want to connect
func (c *Connector) SetConnectBy(connectHow int) {
	c.connectBy = connectHow
}

// Connect makes a connection to the database using the previously defined settings
func (c *Connector) Connect() {
	var err error

	switch {
	case c.connectBy == ConnectByDefaultsFile:
		lib.Logger.Println("connect_by_defaults_file() connecting to database")

		c.dbh, err = mysql_defaults_file.OpenUsingDefaultsFile(sqlDriver, c.defaultsFile, db)
		if err != nil {
			log.Fatal(err)
		}
	case c.connectBy == ConnectByComponents:
		lib.Logger.Println("connect_by_components() connecting to database")

		newDsn := mysql_defaults_file.BuildDSN(c.components, db)
		c.dbh, err = sql.Open(sqlDriver, newDsn)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("Connector.Connect() c.connectBy not ConnectByDefaultsFile/ConnectByComponents")
	}
	c.postConnectAction()
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

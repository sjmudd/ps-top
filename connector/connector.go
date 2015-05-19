// Use Connector to specify how to connect to MySQL.
// Then get a sql.*DB from it which is returned to the app..
package connector

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"

	"github.com/sjmudd/mysql_defaults_file"
	"github.com/sjmudd/ps-top/lib"
)

const (
	sql_driver = "mysql"
	db         = "performance_schema"

	DEFAULTS_FILE = iota
	COMPONENTS    = iota
)

// connector struct
type Connector struct {
	connectBy     int
	components    map[string]string
	defaults_file string
	dbh           *sql.DB
}

// return the database handle
func (c Connector) Handle() *sql.DB {
	return c.dbh
}

// return the defaults file
func (c Connector) DefaultsFile() string {
	return c.defaults_file
}

// set the defaults file
func (c *Connector) SetDefaultsFile(defaults_file string) {
	c.defaults_file = defaults_file
}

// set the components
func (c *Connector) SetComponents(components map[string]string) {
	c.components = components
}

// things to do after connecting
func (c *Connector) postConnectStuff() {
	if err := c.dbh.Ping(); err != nil {
		log.Fatal(err)
	}

	// deliberately limit the pool size to 5 to avoid "problems" if any queries hang.
	c.dbh.SetMaxOpenConns(5) // hard-coded value!
}

// determine how we want to connect
func (c *Connector) SetConnectBy(connectHow int) {
	c.connectBy = connectHow
}

// make the database connection
func (c *Connector) Connect() {
	var err error

	switch {
	case c.connectBy == DEFAULTS_FILE:
		lib.Logger.Println("connect_by_defaults_file() connecting to database")

		c.dbh, err = mysql_defaults_file.OpenUsingDefaultsFile(sql_driver, c.defaults_file, db)
		if err != nil {
			log.Fatal(err)
		}
	case c.connectBy == COMPONENTS:
		lib.Logger.Println("connect_by_components() connecting to database")

		new_dsn := mysql_defaults_file.BuildDSN(c.components, db)
		c.dbh, err = sql.Open(sql_driver, new_dsn)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("Connector.Connect() c.connectBy not DEFAULTS_FILE/COMPONENTS")
	}
	c.postConnectStuff()
}

// Connect to MySQL using various component parts needed to make the dsn.
func (c *Connector) ConnectByComponents(components map[string]string) {
	c.SetComponents(components)
	c.SetConnectBy(COMPONENTS)
	c.Connect()
}

// Connect to the database with the given defaults-file, or ~/.my.cnf if not provided.
func (c *Connector) ConnectByDefaultsFile(defaults_file string) {
	c.SetDefaultsFile(defaults_file)
	c.SetConnectBy(DEFAULTS_FILE)
	c.Connect()
}

// Package table provides a simple way of checking access to a table
package table

import (
	"database/sql"
	"log"

	"github.com/sjmudd/ps-top/lib"
)

// Access holds a database and table name and information on whether the table is reachable
type Access struct {
	database            string
	table               string
	checkedSelectable   bool
	selectable          bool
	checkedConfigurable bool
	configurable        bool
}

// NewAccess returns a new Access type
func NewAccess(database, table string) Access {
	lib.Logger.Println("NewAccess(", database, ",", table, ")")
	return Access{database: database, table: table}
}

// Database returns the database name
func (ta Access) Database() string {
	return ta.database
}

// Table returns the table name
func (ta Access) Table() string {
	return ta.table
}

// Name returns the fully qualified table name
func (ta Access) Name() string {
	if len(ta.database) > 0 && len(ta.table) > 0 {
		return ta.database + "." + ta.table
	}
	return ""
}

// Selectable returns whether SELECT works on the table
func (ta *Access) CheckSelectable(dbh *sql.DB) bool {
	query := "SELECT 1 FROM " + ta.Name() + " LIMIT 1"

	if ta.checkedSelectable {
		return ta.selectable
	}

	var one int
	if err := dbh.QueryRow(query).Scan(&one); err == nil {
		ta.selectable = true
	} else {
		ta.selectable = false
	}
	ta.checkedSelectable = true

	return ta.selectable
}

// this hands back whatever it has
func (ta Access) Selectable() bool {
	if !ta.checkedSelectable {
		log.Fatal("table.Access.Selectable(", ta, ") called without having called CheckSelectable() first")
	}
	return ta.selectable
}

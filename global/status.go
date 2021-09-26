// Package global provides information on global variables and status
package global

import (
	"database/sql"
	"log"
)

func selectStatusFrom(seenError bool) string {
	if !seenError {
		return "INFORMATION_SCHEMA.GLOBAL_STATUS"
	}
	return "performance_schema.global_status"
}

// Status holds a handle to the database where the status can be queried
type Status struct {
	dbh *sql.DB
}

// NewStatus returns a *Status structure to the user
func NewStatus(dbh *sql.DB) *Status {
	if dbh == nil {
		log.Fatal("NewStatus() dbh is nil")
	}
	s := new(Status)
	s.dbh = dbh

	return s
}

/*
** mysql> select VARIABLE_VALUE from global_status where VARIABLE_NAME = 'UPTIME';
* +----------------+
* | VARIABLE_VALUE |
* +----------------+
* | 251107         |
* +----------------+
* 1 row in set (0.00 sec)
**/

// Get returns the value of the variable name requested (if found), or if not an error
// - note: we assume we have checked a variable first as there's no logic here to switch between I_S and P_S
func (status *Status) Get(name string) int {
	var value int

	query := "SELECT VARIABLE_VALUE from " + selectStatusFrom(seenCompatibiltyError) + " WHERE VARIABLE_NAME = ?"

	err := status.dbh.QueryRow(query, name).Scan(&value)
	switch {
	case err == sql.ErrNoRows:
		log.Println("global.SelectStatusByName(" + name + "): no status with this name")
	case err != nil:
		log.Fatal(err)
	default:
		// fmt.Println("value for", name, "is", value)
	}

	if err != nil {
		log.Fatal("Unable to retrieve status for '"+name+"':", err)
	}

	return value
}

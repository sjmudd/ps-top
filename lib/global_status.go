// Package lib provides global status information
package lib

import (
	"database/sql"
	"log"
)

/*
** mysql> select VARIABLE_VALUE from global_status where VARIABLE_NAME = 'UPTIME';
* +----------------+
* | VARIABLE_VALUE |
* +----------------+
* | 251107         |
* +----------------+
* 1 row in set (0.00 sec)
**/

// SelectGlobalStatusByVariableName returns the variable value of the given variable name (if found), or if not an error
func SelectGlobalStatusByVariableName(dbh *sql.DB, name string) (int, error) {
	sqlSelect := "SELECT VARIABLE_VALUE from INFORMATION_SCHEMA.GLOBAL_STATUS WHERE VARIABLE_NAME = ?"

	var value int
	err := dbh.QueryRow(sqlSelect, name).Scan(&value)
	switch {
	case err == sql.ErrNoRows:
		log.Println("No setting with that name", name)
	case err != nil:
		log.Fatal(err)
	default:
		// fmt.Println("value for", name, "is", value)
	}

	return value, err
}

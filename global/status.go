// Package global provides information on global variables and status
package global

import (
	"database/sql"

	"github.com/sjmudd/ps-top/logger"
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

// SelectStatusByName returns the variable value of the given variable name (if found), or if not an error
// - note: we assume we have checked a variable first as there's no logic here to switch between I_S and P_S
func SelectStatusByName(dbh *sql.DB, name string) (int, error) {
	query := "SELECT VARIABLE_VALUE from " + globalVariablesSchema + ".GLOBAL_STATUS WHERE VARIABLE_NAME = ?"

	var value int
	err := dbh.QueryRow(query, name).Scan(&value)
	switch {
	case err == sql.ErrNoRows:
		logger.Println("global.SelectStatusByName(" + name + "): no status with this name")
	case err != nil:
		logger.Fatal(err)
	default:
		// fmt.Println("value for", name, "is", value)
	}

	return value, err
}

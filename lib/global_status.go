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

// return the variable value of the given variable name (if found), or if not an error
func SelectGlobalStatusByVariableName(dbh *sql.DB, variable_name string) (error, int) {
	sql_select := "SELECT VARIABLE_VALUE from INFORMATION_SCHEMA.GLOBAL_STATUS WHERE VARIABLE_NAME = ?"

	var variable_value int
	err := dbh.QueryRow(sql_select, variable_name).Scan(&variable_value)
	switch {
	case err == sql.ErrNoRows:
		log.Println("No setting with that variable_name", variable_name)
	case err != nil:
		log.Fatal(err)
	default:
		// fmt.Println("variable_value for", variable_name, "is", variable_value)
	}

	return err, variable_value
}

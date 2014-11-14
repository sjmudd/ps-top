package lib

import (
	"database/sql"
	"log"
	"strings"
)

/*
** mysql> select VARIABLE_VALUE from information_schema.global_variables where variable_name = 'hostname';
* +----------------+
* | VARIABLE_VALUE |
* +----------------+
* | myhostname     |
* +----------------+
* 1 row in set (0.00 sec)
**/
func SelectGlobalVariableByVariableName(dbh *sql.DB, variable_name string) (error, string) {
	sql_select := "SELECT VARIABLE_VALUE FROM INFORMATION_SCHEMA.GLOBAL_VARIABLES WHERE VARIABLE_NAME = ?"

	var variable_value string
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

// Provide a slice of string and get back a hash of variable_name to value.
// - note the query is case insensitive for variable names.
// - they key values are lower-cased.
func SelectGlobalVariablesByVariableName(dbh *sql.DB, wanted []string) (error, map[string]string) {
	hashref := make(map[string]string)

	// create an IN list to make up the query
	quoted := make([]string, 0, len(wanted))
	sql_select := "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM INFORMATION_SCHEMA.GLOBAL_VARIABLES WHERE VARIABLE_NAME IN ("

	if len(wanted) == 0 {
		log.Fatal("SelectGlobalVariablesByVariableName() needs at least one entry")
	}

	for i := range wanted {
		quoted = append(quoted, "'"+wanted[i]+"'")

	}
	sql_select += strings.Join(quoted, ",")
	sql_select += ")"

	rows, err := dbh.Query(sql_select)
	defer rows.Close()

	for rows.Next() {
		var variable, value string
		if err := rows.Scan(&variable, &value); err != nil {
			log.Fatal(err)
		}
		hashref[strings.ToLower(variable)] = value
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return err, hashref
}

// Return all global variables as a hashref
func SelectAllGlobalVariablesByVariableName(dbh *sql.DB) (error, map[string]string) {
	hashref := make(map[string]string)

	sql_select := "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM INFORMATION_SCHEMA.GLOBAL_VARIABLES"

	rows, err := dbh.Query(sql_select)
	defer rows.Close()

	for rows.Next() {
		var variable, value string
		if err := rows.Scan(&variable, &value); err != nil {
			log.Fatal(err)
		}
		hashref[strings.ToLower(variable)] = value
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return err, hashref
}

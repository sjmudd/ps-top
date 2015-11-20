// Package lib with support for MySQL global variables
package lib

import (
	"database/sql"
	"log"
	"strings"
)

// SelectGlobalVariableByVariableName retrieves the value given the name
func SelectGlobalVariableByVariableName(dbh *sql.DB, variableName string) (string, error) {
	sqlSelect := "SELECT VARIABLE_VALUE FROM INFORMATION_SCHEMA.GLOBAL_VARIABLES WHERE VARIABLE_NAME = ?"

	var variableValue string
	err := dbh.QueryRow(sqlSelect, variableName).Scan(&variableValue)
	switch {
	case err == sql.ErrNoRows:
		log.Println("No setting with that variableName", variableName)
	case err != nil:
		log.Fatal(err)
	default:
		// fmt.Println("variableValue for", variableName, "is", variableValue)
	}

	return variableValue, err
}

// SelectGlobalVariablesByVariableName Provides a slice of string and get back a hash of variableName to value.
// - note the query is case insensitive for variable names.
// - they key values are lower-cased.
func SelectGlobalVariablesByVariableName(dbh *sql.DB, wanted []string) (map[string]string, error) {
	hashref := make(map[string]string)

	// create an IN list to make up the query
	quoted := make([]string, 0, len(wanted))
	sqlSelect := "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM INFORMATION_SCHEMA.GLOBAL_VARIABLES WHERE VARIABLE_NAME IN ("

	if len(wanted) == 0 {
		log.Fatal("SelectGlobalVariablesByVariableName() needs at least one entry")
	}

	for i := range wanted {
		quoted = append(quoted, "'"+wanted[i]+"'")

	}
	sqlSelect += strings.Join(quoted, ",")
	sqlSelect += ")"

	rows, err := dbh.Query(sqlSelect)
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

	return hashref, err
}

// SelectAllGlobalVariablesByVariableName returns all global variables as a hashref
func SelectAllGlobalVariablesByVariableName(dbh *sql.DB) (map[string]string, error) {
	hashref := make(map[string]string)

	sqlSelect := "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM INFORMATION_SCHEMA.GLOBAL_VARIABLES"

	rows, err := dbh.Query(sqlSelect)
	if err != nil {
		log.Fatal("SELECT on global variables failed:", err)
	}
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

	return hashref, err
}

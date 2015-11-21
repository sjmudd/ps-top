// Package global provides information on global variables and status
package global

import (
	"database/sql"
	"strings"

	"github.com/sjmudd/ps-top/logger"
)

// We expect to use I_S to query Global Variables. 5.7 now wants us to use P_S,
// so this variable will be changed if we see the show_compatibility_56 error message
var globalVariablesSchema = "INFORMATION_SCHEMA"

const showCompatibility56Error = "Error 3167: The 'INFORMATION_SCHEMA.GLOBAL_VARIABLES' feature is disabled; see the documentation for 'show_compatibility_56'"

// Generate the required SQL using the given database name.
func globalVariablesSelect(wanted []string) string {
	query := "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM " + globalVariablesSchema + ".GLOBAL_VARIABLES"

	if len(wanted) > 0 {
		// create an IN list to make up the query
		quoted := make([]string, 0, len(wanted))

		query += " WHERE VARIABLE_NAME IN ("

		for i := range wanted {
			quoted = append(quoted, "'"+wanted[i]+"'")

		}
		query += strings.Join(quoted, ",")
		query += ")"
	}

	return query
}

// SelectVariableByName retrieves the value given the name
// - note this is done case insensitively and if the value is not found an empty string is returned
func SelectVariableByName(dbh *sql.DB, variableName string) (string, error) {
	logger.Println("SelectVariableByName(?,", variableName, ")")
	m, err := SelectVariablesByName(dbh, []string{variableName})

	variableValue := ""
	for _, v := range m {
		variableValue = v
	}
	return variableValue, err
}

// SelectVariablesByName Provides a slice of string and get back a hash of variableName to value.
// - note the query is case insensitive for variable names.
// - the returned keys are lower-cased.
func SelectVariablesByName(dbh *sql.DB, wanted []string) (map[string]string, error) {
	logger.Println("SelectVariablesByName(?,", wanted, ")")

	hashref := make(map[string]string)
	query := globalVariablesSelect(wanted)
	logger.Println("query:", query)

	rows, err := dbh.Query(query)
	if err != nil {
		if (globalVariablesSchema == "INFORMATION_SCHEMA") && (err.Error() == showCompatibility56Error) {
			logger.Println("SelectVariablesByName() I_S query failed, trying with P_S")
			globalVariablesSchema = "PERFORMANCE_SCHEMA" // Change global variable to use P_S
			query = globalVariablesSelect(wanted)
			logger.Println("query:", query)

			rows, err = dbh.Query(query)
		}
		if err != nil {
			logger.Println("SelectVariablesByName() query failed with:", err)
			return nil, err
		}
	}
	logger.Println("SelectVariablesByName() query succeeded")
	defer rows.Close()

	for rows.Next() {
		var variable, value string
		if err := rows.Scan(&variable, &value); err != nil {
			logger.Fatal(err)
		}
		hashref[strings.ToLower(variable)] = value
	}
	if err := rows.Err(); err != nil {
		logger.Fatal(err)
	}
	logger.Println("SelectVariablesByName() result has", len(hashref), "rows")

	return hashref, err
}

// SelectAllVariables returns all global variables as a hashref
func SelectAllVariables(dbh *sql.DB) (map[string]string, error) {
	logger.Println("SelectAllVariables(?)")
	return SelectVariablesByName(dbh, nil)
}

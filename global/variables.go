// Package global provides information on global variables and status
package global

import (
	"database/sql"
	"strings"

	"github.com/sjmudd/ps-top/logger"
)

const showCompatibility56Error = "Error 3167: The 'INFORMATION_SCHEMA.GLOBAL_VARIABLES' feature is disabled; see the documentation for 'show_compatibility_56'"

// We expect to use I_S to query Global Variables. 5.7 now wants us to use P_S,
// so this variable will be changed if we see the show_compatibility_56 error message
var globalVariablesSchema = "INFORMATION_SCHEMA"

// Variables holds the handle and variables collected from the database
type Variables struct {
	dbh       *sql.DB
	variables map[string]string
}

// NewVariables returns a pointer to an initialised Variables structure
func NewVariables(dbh *sql.DB) *Variables {
	var err error
	if dbh == nil {
		logger.Fatal("NewVariables(): dbh == nil")
	}
	v := new(Variables)
	v.dbh = dbh
	if v.variables, err = SelectAllVariables(dbh); err != nil {
		logger.Fatal("NewVariables():", err)

	}
	return v
}

// Get returns the value of the given variable
func (v Variables) Get(key string) string {
	var result string
	var ok bool

	if result, ok = v.variables[key]; !ok {
		result = ""
	}

	return result
}

// SelectAllVariables returns all global variables as a hashref
// - note the query is case insensitive for variable names.
// - the returned keys are lower-cased.
func SelectAllVariables(dbh *sql.DB) (map[string]string, error) {
	logger.Println("SelectAllVariables(?)")

	hashref := make(map[string]string)
	query := "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM " + globalVariablesSchema + ".GLOBAL_VARIABLES"
	logger.Println("query:", query)

	rows, err := dbh.Query(query)
	if err != nil {
		if (globalVariablesSchema == "INFORMATION_SCHEMA") && (err.Error() == showCompatibility56Error) {
			logger.Println("SelectVariablesByName() I_S query failed, trying with P_S")
			globalVariablesSchema = "PERFORMANCE_SCHEMA" // Change global variable to use P_S
			query = "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM " + globalVariablesSchema + ".GLOBAL_VARIABLES"
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

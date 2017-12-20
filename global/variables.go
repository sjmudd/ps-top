// Package global provides information on global variables and status
package global

import (
	"database/sql"
	"strings"

	"github.com/sjmudd/ps-top/logger"
)

const (
	showCompatibility56Error = "Error 3167: The 'INFORMATION_SCHEMA.GLOBAL_VARIABLES' feature is disabled; see the documentation for 'show_compatibility_56'"

	defaultGlobalVariablesSchema = "INFORMATION_SCHEMA"
	defaultGlobalVariablesTable  = "GLOBAL_VARIABLES"
)

var (
	// We expect to use I_S to query Global Variables. 5.7 now wants us to use P_S,
	// so this variable will be changed if we see the show_compatibility_56 error message
	seenCompatibiltyError = false
)

func selectVariablesFrom(seenError bool) string {
	if !seenError {
		return "INFORMATION_SCHEMA.GLOBAL_VARIABLES"
	}
	return "performance_schema.global_variables"
}

// Variables holds the handle and variables collected from the database
type Variables struct {
	dbh       *sql.DB
	variables map[string]string
}

// NewVariables returns a pointer to an initialised Variables structure
func NewVariables(dbh *sql.DB) *Variables {
	if dbh == nil {
		logger.Fatal("NewVariables(): dbh == nil")
	}
	v := &Variables{
		dbh: dbh,
	}
	v.selectAll()

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

// selectAll() collects all variables from the database and stores for later use.
// - all returned keys are lower-cased.
func (v *Variables) selectAll() {
	hashref := make(map[string]string)

	query := "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM " + selectVariablesFrom(seenCompatibiltyError)
	logger.Println("query:", query)

	rows, err := v.dbh.Query(query)
	if err != nil {
		if !seenCompatibiltyError && err.Error() == showCompatibility56Error {
			logger.Println("selectAll() I_S query failed, trying with P_S")
			seenCompatibiltyError = true
			query = "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM " + selectVariablesFrom(seenCompatibiltyError)
			logger.Println("query:", query)

			rows, err = v.dbh.Query(query)
		}
		if err != nil {
			logger.Fatal("selectAll() query failed with:", err)
		}
	}
	logger.Println("selectAll() query succeeded")
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
	logger.Println("selectAll() result has", len(hashref), "rows")

	v.variables = hashref
}

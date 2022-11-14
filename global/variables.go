// Package global provides information on global variables and status
package global

import (
	"database/sql"
	"log"
	"strings"

	"github.com/sjmudd/ps-top/mylog"
)

const (
	showCompatibility56Error    = "Error 3167: The 'INFORMATION_SCHEMA.GLOBAL_VARIABLES' feature is disabled; see the documentation for 'show_compatibility_56'"
	globalVariablesNotInISError = "Error 1109: Unknown table 'GLOBAL_VARIABLES' in information_schema"

	informationSchemaGlobalVariables = "INFORMATION_SCHEMA.GLOBAL_VARIABLES"
	performanceSchemaGlobalVariables = "performance_schema.global_variables"
)

// We expect to use I_S to query Global Variables. 5.7+ now wants us to use P_S,
// so this variable will be changed if we see the show_compatibility_56 error message

// globally used by Status and Variables
var seenCompatibilityError bool

// may be modified by usePerformanceSchema()
var globalVariablesTable = informationSchemaGlobalVariables // default

// Variables holds the handle and variables collected from the database
type Variables struct {
	dbh       *sql.DB
	variables map[string]string
}

// shared by Status and Variables
// - no locking. Not sure if absolutely necessary.
func usePerformanceSchema() {
	seenCompatibilityError = true
	globalStatusTable = performanceSchemaGlobalStatus
	globalVariablesTable = performanceSchemaGlobalVariables
}

// NewVariables returns a pointer to an initialised Variables structure
func NewVariables(dbh *sql.DB) *Variables {
	if dbh == nil {
		mylog.Fatal("NewVariables(): dbh == nil")
	}

	return &Variables{
		dbh: dbh,
	}
}

// Get returns the value of the given variable if found or an empty string if not.
func (v Variables) Get(key string) string {
	var result string
	var ok bool

	if result, ok = v.variables[key]; !ok {
		result = ""
	}

	return result
}

// SelectAll collects all variables from the database and stores for later use.
// - all returned keys are lower-cased.
func (v *Variables) SelectAll() *Variables {
	hashref := make(map[string]string)

	query := "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM " + globalVariablesTable
	log.Println("query:", query)

	rows, err := v.dbh.Query(query)
	if err != nil {
		if !seenCompatibilityError && (err.Error() == showCompatibility56Error || err.Error() == globalVariablesNotInISError) {
			log.Println("selectAll() I_S query failed, trying with P_S")
			usePerformanceSchema()
			query = "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM " + globalVariablesTable
			log.Println("query:", query)

			rows, err = v.dbh.Query(query)
		}
		if err != nil {
			mylog.Fatal("selectAll() query", query, "failed with:", err)
		}
	}
	log.Println("selectAll() query succeeded")
	defer rows.Close()

	for rows.Next() {
		var variable, value string
		if err := rows.Scan(&variable, &value); err != nil {
			mylog.Fatal(err)
		}
		hashref[strings.ToLower(variable)] = value
	}
	if err := rows.Err(); err != nil {
		mylog.Fatal(err)
	}
	log.Println("selectAll() result has", len(hashref), "rows")

	v.variables = hashref

	return v
}

// Package global provides information on global variables and status
package global

import (
	"database/sql"
	"strings"

	"github.com/sjmudd/ps-top/log"
)

const (
	informationSchemaGlobalVariables = "INFORMATION_SCHEMA.GLOBAL_VARIABLES"
	performanceSchemaGlobalVariables = "performance_schema.global_variables"
)

// may be modified by usePerformanceSchema()
var variablesTable = informationSchemaGlobalVariables // default

// Variables holds the handle and variables collected from the database
type Variables struct {
	db        *sql.DB
	variables map[string]string
}

// NewVariables returns a pointer to an initialised Variables structure with one collection done.
func NewVariables(db *sql.DB) *Variables {
	if db == nil {
		log.Fatal("NewVariables(): db == nil")
	}

	v := &Variables{
		db: db,
	}
	return v.selectAll()
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

// selectAll collects all variables from the database and stores for later use.
// - all returned keys are lower-cased.
func (v *Variables) selectAll() *Variables {
	hashref := make(map[string]string)

	query := "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM " + variablesTable
	log.Println("query:", query)

	rows, err := v.db.Query(query)
	if err != nil {
		if !seenCompatibilityError && (IsMysqlError(err, showCompatibility56ErrorNum) || IsMysqlError(err, variablesNotInISErrorNum)) {
			log.Println("selectAll() I_S query failed, trying with P_S")
			usePerformanceSchema()
			query = "SELECT VARIABLE_NAME, VARIABLE_VALUE FROM " + variablesTable
			log.Println("query:", query)

			rows, err = v.db.Query(query)
		}
		if err != nil {
			log.Fatal("selectAll() query", query, "failed with:", err)
		}
	}
	log.Println("selectAll() query succeeded")

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
	_ = rows.Close()

	log.Println("selectAll() result has", len(hashref), "rows")

	v.variables = hashref

	return v
}

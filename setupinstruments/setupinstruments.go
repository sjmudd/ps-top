// Package setupinstruments manages the configuration of
// performance_schema.setupinstruments.
package setupinstruments

import (
	"database/sql"

	"github.com/sjmudd/ps-top/log"
)

// List of expected errors to an UPDATE statement.  Checks are only
// done against the error numbers.
var expectedErrors = []string{
	"Error 1142: UPDATE command denied to user 'myuser'@'10.11.12.13' for table 'setup_instruments'",
	"Error 1290: The MySQL server is running with the --read-only option so it cannot execute this statement",
}

// Row contains one row of performance_schema.setup_instruments
type Row struct {
	name    string
	enabled string
	timed   string
}

// SetupInstruments "object"
type SetupInstruments struct {
	updateTried     bool
	updateSucceeded bool
	rows            []Row
	db              *sql.DB
}

// NewSetupInstruments returns a pointer to a newly initialised
// SetupInstruments.
func NewSetupInstruments(db *sql.DB) *SetupInstruments {
	return &SetupInstruments{db: db}
}

// EnableMonitoring enables mutex and stage monitoring
func (si *SetupInstruments) EnableMonitoring() {
	si.EnableMutexMonitoring()
	si.EnableStageMonitoring()
}

// EnableStageMonitoring change settings to monitor stage/sql/%
func (si *SetupInstruments) EnableStageMonitoring() {
	log.Println("EnableStageMonitoring")
	sqlMatch := "stage/sql/%"
	sqlSelect := "SELECT NAME, ENABLED, TIMED FROM setup_instruments WHERE NAME LIKE '" + sqlMatch + "' AND 'YES' NOT IN (ENABLED,TIMED)"

	collecting := "Collecting setup_instruments stage/sql configuration settings"
	updating := "Updating setup_instruments configuration for: stage/sql"

	si.Configure(sqlSelect, collecting, updating)
	log.Println("EnableStageMonitoring finishes")
}

// EnableMutexMonitoring changes settings to monitor wait/synch/mutex/%
func (si *SetupInstruments) EnableMutexMonitoring() {
	log.Println("EnableMutexMonitoring")
	sqlMatch := "wait/synch/mutex/%"
	sqlSelect := "SELECT NAME, ENABLED, TIMED FROM setup_instruments WHERE NAME LIKE '" + sqlMatch + "' AND 'YES' NOT IN (ENABLED,TIMED)"
	collecting := "Collecting setup_instruments wait/synch/mutex configuration settings"
	updating := "Updating setup_instruments configuration for: wait/synch/mutex"

	si.Configure(sqlSelect, collecting, updating)
	log.Println("EnableMutexMonitoring finishes")
}

// isExpectedError returns true if the error is in the expected list of errors
// - we only match on the error number
func isExpectedError(actualError string) bool {
	var expected bool

	e := actualError[0:11]
	for _, val := range expectedErrors {
		if e == val[0:11] {
			expected = true
			break
		}
	}
	return expected
}

// Configure updates setup_instruments so we can monitor tables correctly.
func (si *SetupInstruments) Configure(sqlSelect string, collecting, updating string) {
	const updateSQL = "UPDATE setup_instruments SET enabled = ?, TIMED = ? WHERE NAME = ?"

	log.Printf("Configure(%q,%q,%q)", sqlSelect, collecting, updating)
	// skip if we've tried and failed
	if si.updateTried && !si.updateSucceeded {
		log.Println("SetupInstruments.Configure() - Skipping further configuration")
		return
	}

	// setup the old values in case they're not set
	if si.rows == nil {
		si.rows = make([]Row, 0, 500)
	}

	log.Println(collecting)

	log.Println("db.query", sqlSelect)
	rows, err := si.db.Query(sqlSelect)
	if err != nil {
		log.Fatal(err)
	}

	count := 0
	for rows.Next() {
		var r Row
		if err := rows.Scan(
			&r.name,
			&r.enabled,
			&r.timed); err != nil {
			log.Fatal(err)
		}
		si.rows = append(si.rows, r)
		count++
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	log.Println("- found", count, "rows whose configuration need changing")
	_ = rows.Close()

	// update the rows which need to be set - do multiple updates but I don't care
	log.Println(updating)

	log.Println("Preparing statement:", updateSQL)
	si.updateTried = true
	log.Println("db.Prepare", updateSQL)
	stmt, err := si.db.Prepare(updateSQL)
	if err != nil {
		log.Println("- prepare gave error:", err.Error())
		if !isExpectedError(err.Error()) {
			log.Fatal("Not expected error so giving up")
		} else {
			log.Println("- expected error so not running statement")
			_ = stmt.Close()
		}
	} else {
		log.Println("Prepare succeeded, trying to update", len(si.rows), "row(s)")
		count = 0
		for i := range si.rows {
			log.Println("- changing row:", si.rows[i].name)
			log.Println("stmt.Exec", "YES", "YES", si.rows[i].name)
			if res, err := stmt.Exec("YES", "YES", si.rows[i].name); err == nil {
				log.Println("update succeeded")
				si.updateSucceeded = true
				c, _ := res.RowsAffected()
				count += int(c)
			} else {
				si.updateSucceeded = false
				if isExpectedError(err.Error()) {
					log.Println("Insufficient privileges to UPDATE setup_instruments: " + err.Error())
					log.Println("Not attempting further updates")
					return
				}
				log.Fatal(err)
			}
		}
		if si.updateSucceeded {
			log.Println(count, "rows changed in p_s.setup_instruments")
		}
		_ = stmt.Close()
	}
	log.Println("Configure() returns updateTried", si.updateTried, ", updateSucceeded", si.updateSucceeded)
}

// RestoreConfiguration restores setup_instruments rows to their previous settings (if changed previously).
func (si *SetupInstruments) RestoreConfiguration() {
	log.Println("RestoreConfiguration()")
	// If the previous update didn't work then don't try to restore
	if !si.updateSucceeded {
		log.Println("Not restoring p_s.setup_instruments to original settings as initial configuration attempt failed")
		return
	}
	log.Println("Restoring p_s.setup_instruments to its original settings")

	// update the rows which need to be set - do multiple updates but I don't care
	updateSQL := "UPDATE setup_instruments SET enabled = ?, TIMED = ? WHERE NAME = ?"
	log.Println("db.Prepare(", updateSQL, ")")
	stmt, err := si.db.Prepare(updateSQL)
	if err != nil {
		log.Fatal(err)
	}
	count := 0
	for i := range si.rows {
		log.Println("stmt.Exec(", si.rows[i].enabled, si.rows[i].timed, si.rows[i].name, ")")
		if _, err := stmt.Exec(si.rows[i].enabled, si.rows[i].timed, si.rows[i].name); err != nil {
			log.Fatal(err)
		}
		count++
	}
	log.Println("stmt.Close()")
	_ = stmt.Close()
	log.Println(count, "rows changed in p_s.setup_instruments")
}

// Package setupinstruments manages the configuration of
// performance_schema.setupinstruments.
package setupinstruments

import (
	"database/sql"

	"github.com/sjmudd/ps-top/log"
)

const (
	mutexPrefix             = "wait/synch/mutex"
	sqlMutexMonitoringMatch = "wait/synch/mutex/%"
	sqlPrefix               = "stage/sql"
	sqlStageMonitoringMatch = "stage/sql/%"
)

// List of expected errors to an UPDATE statement.  Checks are only
// done against the error numbers.
var expectedErrors = []string{
	"Error 1142: UPDATE command denied to user 'myuser'@'10.11.12.13' for table 'setup_instruments'",
	"Error 1290: The MySQL server is running with the --read-only option so it cannot execute this statement",
}

func setupInstrumentsFilter(sqlMatch string) string {
	return "SELECT NAME, ENABLED, TIMED FROM setup_instruments WHERE NAME LIKE '" + sqlMatch + "' AND 'YES' NOT IN (ENABLED,TIMED)"
}

func collectingSetupInstrumentsMessage(filter string) string {
	return "Collecting setup_instruments " + filter + " configuration settings"
}

func updatingSetupInstrumentsMessage(filter string) string {
	return "Updating setup_instruments configuration for: " + filter
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

	si.Configure(
		setupInstrumentsFilter(sqlStageMonitoringMatch),
		collectingSetupInstrumentsMessage(sqlPrefix),
		updatingSetupInstrumentsMessage(sqlPrefix),
	)

	log.Println("EnableStageMonitoring finishes")
}

// EnableMutexMonitoring changes settings to monitor wait/synch/mutex/%
func (si *SetupInstruments) EnableMutexMonitoring() {
	log.Println("EnableMutexMonitoring")

	si.Configure(
		setupInstrumentsFilter(sqlMutexMonitoringMatch),
		collectingSetupInstrumentsMessage(mutexPrefix),
		updatingSetupInstrumentsMessage(mutexPrefix),
	)

	log.Println("EnableMutexMonitoring finishes")
}

// expectedError returns true if the error is in the expected list of errors
// - we only match on the error number
func expectedError(actualError string) bool {
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
	const (
		maxSetupInstrumentsRows = 1000
		updateSQL               = "UPDATE setup_instruments SET enabled = ?, TIMED = ? WHERE NAME = ?"
	)
	log.Printf("Configure(%q,%q,%q)", sqlSelect, collecting, updating)
	// skip if we've tried and failed
	if si.updateTried && !si.updateSucceeded {
		log.Println("SetupInstruments.Configure() - Skipping further configuration")
		return
	}

	// setup the old values in case they're not set
	if si.rows == nil {
		si.rows = make([]Row, 0, maxSetupInstrumentsRows)
	}

	log.Println(collecting)

	// fetch rows into si.rows
	count, err := si.fetchRows(sqlSelect)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("- found", count, "rows whose configuration need changing")

	if count > maxSetupInstrumentsRows {
		log.Printf("Warning: Unable to restore complete setup_instruments configuration. maxSetupInstrumentsRows=%v is too low. It should be at least %v", maxSetupInstrumentsRows, count)
	}

	// update the rows which need to be set - do multiple updates but I don't care
	log.Println(updating)

	log.Println("Preparing statement:", updateSQL)
	log.Println("db.Prepare", updateSQL)
	stmt, perr := si.prepareUpdateStmt(updateSQL)
	if perr != nil {
		log.Fatal(perr)
	}
	if stmt == nil {
		// expected error path - nothing to do
		return
	}

	// Ensure statement is closed when we're done.
	defer func() {
		_ = stmt.Close()
	}()

	si.updateTried = true
	si.updateSucceeded, count = si.executeUpdates(stmt)

	if si.updateSucceeded {
		log.Println(count, "rows changed in p_s.setup_instruments")
	}
	log.Println("Configure() returns updateTried", si.updateTried, ", updateSucceeded", si.updateSucceeded)
}

// fetchRows queries the DB and appends results into si.rows, returning the
// number of rows found or an error.
func (si *SetupInstruments) fetchRows(sqlSelect string) (int, error) {
	log.Println("db.query", sqlSelect)
	rows, err := si.db.Query(sqlSelect)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()

	count := 0
	for rows.Next() {
		var r Row
		if err = rows.Scan(&r.name, &r.enabled, &r.timed); err != nil {
			return 0, err
		}
		si.rows = append(si.rows, r)
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

// prepareUpdateStmt prepares the update statement and handles expected
// permission/read-only errors. Returns (nil,nil) if an expected error
// occurred (caller should treat as no-op). Returns (stmt,nil) on success
// or (nil,err) on unexpected fatal error.
func (si *SetupInstruments) prepareUpdateStmt(updateSQL string) (*sql.Stmt, error) {
	stmt, err := si.db.Prepare(updateSQL)
	if err != nil {
		log.Println("- prepare gave error:", err.Error())
		if !expectedError(err.Error()) {
			return nil, err
		}
		// expected error - nothing to do
		log.Println("- expected error so not running statement")
		return nil, nil
	}
	return stmt, nil
}

// executeUpdates runs the prepared statement against previously fetched
// rows. Returns whether any update succeeded and the number of affected rows.
func (si *SetupInstruments) executeUpdates(stmt *sql.Stmt) (bool, int) {
	log.Println("Prepare succeeded, trying to update", len(si.rows), "row(s)")
	count := 0
	for i := range si.rows {
		log.Println("- changing row:", si.rows[i].name)
		log.Println("- stmt.Exec", "YES", "YES", si.rows[i].name)
		res, err := stmt.Exec("YES", "YES", si.rows[i].name)
		if err == nil {
			log.Println("- update succeeded")
			si.updateSucceeded = true
			c, rerr := res.RowsAffected()
			if rerr != nil {
				log.Println("RowsAffected error:", rerr)
			} else {
				count += int(c)
			}
		} else {
			si.updateSucceeded = false
			if expectedError(err.Error()) {
				log.Println("Insufficient privileges to UPDATE setup_instruments: " + err.Error())
				log.Println("Not attempting further updates")
				return si.updateSucceeded, count
			}
			// Unexpected error: log and return so defer runs
			log.Println("Unexpected error during stmt.Exec:", err)
			return si.updateSucceeded, count
		}
	}
	return si.updateSucceeded, count
}

// RestoreConfiguration restores setup_instruments rows to their previous settings (if changed previously).
func (si *SetupInstruments) RestoreConfiguration() {
	const updateSQL = "UPDATE setup_instruments SET enabled = ?, TIMED = ? WHERE NAME = ?"

	log.Println("RestoreConfiguration()")
	// If the previous update didn't work then don't try to restore
	if !si.updateSucceeded {
		log.Println("Not restoring p_s.setup_instruments to original settings as initial configuration attempt failed")
		return
	}
	log.Println("Restoring p_s.setup_instruments to its original settings")

	// update the rows which need to be set - do multiple updates but I don't care
	log.Println("db.Prepare(", updateSQL, ")")
	stmt, err := si.db.Prepare(updateSQL)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = stmt.Close()
		log.Println("stmt.Close()")
	}()

	changed := 0
	for i := range si.rows {
		log.Println("stmt.Exec(", si.rows[i].enabled, si.rows[i].timed, si.rows[i].name, ")")
		if _, err := stmt.Exec(si.rows[i].enabled, si.rows[i].timed, si.rows[i].name); err != nil {
			// Avoid calling Fatal inside a function that has a defer (stmt.Close).
			// Log the error and return so deferred cleanup runs.
			log.Println("stmt.Exec error:", err)
			return
		}
		changed++
	}
	log.Println(changed, "rows changed in p_s.setup_instruments")
}

package probe

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"

	"github.com/sjmudd/ps-top/v2/internal/fakesql"
)

var processlist = []Candidate{
	{Name: "ps", Query: "SELECT ID FROM performance_schema.processlist"},
	{Name: "is", Query: "SELECT ID FROM information_schema.processlist"},
}

func TestChoose(t *testing.T) {
	mysql := &Server{Version: "9.7.2", Usable: map[string]bool{"ps": true, "is": true}}
	mariadb := &Server{Flavour: MariaDB, Version: "12.3.3-MariaDB",
		Usable:   map[string]bool{"is": true},
		Unusable: map[string]string{"ps": "Error 1146"},
	}
	if c, err := mysql.Choose(processlist); err != nil || c.Name != "ps" {
		t.Errorf("MySQL: %v %v, want ps", c.Name, err)
	}
	if c, err := mariadb.Choose(processlist); err != nil || c.Name != "is" {
		t.Errorf("MariaDB: %v %v, want is", c.Name, err)
	}
}

func TestChooseNone(t *testing.T) {
	s := &Server{Version: "8.0.46", Unusable: map[string]string{"ps": "Error 1142"}}
	_, err := s.Choose(processlist)
	if err == nil {
		t.Fatal("nothing usable: no error")
	}
	for _, want := range []string{"8.0.46", "ps: Error 1142", "is: not probed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}

func TestSQLBuilders(t *testing.T) {
	if q := queryProbeSQL("SELECT a FROM s.t WHERE a > 0 LIMIT 5"); q != "SELECT * FROM (SELECT a FROM s.t WHERE a > 0 LIMIT 5) AS p LIMIT 0" {
		t.Errorf("queryProbeSQL = %q", q)
	}
	if q, err := updateProbeSQL("performance_schema.setup_instruments", "enabled"); err != nil ||
		q != "UPDATE performance_schema.setup_instruments SET enabled = enabled WHERE 1=0" {
		t.Errorf("updateProbeSQL = %q, %v", q, err)
	}
	if q, err := showWhereSQL(showStatus, []string{"uptime", "questions"}); err != nil ||
		q != "SHOW GLOBAL STATUS WHERE Variable_name IN ('uptime', 'questions')" {
		t.Errorf("showWhereSQL = %q, %v", q, err)
	}
	for _, bad := range []string{"processlist", "a.b; DROP TABLE x", "a.b c", "A.B", "a.b.c"} {
		if _, err := updateProbeSQL(bad, "c"); err == nil {
			t.Errorf("updateProbeSQL(%q): no error", bad)
		}
	}
	if _, err := updateProbeSQL("a.b", "c = 1"); err == nil {
		t.Error("updateProbeSQL with bad column: no error")
	}
	if _, err := showWhereSQL(showVariables, []string{"x') OR ('1"}); err == nil {
		t.Error("showWhereSQL with quote: no error")
	}
}

func TestNumber(t *testing.T) {
	if n := number(nil); n != 0 {
		t.Errorf("number(nil) = %d", n)
	}
	if n := number(&mysql.MySQLError{Number: 1146}); n != errNoSuchTable {
		t.Errorf("number(1146) = %d", n)
	}
	if n := number(fmt.Errorf("wrapped: %w", &mysql.MySQLError{Number: 1142})); n != errTableAccessDenied {
		t.Errorf("number(wrapped 1142) = %d", n)
	}
	if n := number(errors.New("driver: bad connection")); n != -1 {
		t.Errorf("number(other) = %d", n)
	}
}

func TestFlavourString(t *testing.T) {
	if MySQL.String() != "MySQL" || MariaDB.String() != "MariaDB" {
		t.Error("Flavour.String")
	}
}

// Server errors as the driver returns them.
var (
	errTable    = &mysql.MySQLError{Number: 1146, Message: "Table 'performance_schema.processlist' doesn't exist"}
	errDenied   = &mysql.MySQLError{Number: 1142, Message: "UPDATE command denied to user 'app'@'localhost' for table 'setup_instruments'"}
	errColumn   = &mysql.MySQLError{Number: 1143, Message: "SELECT command denied to user 'app'@'localhost' for column 'ID'"}
	errField    = &mysql.MySQLError{Number: 1054, Message: "Unknown column 'X' in 'field list'"}
	errGone     = &mysql.MySQLError{Number: 2013, Message: "Lost connection to server during query"}
	errNotMySQL = errors.New("driver: bad connection")
)

const setupInstruments = "performance_schema.setup_instruments"

// server is a fake server: its version, the error of each statement that
// fails, and the rows of SHOW GLOBAL VARIABLES and STATUS.
type server struct {
	version   string
	versionOK bool             // false: the version query fails
	errors    map[string]error // statement: error
	variables [][]driver.Value // rows for SHOW GLOBAL VARIABLES
	status    [][]driver.Value // rows for SHOW GLOBAL STATUS
	columns   []string         // SHOW columns, default Variable_name, Value
}

func (s server) handle(st fakesql.Statement) fakesql.Result {
	if err := s.errors[st.Query]; err != nil {
		return fakesql.Result{Err: err}
	}
	columns := s.columns
	if columns == nil {
		columns = []string{"Variable_name", "Value"}
	}
	switch {
	case st.Query == "SELECT @@version":
		if !s.versionOK {
			return fakesql.Result{Err: errGone}
		}
		return fakesql.Result{Columns: []string{"@@version"}, Rows: [][]driver.Value{{s.version}}}
	case strings.HasPrefix(st.Query, "SHOW GLOBAL VARIABLES"):
		return fakesql.Result{Columns: columns, Rows: s.variables}
	case strings.HasPrefix(st.Query, "SHOW GLOBAL STATUS"):
		return fakesql.Result{Columns: columns, Rows: s.status}
	}
	return fakesql.Result{} // candidate and update probes succeed with no rows
}

// queries returns the text of each statement run.
func queries(db *fakesql.DB) []string {
	var out []string
	for _, st := range db.Statements() {
		out = append(out, st.Query)
	}
	return out
}

func TestRunMySQL(t *testing.T) {
	db := fakesql.OpenFunc(server{
		version:   "9.7.2",
		versionOK: true,
		variables: [][]driver.Value{{"VERSION_COMMENT", "MySQL Community Server - GPL"}, {"datadir", "/var/lib/mysql/"}},
		status:    [][]driver.Value{{"Uptime", "1234"}},
	}.handle)
	s, err := Run(db.DB, Request{
		Candidates: processlist,
		Variables:  []string{"datadir", "Version_Comment", "relay_log_basename"},
		Status:     []string{"Uptime"},
		Updates:    []Update{{Table: "Performance_Schema.Setup_Instruments", Column: "ENABLED"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := &Server{
		Flavour:   MySQL,
		Version:   "9.7.2",
		Usable:    map[string]bool{"ps": true, "is": true},
		Unusable:  map[string]string{},
		Updatable: map[string]bool{setupInstruments: true},
		Denied:    map[string]string{},
		Variables: map[string]string{"version_comment": "MySQL Community Server - GPL", "datadir": "/var/lib/mysql/"},
		Status:    map[string]string{"uptime": "1234"},
	}
	if !reflect.DeepEqual(s, want) {
		t.Errorf("got  %+v\nwant %+v", s, want)
	}
	wantQueries := []string{
		"SELECT @@version",
		"SELECT * FROM (SELECT ID FROM performance_schema.processlist) AS p LIMIT 0",
		"SELECT * FROM (SELECT ID FROM information_schema.processlist) AS p LIMIT 0",
		"UPDATE performance_schema.setup_instruments SET enabled = enabled WHERE 1=0",
		"SHOW GLOBAL VARIABLES WHERE Variable_name IN ('datadir', 'version_comment', 'relay_log_basename')",
		"SHOW GLOBAL STATUS WHERE Variable_name IN ('uptime')",
	}
	if got := queries(db); !reflect.DeepEqual(got, wantQueries) {
		t.Errorf("statements:\n got  %q\n want %q", got, wantQueries)
	}
}

func TestRunMariaDB(t *testing.T) {
	db := fakesql.OpenFunc(server{
		version:   "12.3.3-MariaDB-ubu2404-log",
		versionOK: true,
		errors: map[string]error{
			"SELECT * FROM (SELECT ID FROM performance_schema.processlist) AS p LIMIT 0":  errTable,
			"UPDATE performance_schema.setup_instruments SET enabled = enabled WHERE 1=0": errDenied,
		},
	}.handle)
	s, err := Run(db.DB, Request{Candidates: processlist, Updates: []Update{{Table: setupInstruments, Column: "enabled"}}})
	if err != nil {
		t.Fatal(err)
	}
	if s.Flavour != MariaDB {
		t.Errorf("flavour %v, want MariaDB", s.Flavour)
	}
	if !reflect.DeepEqual(s.Usable, map[string]bool{"is": true}) ||
		!reflect.DeepEqual(s.Unusable, map[string]string{"ps": errTable.Error()}) {
		t.Errorf("usable %v, unusable %v", s.Usable, s.Unusable)
	}
	if !reflect.DeepEqual(s.Updatable, map[string]bool{setupInstruments: false}) ||
		!reflect.DeepEqual(s.Denied, map[string]string{setupInstruments: errDenied.Error()}) {
		t.Errorf("updatable %v, denied %v", s.Updatable, s.Denied)
	}
	if len(s.Variables) != 0 || len(s.Status) != 0 {
		t.Errorf("variables %v, status %v, want none", s.Variables, s.Status)
	}
	for _, q := range queries(db) {
		if strings.HasPrefix(q, "SHOW") {
			t.Errorf("%q run with no names asked for", q)
		}
	}
}

func TestRunCandidateErrors(t *testing.T) {
	const probeSQL = "SELECT * FROM (SELECT ID FROM performance_schema.processlist) AS p LIMIT 0"
	for _, e := range []*mysql.MySQLError{errTable, errDenied, errColumn, errField} {
		db := fakesql.OpenFunc(server{version: "8.4.8", versionOK: true, errors: map[string]error{probeSQL: e}}.handle)
		s, err := Run(db.DB, Request{Candidates: processlist[:1]})
		if err != nil {
			t.Errorf("%d: %v, want the candidate unusable", e.Number, err)
			continue
		}
		if s.Usable["ps"] || s.Unusable["ps"] != e.Error() {
			t.Errorf("%d: usable %v, unusable %v", e.Number, s.Usable, s.Unusable)
		}
	}
	for _, e := range []error{errGone, errNotMySQL} {
		db := fakesql.OpenFunc(server{version: "8.4.8", versionOK: true, errors: map[string]error{probeSQL: e}}.handle)
		_, err := Run(db.DB, Request{Candidates: processlist[:1]})
		if err == nil || !errors.Is(err, e) || !strings.HasPrefix(err.Error(), "probe: ps: ") {
			t.Errorf("%v: got %v, want it returned for candidate ps", e, err)
		}
	}
}

func TestRunUpdateErrors(t *testing.T) {
	const updateSQL = "UPDATE performance_schema.setup_instruments SET enabled = enabled WHERE 1=0"
	update := []Update{{Table: setupInstruments, Column: "enabled"}}
	for _, e := range []*mysql.MySQLError{errDenied, errColumn, errTable} {
		db := fakesql.OpenFunc(server{version: "8.4.8", versionOK: true, errors: map[string]error{updateSQL: e}}.handle)
		s, err := Run(db.DB, Request{Updates: update})
		if err != nil {
			t.Errorf("%d: %v, want the update denied", e.Number, err)
			continue
		}
		if s.Updatable[setupInstruments] || s.Denied[setupInstruments] != e.Error() {
			t.Errorf("%d: updatable %v, denied %v", e.Number, s.Updatable, s.Denied)
		}
	}
	for _, e := range []error{errField, errGone, errNotMySQL} {
		db := fakesql.OpenFunc(server{version: "8.4.8", versionOK: true, errors: map[string]error{updateSQL: e}}.handle)
		_, err := Run(db.DB, Request{Updates: update})
		if err == nil || !errors.Is(err, e) || !strings.HasPrefix(err.Error(), "probe: update "+setupInstruments+": ") {
			t.Errorf("%v: got %v, want it returned for the update", e, err)
		}
	}
}

func TestRunErrors(t *testing.T) {
	ok := server{version: "8.4.8", versionOK: true}
	tests := []struct {
		name   string
		server server
		req    Request
		want   string
	}{
		{"version fails", server{}, Request{}, "probe: version: "},
		{"duplicate candidate", ok, Request{Candidates: []Candidate{processlist[0], processlist[0]}}, `probe: duplicate candidate name "ps"`},
		{"invalid update table", ok, Request{Updates: []Update{{Table: "setup_instruments", Column: "enabled"}}}, "probe: invalid update"},
		{"invalid variable name", ok, Request{Variables: []string{"a'b"}}, `probe: invalid name "a'b"`},
		{"show variables fails", server{version: "8.4.8", versionOK: true, errors: map[string]error{
			"SHOW GLOBAL VARIABLES WHERE Variable_name IN ('datadir')": errGone}},
			Request{Variables: []string{"datadir"}}, "probe: show VARIABLES: "},
		{"show status scan fails", server{version: "8.4.8", versionOK: true, columns: []string{"Variable_name"},
			status: [][]driver.Value{{"uptime"}}},
			Request{Status: []string{"uptime"}}, "probe: show STATUS: "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := Run(fakesql.OpenFunc(tt.server.handle).DB, tt.req)
			if err == nil || !strings.HasPrefix(err.Error(), tt.want) {
				t.Errorf("got %+v, %v, want error starting %q", s, err, tt.want)
			}
		})
	}
}

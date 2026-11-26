// Package probe finds out once, at connect, what the server provides: flavour,
// version, which metric queries run, which tables may be updated, and the
// values of named variables and status counters. Metrics use the result to
// choose their query.
package probe

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// Flavour is the server family.
type Flavour int

// Flavours.
const (
	MySQL Flavour = iota
	MariaDB
)

func (f Flavour) String() string {
	if f == MariaDB {
		return "MariaDB"
	}
	return "MySQL"
}

// Server error numbers the probe tells apart.
const (
	errNoSuchTable       = 1146 // ER_NO_SUCH_TABLE
	errTableAccessDenied = 1142 // ER_TABLEACCESS_DENIED_ERROR
	errColumnAccessDeny  = 1143 // ER_COLUMNACCESS_DENIED_ERROR
	errBadField          = 1054 // ER_BAD_FIELD_ERROR: unknown column
)

// showKind is what SHOW GLOBAL lists.
type showKind string

const (
	showVariables showKind = "VARIABLES"
	showStatus    showKind = "STATUS"
)

// Update is a table that must be updatable, checked with
// UPDATE <Table> SET <Column> = <Column> WHERE 1=0.
type Update struct {
	Table  string // "schema.table"
	Column string
}

// Request lists what to probe. Table names are "schema.table".
type Request struct {
	Candidates []Candidate // each query is run, returning no rows
	Variables  []string    // SHOW GLOBAL VARIABLES
	Status     []string    // SHOW GLOBAL STATUS
	Updates    []Update    // UPDATE privilege
}

// Server is the probe result. Table names are lower case.
type Server struct {
	Flavour   Flavour
	Version   string
	Usable    map[string]bool   // candidate name: its query runs
	Unusable  map[string]string // candidate name: server error from its query
	Updatable map[string]bool   // UPDATE allowed, for requested updates
	Denied    map[string]string // table: server error from the UPDATE check
	Variables map[string]string // only those requested; missing ones absent
	Status    map[string]string // only those requested; missing ones absent
}

// Candidate is one way to read a metric. Names are unique across a Request.
// Query must be valid as a derived table: no duplicate column names.
type Candidate struct {
	Name  string
	Query string
}

var (
	reName  = regexp.MustCompile(`^[a-z0-9_]+$`)
	reTable = regexp.MustCompile(`^[a-z0-9_]+\.[a-z0-9_]+$`)
)

// lower returns name in lower case. performance_schema names must be lower
// case on MySQL with lower_case_table_names=0 (an upper case schema is denied,
// an upper case table does not exist); information_schema accepts any case.
func lower(name string) string { return strings.ToLower(name) }

// queryProbeSQL returns query wrapped so that it returns no rows, whatever its
// own clauses. Query is a metric definition from the code, not user input.
func queryProbeSQL(query string) string {
	return "SELECT * FROM (" + query + ") AS p LIMIT 0"
}

// updateProbeSQL returns a statement that updates no rows of table.
// Concatenation is safe: table and column must match reTable and reName.
func updateProbeSQL(table, column string) (string, error) {
	if !reTable.MatchString(table) || !reName.MatchString(column) {
		return "", fmt.Errorf("probe: invalid update %q.%q", table, column)
	}
	return "UPDATE " + table + " SET " + column + " = " + column + " WHERE 1=0", nil
}

// showWhereSQL returns SHOW GLOBAL <kind> limited to names.
// Concatenation is safe: kind is a constant and each name must match reName.
func showWhereSQL(kind showKind, names []string) (string, error) {
	quoted := make([]string, len(names))
	for i, n := range names {
		if !reName.MatchString(n) {
			return "", fmt.Errorf("probe: invalid name %q", n)
		}
		quoted[i] = "'" + n + "'"
	}
	return "SHOW GLOBAL " + string(kind) + " WHERE Variable_name IN (" + strings.Join(quoted, ", ") + ")", nil
}

// Run probes db once.
func Run(db *sql.DB, req Request) (*Server, error) {
	s := &Server{
		Usable:    map[string]bool{},
		Unusable:  map[string]string{},
		Updatable: map[string]bool{},
		Denied:    map[string]string{},
	}
	if err := db.QueryRow("SELECT @@version").Scan(&s.Version); err != nil {
		return nil, fmt.Errorf("probe: version: %w", err)
	}
	if strings.Contains(s.Version, "MariaDB") {
		s.Flavour = MariaDB
	}
	seen := map[string]bool{}
	for _, c := range req.Candidates {
		if seen[c.Name] {
			return nil, fmt.Errorf("probe: duplicate candidate name %q", c.Name)
		}
		seen[c.Name] = true
		if err := s.probeCandidate(db, c); err != nil {
			return nil, err
		}
	}
	for _, u := range req.Updates {
		if err := s.probeUpdate(db, lower(u.Table), lower(u.Column)); err != nil {
			return nil, err
		}
	}
	var err error
	if s.Variables, err = show(db, showVariables, req.Variables); err != nil {
		return nil, err
	}
	if s.Status, err = show(db, showStatus, req.Status); err != nil {
		return nil, err
	}
	return s, nil
}

// probeCandidate records whether c's query runs: a missing table or column,
// or denied access, makes it unusable. Without a grant MariaDB denies before
// checking that the table exists. Unexpected errors are returned.
func (s *Server) probeCandidate(db *sql.DB, c Candidate) error {
	_, err := db.Exec(queryProbeSQL(c.Query))
	switch number(err) {
	case 0:
		s.Usable[c.Name] = true
	case errNoSuchTable, errTableAccessDenied, errColumnAccessDeny, errBadField:
		s.Unusable[c.Name] = err.Error()
	default:
		return fmt.Errorf("probe: %s: %w", c.Name, err)
	}
	return nil
}

func (s *Server) probeUpdate(db *sql.DB, table, column string) error {
	q, err := updateProbeSQL(table, column)
	if err != nil {
		return err
	}
	_, err = db.Exec(q)
	switch number(err) {
	case 0:
		s.Updatable[table] = true
	case errTableAccessDenied, errColumnAccessDeny, errNoSuchTable:
		s.Updatable[table] = false
		s.Denied[table] = err.Error()
	default:
		return fmt.Errorf("probe: update %s: %w", table, err)
	}
	return nil
}

// number returns the server error number of err, 0 for nil, -1 if err is not
// a server error.
func number(err error) int {
	if err == nil {
		return 0
	}
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return int(me.Number)
	}
	return -1
}

// show returns SHOW GLOBAL <kind> for the named entries only, keyed in lower case.
func show(db *sql.DB, kind showKind, names []string) (map[string]string, error) {
	out := map[string]string{}
	if len(names) == 0 {
		return out, nil
	}
	lowered := make([]string, len(names))
	for i, n := range names {
		lowered[i] = lower(n)
	}
	q, err := showWhereSQL(kind, lowered)
	if err != nil {
		return nil, err
	}
	rs, err := db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("probe: show %s: %w", kind, err)
	}
	defer func() { _ = rs.Close() }()
	for rs.Next() {
		var name, value string
		if err := rs.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("probe: show %s: %w", kind, err)
		}
		out[lower(name)] = value
	}
	return out, rs.Err()
}

// Choose returns the first candidate whose query runs.
func (s *Server) Choose(candidates []Candidate) (Candidate, error) {
	var why []string
	for _, c := range candidates {
		switch {
		case s.Usable[c.Name]:
			return c, nil
		case s.Unusable[c.Name] != "":
			why = append(why, c.Name+": "+s.Unusable[c.Name])
		default:
			why = append(why, c.Name+": not probed")
		}
	}
	return Candidate{}, fmt.Errorf("probe: no usable query on %v %s (%s)", s.Flavour, s.Version, strings.Join(why, "; "))
}

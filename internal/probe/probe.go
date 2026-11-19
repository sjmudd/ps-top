// Package probe finds out once, at connect, what the server provides: flavour,
// version, which tables exist and may be read or updated, and the values of
// named variables and status counters. Metrics use the result to choose their
// query.
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

// Request lists what to probe. Names are "schema.table".
type Request struct {
	Tables    []string // existence and SELECT privilege
	Variables []string // SHOW GLOBAL VARIABLES
	Status    []string // SHOW GLOBAL STATUS
	Updates   []Update // UPDATE privilege
}

// Server is the probe result. Table names are lower case.
type Server struct {
	Flavour   Flavour
	Version   string
	Tables    map[string]bool   // true: exists; false: does not exist; absent: unknown (SELECT denied)
	Readable  map[string]bool   // SELECT allowed
	Updatable map[string]bool   // UPDATE allowed, for requested updates
	Denied    map[string]string // table: server error from the SELECT or UPDATE check
	Variables map[string]string // only those requested; missing ones absent
	Status    map[string]string // only those requested; missing ones absent
}

// Candidate is one way to read a metric.
type Candidate struct {
	Name  string
	Needs []string // tables, "schema.table"
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

// selectProbeSQL returns a statement that reads no rows from table.
// Concatenation is safe: table must match reTable.
func selectProbeSQL(table string) (string, error) {
	if !reTable.MatchString(table) {
		return "", fmt.Errorf("probe: invalid table name %q", table)
	}
	return "SELECT 1 FROM " + table + " LIMIT 0", nil
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
		Tables:    map[string]bool{},
		Readable:  map[string]bool{},
		Updatable: map[string]bool{},
		Denied:    map[string]string{},
	}
	if err := db.QueryRow("SELECT @@version").Scan(&s.Version); err != nil {
		return nil, fmt.Errorf("probe: version: %w", err)
	}
	if strings.Contains(s.Version, "MariaDB") {
		s.Flavour = MariaDB
	}
	for _, t := range req.Tables {
		if err := s.probeTable(db, lower(t)); err != nil {
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

// probeTable records whether table exists and may be read. Unexpected errors
// are returned.
func (s *Server) probeTable(db *sql.DB, table string) error {
	q, err := selectProbeSQL(table)
	if err != nil {
		return err
	}
	_, err = db.Exec(q)
	switch number(err) {
	case 0:
		s.Tables[table] = true
		s.Readable[table] = true
	case errNoSuchTable:
		s.Tables[table] = false
	case errTableAccessDenied, errColumnAccessDeny:
		// Existence unknown: without a grant MariaDB denies before checking
		// that the table exists.
		s.Denied[table] = err.Error()
	default:
		return fmt.Errorf("probe: %s: %w", table, err)
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

// Choose returns the first candidate whose tables all exist and are readable.
func (s *Server) Choose(candidates []Candidate) (Candidate, error) {
	var why []string
	for _, c := range candidates {
		var lacks []string
		for _, t := range c.Needs {
			t = lower(t)
			switch {
			case s.Readable[t]:
			case s.Denied[t] != "":
				lacks = append(lacks, "no SELECT on "+t+" (existence unknown)")
			default:
				lacks = append(lacks, "no "+t)
			}
		}
		if len(lacks) == 0 {
			return c, nil
		}
		why = append(why, c.Name+": "+strings.Join(lacks, ", "))
	}
	return Candidate{}, fmt.Errorf("probe: no usable query on %v %s (%s)", s.Flavour, s.Version, strings.Join(why, "; "))
}

// Needs returns every table named by the candidates, for Request.Tables.
func Needs(candidates ...[]Candidate) []string {
	seen := map[string]bool{}
	var out []string
	for _, cs := range candidates {
		for _, c := range cs {
			for _, t := range c.Needs {
				t = lower(t)
				if !seen[t] {
					seen[t] = true
					out = append(out, t)
				}
			}
		}
	}
	return out
}

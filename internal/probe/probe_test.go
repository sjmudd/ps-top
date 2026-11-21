package probe

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
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

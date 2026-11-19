package probe

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
)

var processlist = []Candidate{
	{Name: "ps", Needs: []string{"performance_schema.processlist"}, Query: "SELECT ... FROM performance_schema.processlist"},
	{Name: "is", Needs: []string{"information_schema.PROCESSLIST"}, Query: "SELECT ... FROM information_schema.processlist"},
}

func TestChoose(t *testing.T) {
	both := map[string]bool{"performance_schema.processlist": true, "information_schema.processlist": true}
	mysql := &Server{Version: "9.7.2", Tables: both, Readable: both}
	is := map[string]bool{"performance_schema.processlist": false, "information_schema.processlist": true}
	mariadb := &Server{Flavour: MariaDB, Version: "12.3.3-MariaDB", Tables: is, Readable: is}
	if c, err := mysql.Choose(processlist); err != nil || c.Name != "ps" {
		t.Errorf("MySQL: %v %v, want ps", c.Name, err)
	}
	if c, err := mariadb.Choose(processlist); err != nil || c.Name != "is" {
		t.Errorf("MariaDB: %v %v, want is", c.Name, err)
	}
}

func TestChooseNone(t *testing.T) {
	s := &Server{Version: "8.0.46", Tables: map[string]bool{}}
	_, err := s.Choose(processlist)
	if err == nil {
		t.Fatal("no tables: no error")
	}
	for _, want := range []string{"8.0.46", "ps: no performance_schema.processlist", "is: no information_schema.processlist"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}

func TestChooseSkipsUnreadable(t *testing.T) {
	// P_S table present but SELECT denied: fall back to I_S.
	s := &Server{Version: "9.7.2",
		Tables:   map[string]bool{"information_schema.processlist": true},
		Readable: map[string]bool{"information_schema.processlist": true},
		Denied:   map[string]string{"performance_schema.processlist": "Error 1142"},
	}
	if c, err := s.Choose(processlist); err != nil || c.Name != "is" {
		t.Errorf("got %v %v, want is", c.Name, err)
	}
	s.Readable = map[string]bool{}
	_, err := s.Choose(processlist)
	if err == nil || !strings.Contains(err.Error(), "no SELECT on performance_schema.processlist (existence unknown)") {
		t.Errorf("error = %v, want no SELECT, existence unknown", err)
	}
}

func TestNeeds(t *testing.T) {
	other := []Candidate{{Name: "x", Needs: []string{"performance_schema.processlist", "performance_schema.file_summary_by_instance"}}}
	got := Needs(processlist, other)
	want := []string{"performance_schema.processlist", "information_schema.processlist", "performance_schema.file_summary_by_instance"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Needs = %v, want %v", got, want)
	}
}

func TestSQLBuilders(t *testing.T) {
	if q, err := selectProbeSQL("performance_schema.processlist"); err != nil || q != "SELECT 1 FROM performance_schema.processlist LIMIT 0" {
		t.Errorf("selectProbeSQL = %q, %v", q, err)
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
		if _, err := selectProbeSQL(bad); err == nil {
			t.Errorf("selectProbeSQL(%q): no error", bad)
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

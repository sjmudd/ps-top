// Package filename contains the routines for
// managing the file_summary_by_instance table.
package filename

import (
	"testing"
)

type testConfig map[string]string

// return the value if found, if not an empty string
func (tc testConfig) Get(name string) string {
	if val, found := tc[name]; found {
		return val
	}
	return ""
}

// noopMunger returns the input string (does nothing)
func noopMunger(filename string) string {
	return filename
}

// qualitifedNamer returns <schema>, <table> as "<schema>.<table>"
func qualifiedNamer(schema, table string) string {
	return schema + "." + table
}

func TestSimplify(t *testing.T) {
	const (
		datadir  = "/path/to/datadir/"
		relaylog = "otused"
	)
	var tests = []struct {
		path     string
		expected string
	}{
		{`/path/to/@0024`, `/path/to/$`},
		{`/path/to/datadir/#sql-12345.ibd`, `<temp_table>`},
		{`/path/to/datadir/#innodb_temp/temp_6.ibt`, `<temp_table>`},
		{`/path/to/datadir/`, `<datadir>/`},
		{`/path/to/datadir/whatever`, `<datadir>/whatever`},
		{`/path/to/datadir/auto.cnf`, `<auto_cnf>`},
		{`/path/to/datadir//auto.cnf`, `<auto_cnf>`},
		{`/path/to/datadir/./auto.cnf`, `<auto_cnf>`},
		{`/path/to/datadir/somedb/sometable.frm`, `somedb.sometable`},
		{`/path/to/datadir/somedb/sometable.ibd`, `somedb.sometable`},
		{`/path/to/datadir/somedb/sometable.MYD`, `somedb.sometable`},
		{`/path/to/datadir/somedb/sometable.MYI`, `somedb.sometable`},
		{`/path/to/datadir/somedb/sometable.CSM`, `somedb.sometable`},
		{`/path/to/datadir/somedb/sometable.CSV`, `somedb.sometable`},
		{`/path/to/datadir/somedb/sometable.par`, `somedb.sometable`},
		{`/path/to/datadir/somedb/sometable#P#p0001.ibd`, `somedb.sometable`},
		{`/path/to/datadir/somedb/sometable#P#pMAX.ibd`, `somedb.sometable`},
		{`/path/to/datadir/#ib_16384_0.dblwr`, `<doublewrite>`},
		{`/path/to/datadir/ibdata1`, `<ibdata>`},
		{`/path/to/datadir/ibtmp000`, `<ibtmp>`},
		{`/path/to/datadir/ib_logfile1`, `<redo_log>`},
		{`/path/to/datadir/#innodb_redo/#ib_redo4226`, `<redo_log>`},
		{`/path/to/datadir/undo_0000`, `<undo_log>`},
		{`/path/to/datadir/binlog.123456`, `<binlog>`},
		{`/path/to/datadir/binlog.index`, `<binlog>`},
		{`/path/to/datadir/db.opt`, `<db_opt>`},
		{`/path/to/datadir/slowlog`, `<slow_log>`},
		{`/path/to/datadir/xxxx.pid`, `<pid_file>`},
		{`/path/to/share/whatver/errmsg.sys`, `<errmsg>`},
		{`/path/to/share/charsets/Index.xml`, `<charset>`},
	}

	for _, test := range tests {
		got := uncachedSimplify(test.path, noopMunger, qualifiedNamer, datadir, relaylog)
		if got != test.expected {
			t.Errorf("uncachedSimplify(%q) != expected %q, got: %q", test.path, test.expected, got)
		}
	}
}

// Package file_io contains the routines for
// managing the file_summary_by_instance table.
package file_io

import (
	"testing"

	"github.com/sjmudd/anonymiser"
)

type testGetter map[string]string

// return the value if found, if not an empty string
func (tg testGetter) Get(name string) string {
	if val, found := tg[name]; found {
		return val
	}
	return ""
}

func TestSimplify(t *testing.T) {
	var globalVariables = testGetter{
		"datadir": "/some/path/to/datadir/",
	}
	var tests = []struct {
		path     string
		expected string
	}{
		{`/some/path/to/@0024`, `/some/path/to/$`},
		{`/some/path/to/datadir/#sql-12345.ibd`, `<temp_table>`},
		{`/some/path/to/datadir/`, `<datadir>/`},
		{`/some/path/to/datadir/whatever`, `<datadir>/whatever`},
		{`/some/path/to/datadir/auto.cnf`, `<auto_cnf>`},
		{`/some/path/to/datadir//auto.cnf`, `<auto_cnf>`},
		{`/some/path/to/datadir/./auto.cnf`, `<auto_cnf>`},
		{`/some/path/to/datadir/somedb/sometable.frm`, `somedb.sometable`},
		{`/some/path/to/datadir/somedb/sometable.ibd`, `somedb.sometable`},
		{`/some/path/to/datadir/somedb/sometable.MYD`, `somedb.sometable`},
		{`/some/path/to/datadir/somedb/sometable.MYI`, `somedb.sometable`},
		{`/some/path/to/datadir/somedb/sometable.CSM`, `somedb.sometable`},
		{`/some/path/to/datadir/somedb/sometable.CSV`, `somedb.sometable`},
		{`/some/path/to/datadir/somedb/sometable.par`, `somedb.sometable`},
		{`/some/path/to/datadir/somedb/sometable#P#p0001.ibd`, `somedb.sometable`},
		{`/some/path/to/datadir/somedb/sometable#P#pMAX.ibd`, `somedb.sometable`},
		{`/some/path/to/datadir/#ib_16384_0.dblwr`, `<doublewrite>`},
		{`/some/path/to/datadir/ibdata1`, `<ibdata>`},
		{`/some/path/to/datadir/ibtmp000`, `<ibtmp>`},
		{`/some/path/to/datadir/ib_logfile1`, `<redo_log>`},
		{`/some/path/to/datadir/undo_0000`, `<undo_log>`},
		{`/some/path/to/datadir/binlog.123456`, `<binlog>`},
		{`/some/path/to/datadir/binlog.index`, `<binlog>`},
		{`/some/path/to/datadir/db.opt`, `<db_opt>`},
		{`/some/path/to/datadir/slowlog`, `<slow_log>`},
		{`/some/path/to/datadir/xxxx.pid`, `<pid_file>`},
		{`/some/path/to/share/whatver/errmsg.sys`, `<errmsg>`},
		{`/some/path/to/share/charsets/Index.xml`, `<charset>`},
	}

	anonymiser.Enable(false) // we don't want to anonymise tablenames

	for _, test := range tests {
		got := uncachedSimplify(test.path, globalVariables)
		if got != test.expected {
			t.Errorf("uncachedSimplify(%q) != expected %q, got: %q", test.path, test.expected, got)
		}
	}
}

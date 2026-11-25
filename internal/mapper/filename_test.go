package mapper

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sjmudd/ps-top/internal/snapshot"
)

const dd = "/var/lib/mysql/"

func fileName(t *testing.T, s FileNameSettings) *FileName {
	t.Helper()
	m, err := NewFileName(s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMap(t *testing.T) {
	m := fileName(t, FileNameSettings{
		DataDir:      dd,
		BinlogBase:   dd + "mysqld-bin",
		RelayLogBase: dd + "host1-relay-bin",
	})
	tests := []struct{ path, want string }{
		// tables
		{dd + "sbtest/sbtest1.ibd", "sbtest.sbtest1"},
		{dd + "sbtest/sbtest1.frm", "sbtest.sbtest1"},
		{dd + "mysql/db.MAD", "mysql.db"},
		{dd + "mysql/db.MAI", "mysql.db"},
		{dd + "sbtest/t.MYD", "sbtest.t"},
		{dd + "sbtest/t.sdi", "sbtest.t"},
		{dd + "sbtest/ptest.par", "sbtest.ptest"},
		{dd + "sbtest/ptest#P#p2024.ibd", "sbtest.ptest"},           // MariaDB
		{dd + "sbtest/ptest#P#pmax.ibd", "sbtest.ptest"},            // MariaDB
		{dd + "sbtest/ptest#p#p2024.ibd", "sbtest.ptest"},           // MySQL 8.0+
		{dd + "sbtest/sptest#P#p0#SP#p0sp1.ibd", "sbtest.sptest"},   // MariaDB
		{dd + "sbtest/sptest#p#p0#sp#p0sp1.ibd", "sbtest.sptest"},   // MySQL 8.0+
		{dd + "sbtest/my@002dtable@0024x.ibd", "sbtest.my-table$x"}, // @XXXX decoded
		{"/data/elsewhere/shop/orders.ibd", "shop.orders"},          // DATA DIRECTORY
		{dd + "sbtest//sbtest1.ibd", "sbtest.sbtest1"},              // path cleanup
		{dd + "x/../sbtest/./sbtest1.ibd", "sbtest.sbtest1"},
		// temp tables
		{dd + "#innodb_temp/temp_7.ibt", "<temp_table>"},
		{dd + "sbtest/#sql-4c5_2.ibd", "<temp_table>"},
		{"/tmp/#sql-ib123-456.ibd", "<temp_table>"},
		// server files
		{dd + "mysql.ibd", "<data_dictionary>"},
		{dd + "ibdata1", "<ibdata>"},
		{dd + "ibtmp1", "<ibtmp>"},
		{dd + "undo_001", "<undo_log>"},
		{dd + "undo002", "<undo_log>"},
		{dd + "ib_logfile0", "<redo_log>"},
		{dd + "#innodb_redo/#ib_redo12", "<redo_log>"},
		{dd + "#innodb_redo/#ib_redo13_tmp", "<redo_log>"},
		{dd + "#ib_16384_0.dblwr", "<doublewrite>"},
		{dd + "aria_log.00000001", "<aria_log>"},
		{dd + "aria_log_control", "<aria_log>"},
		{dd + "ddl_recovery.log", "<ddl_log>"},
		{"/tmp/Innodb Merge Temp File", "<innodb_merge_temp>"},
		{dd + "sbtest/db.opt", "<db_opt>"},
		{dd + "auto.cnf", "<auto_cnf>"},
		{dd + "slowlog", "<slow_log>"},
		{"/run/mysqld/mysqld.pid", "<pid_file>"},
		{"/usr/share/mysql-8.0/english/errmsg.sys", "<errmsg>"},
		{"/usr/share/mysql-8.0/charsets/Index.xml", "<charset>"},
		{"/usr/share/mariadb/charsets/Index.xml", "<charset>"},
		{dd + "mysqld-bin.000002", "<binlog>"},
		{dd + "mysqld-bin.000002.idx", "<binlog>"},
		{dd + "mysqld-bin.index", "<binlog>"},
		{dd + "mysqld-bin.state", "<binlog>"},
		{dd + "host1-relay-bin.000003", "<relay_log>"},
		{dd + "host1-relay-bin.index", "<relay_log>"},
		// unknown
		{dd + "whatever", "<datadir>/whatever"},
		{"/somewhere/else", "/somewhere/else"},
	}
	for _, tc := range tests {
		if got := snapshot.KeyText(m.Map(tc.path)); got != tc.want {
			t.Errorf("Map(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// TestMapTableParts checks a table file maps to a key of two parts, schema
// and table, even when a name contains ".".
func TestMapTableParts(t *testing.T) {
	m := fileName(t, FileNameSettings{DataDir: dd})
	tests := []struct {
		path string
		want []string
	}{
		{dd + "shop/orders.ibd", []string{"shop", "orders"}},
		{dd + "my@002eshop/order@002elines.ibd", []string{"my.shop", "order.lines"}},
		{dd + "ibdata1", []string{"<ibdata>"}},
	}
	for _, tc := range tests {
		if got := snapshot.KeyParts(m.Map(tc.path)); !slices.Equal(got, tc.want) {
			t.Errorf("Map(%q) parts = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// TestMapRelativeLogs checks binlog and relay log base names set relative
// to the datadir, as the server reports them, e.g. "../log/binlog", match
// the files performance_schema names, resolved against the datadir.
func TestMapRelativeLogs(t *testing.T) {
	for _, s := range []FileNameSettings{
		{DataDir: dd, BinlogBase: "../log/binlog", RelayLogBase: "../log/relay"},
		{DataDir: dd, BinlogBase: "../log/binlog", RelayLog: "../log/relay"},
	} {
		m := fileName(t, s)
		for path, want := range map[string]string{
			"/var/lib/log/binlog.000001": "<binlog>",
			"/var/lib/log/binlog.index":  "<binlog>",
			"/var/lib/log/binlog.state":  "<binlog>",
			"/var/lib/log/relay.000002":  "<relay_log>",
			"/var/lib/log/relay.index":   "<relay_log>",
			"/var/lib/log/other.000001":  "/var/lib/log/other.000001",
		} {
			if got := snapshot.KeyText(m.Map(path)); got != want {
				t.Errorf("%+v: Map(%q) = %q, want %q", s, path, got, want)
			}
		}
	}
}

// TestMapMunge checks Munge renames "schema.table" names only, split again
// at the first ".", and a name without a "." keeps its schema.
func TestMapMunge(t *testing.T) {
	m := fileName(t, FileNameSettings{DataDir: dd, Munge: func(s string) string {
		switch s {
		case "shop.orders_20260101":
			return "shop.orders_YYYYMMDD"
		case "shop.all":
			return "everything"
		}
		return s + "!"
	}})
	tests := []struct {
		path string
		want []string
	}{
		{dd + "shop/orders_20260101.ibd", []string{"shop", "orders_YYYYMMDD"}},
		{dd + "shop/all.ibd", []string{"shop", "everything"}},
		{dd + "shop/t.ibd", []string{"shop", "t!"}},
		{dd + "ibdata1", []string{"<ibdata>"}},
		{dd + "other.file", []string{"<datadir>/other.file"}},
	}
	for _, tc := range tests {
		if got := snapshot.KeyParts(m.Map(tc.path)); !slices.Equal(got, tc.want) {
			t.Errorf("Map(%q) parts = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestSettingsAreLiteral(t *testing.T) {
	// "." in a setting matches only ".", and regexp metacharacters do not panic.
	m := fileName(t, FileNameSettings{DataDir: "/data/my.sql+(1)/", BinlogBase: "/data/my.sql+(1)/bin.log"})
	if got := m.Map("/data/my.sql+(1)/bin.log.000001"); got != "<binlog>" {
		t.Errorf("binlog: got %q", got)
	}
	if got := m.Map("/data/myXsql+(1)/bin.log.000001"); got == "<binlog>" {
		t.Error("binlog base matched with . as a wildcard")
	}
	if got := m.Map("/data/my.sql+(1)/notes"); got != "<datadir>/notes" {
		t.Errorf("datadir: got %q", got)
	}
}

// TestRelayLogAndInfoFiles checks the relay log base name is found from
// whichever setting the server gives, and the replication info files are
// labelled, with settings and names as the replication harness shows them.
func TestRelayLogAndInfoFiles(t *testing.T) {
	tests := []struct {
		name string
		s    FileNameSettings
		maps map[string]string
	}{
		{"MySQL 8.4 replica: relay_log_basename", FileNameSettings{
			DataDir: dd, RelayLogBase: dd + "replica1-relay-bin", RelayLog: "replica1-relay-bin", PidFile: "/var/run/mysqld/mysqld.pid",
		}, map[string]string{
			dd + "replica1-relay-bin.000002": "<relay_log>",
			dd + "replica1-relay-bin.index":  "<relay_log>",
			dd + "mysqld-relay-bin.000001":   "<datadir>/mysqld-relay-bin.000001", // the basename wins
		}},
		{"relay_log only, relative", FileNameSettings{DataDir: dd, RelayLog: "replica1-relay-bin"}, map[string]string{
			dd + "replica1-relay-bin.000002": "<relay_log>",
		}},
		{"relay_log with an extension, as MariaDB strips it", FileNameSettings{DataDir: dd, RelayLog: "/logs/relay.log"}, map[string]string{
			"/logs/relay.000001": "<relay_log>",
		}},
		{"MariaDB 12.3 replica: named after the pid file", FileNameSettings{
			DataDir: dd, PidFile: "/run/mysqld/mysqld.pid", MasterInfoFile: "master.info", RelayLogInfoFile: "relay-log.info",
		}, map[string]string{
			dd + "mysqld-relay-bin.000001": "<relay_log>",
			dd + "mysqld-relay-bin.index":  "<relay_log>",
			dd + "master.info":             "<master.info>",
			dd + "relay-log.info":          "<relay-log.info>",
			"/elsewhere/master.info":       "/elsewhere/master.info", // not the configured file
		}},
		{"info files given as full paths", FileNameSettings{
			DataDir: dd, MasterInfoFile: "/repl/master.info", RelayLogInfoFile: "/repl/relay-log.info",
		}, map[string]string{
			"/repl/master.info":    "<master.info>",
			"/repl/relay-log.info": "<relay-log.info>",
		}},
	}
	for _, tt := range tests {
		m := fileName(t, tt.s)
		for path, want := range tt.maps {
			if got := m.Map(path); got != want {
				t.Errorf("%s: Map(%q) = %q, want %q", tt.name, path, got, want)
			}
		}
	}
}

func TestNoBinlogOrRelayLog(t *testing.T) {
	m := fileName(t, FileNameSettings{DataDir: dd})
	if got := m.Map(dd + "mysqld-bin.000001"); got != "<datadir>/mysqld-bin.000001" {
		t.Errorf("got %q", got)
	}
}

func TestNewErrors(t *testing.T) {
	if _, err := NewFileName(FileNameSettings{DataDir: "/var/lib/mysql"}); err == nil {
		t.Error("datadir without trailing /: no error")
	}
}

func TestCache(t *testing.T) {
	m := fileName(t, FileNameSettings{DataDir: dd})
	m.Map(dd + "sbtest/sbtest1.ibd")
	m.Map(dd + "#innodb_temp/temp_1.ibt")
	if _, ok := m.cache[dd+"sbtest/sbtest1.ibd"]; !ok {
		t.Error("table file not cached")
	}
	if _, ok := m.cache[dd+"#innodb_temp/temp_1.ibt"]; ok {
		t.Error("temp table cached")
	}
}

// TestSurveyedNames maps every file name recorded from the harness servers
// (testdata/filename/<flavour>-<version>) and fails on any left unmapped.
func TestSurveyedNames(t *testing.T) {
	dirs, err := filepath.Glob("testdata/filename/*")
	if err != nil || len(dirs) == 0 {
		t.Fatalf("no testdata: %v", err)
	}
	for _, dir := range dirs {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			set := map[string]string{}
			for _, f := range readTSV(t, filepath.Join(dir, "settings.tsv")) {
				set[f[0]] = f[len(f)-1]
				if len(f) == 1 {
					set[f[0]] = ""
				}
			}
			m := fileName(t, FileNameSettings{DataDir: set["datadir"], BinlogBase: set["log_bin_basename"], RelayLogBase: set["relay_log_basename"]})
			for _, f := range readTSV(t, filepath.Join(dir, "files.tsv")) {
				got := m.Map(f[0])
				if strings.HasPrefix(got, "<datadir>/") || strings.HasPrefix(got, "/") {
					t.Errorf("%s (%s): unmapped, got %q", f[0], f[1], got)
				}
			}
		})
	}
}

func readTSV(t *testing.T, path string) [][]string {
	t.Helper()
	fh, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fh.Close() }()
	var rows [][]string
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		rows = append(rows, strings.Split(sc.Text(), "\t"))
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}

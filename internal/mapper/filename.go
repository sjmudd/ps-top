package mapper

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/sjmudd/ps-top/v2/internal/snapshot"
)

// FileNameSettings are the server values file name mapping depends on.
type FileNameSettings struct {
	DataDir    string // @@datadir, ending in "/"
	BinlogBase string // @@log_bin_basename; empty: no binlog rule
	// The relay log's base name is RelayLogBase if set, else RelayLog, else
	// the pid file's name with "-relay-bin", as MariaDB names relay logs
	// when @@relay_log is empty (MYSQL_LOG::generate_name). RelayLog and the
	// pid file's name are relative to DataDir and lose their extension.
	// None set: no relay log rule.
	RelayLogBase string // @@relay_log_basename (empty on MariaDB 12.3)
	RelayLog     string // @@relay_log, e.g. "replica1-relay-bin"
	PidFile      string // @@pid_file, e.g. "/run/mysqld/mysqld.pid"
	// The replication info files, relative to DataDir unless a full path,
	// shown as <master.info> and <relay-log.info>. MariaDB only: MySQL
	// keeps them in tables. Empty: no rule.
	MasterInfoFile   string // @@master_info_file, e.g. "master.info"
	RelayLogInfoFile string // @@relay_log_info_file, e.g. "relay-log.info"
	// Munge renames a table's "schema.table" name, e.g. by ~/.pstoprc's
	// rules (see internal/munge). The result is split at its first "."
	// into schema and table. nil: no renaming.
	Munge func(string) string
}

// relayLogBase returns the relay log's base name from s, or "" if none.
func (s FileNameSettings) relayLogBase() string {
	switch {
	case s.RelayLogBase != "":
		return s.RelayLogBase
	case s.RelayLog != "":
		return inDataDir(s.DataDir, stripExt(s.RelayLog))
	case s.PidFile != "" && s.DataDir != "":
		base := s.PidFile[strings.LastIndexByte(s.PidFile, '/')+1:]
		return s.DataDir + stripExt(base) + "-relay-bin"
	}
	return ""
}

// serverPath returns a base name setting as performance_schema shows its
// files: in DataDir if relative, and cleaned. The server reports
// log_bin_basename and relay_log_basename as configured, e.g.
// "../log/binlog", but names the files resolved against the datadir, e.g.
// "/var/lib/log/binlog.000001" (MySQL 8.0 to 26.10, MariaDB 10.6 to 12.3).
func (s FileNameSettings) serverPath(name string) string {
	if name == "" {
		return ""
	}
	return cleanPath(inDataDir(s.DataDir, name))
}

// inDataDir returns name, with datadir before it if it is relative.
func inDataDir(datadir, name string) string {
	if name == "" || strings.HasPrefix(name, "/") {
		return name
	}
	return datadir + name
}

// stripExt returns name without the extension of its last part, e.g.
// "mysqld.pid" as "mysqld".
func stripExt(name string) string {
	i := strings.LastIndexByte(name, '.')
	if i > strings.LastIndexByte(name, '/') {
		return name[:i]
	}
	return name
}

const tempTable = "<temp_table>"

type fileRule struct {
	re    *regexp.Regexp
	label string
}

var (
	reEncoded   = regexp.MustCompile(`@([0-9a-fA-F]{4})`)
	reSlashDot  = regexp.MustCompile(`/(\.)?/`)
	reParentDir = regexp.MustCompile(`[^/]+/\.\./`)
	reTempPool  = regexp.MustCompile(`/#innodb_temp/temp_\d+\.ibt$`)
	reTableFile = regexp.MustCompile(`/([^/]+)/([^/]+)\.(frm|ibd|MYD|MYI|MAD|MAI|CSM|CSV|par|sdi)$`)
	rePartition = regexp.MustCompile(`(?i)#p#.*$`)

	// fixedFiles are checked in order, before the table rule.
	fixedFiles = []fileRule{
		{regexp.MustCompile(`/ibdata\d+$`), "<ibdata>"},
		{regexp.MustCompile(`/ibtmp\d+$`), "<ibtmp>"},
		{regexp.MustCompile(`/undo_?\d+$`), "<undo_log>"},
		{regexp.MustCompile(`/(ib_logfile\d+|#innodb_redo/#ib_redo\d+(_tmp)?)$`), "<redo_log>"},
		{regexp.MustCompile(`/#ib_\d+_\d+\.dblwr$`), "<doublewrite>"},
		{regexp.MustCompile(`/aria_log(\.\d+|_control)$`), "<aria_log>"},
		{regexp.MustCompile(`/ddl_recovery(-backup)?\.log$`), "<ddl_log>"},
		{regexp.MustCompile(`/Innodb Merge Temp File$`), "<innodb_merge_temp>"},
		{regexp.MustCompile(`/db\.opt$`), "<db_opt>"},
		{regexp.MustCompile(`/auto\.cnf$`), "<auto_cnf>"},
		{regexp.MustCompile(`/slowlog$`), "<slow_log>"},
		{regexp.MustCompile(`\.pid$`), "<pid_file>"},
		{regexp.MustCompile(`/errmsg\.sys$`), "<errmsg>"},
		{regexp.MustCompile(`/charsets/Index\.xml$`), "<charset>"},
	}
)

// FileName maps a FILE_NAME of performance_schema.file_summary_by_instance
// to a name of what the file represents, so that file I/O can be shown per
// table instead of per file:
//
//	/var/lib/mysql/sbtest/ptest#p#p2024.ibd  -> sbtest.ptest  (partitions merge into their table)
//	/var/lib/mysql/#innodb_temp/temp_7.ibt   -> <temp_table>  (all temporary tables)
//	/var/lib/mysql/ibtmp1                    -> <ibtmp>
//	/var/lib/mysql/mysqld-bin.000002         -> <binlog>
//
// Table files map to a key of two parts, schema and table (see
// snapshot.JoinKey), shown as <schema>.<table>, so a name that contains "."
// stays one part. Other server files map to a label in
// angle brackets, such as <binlog>, <redo_log> or <undo_log>. To create the
// appropriate names, the values of some global variables (datadir,
// log_bin_basename, relay_log_basename) are used where needed. Thus a
// FileName must be built after connecting to the server. Its Map method is
// the key mapping of the file I/O view. It is not safe for concurrent use.
type FileName struct {
	datadir string
	rules   []fileRule
	munge   func(string) string
	cache   map[string]string
}

// NewFileName builds a file name mapper for a server with the given settings.
func NewFileName(s FileNameSettings) (*FileName, error) {
	if s.DataDir != "" && !strings.HasSuffix(s.DataDir, "/") {
		return nil, fmt.Errorf("mapper: datadir %q must end in /", s.DataDir)
	}
	m := &FileName{datadir: s.DataDir, munge: s.Munge, cache: make(map[string]string)}
	if s.DataDir != "" {
		m.rules = append(m.rules, fileRule{regexp.MustCompile(`^` + regexp.QuoteMeta(s.DataDir) + `mysql\.ibd$`), "<data_dictionary>"})
	}
	if base := s.serverPath(s.BinlogBase); base != "" {
		m.rules = append(m.rules, fileRule{regexp.MustCompile(`^` + regexp.QuoteMeta(base) + `\.(\d+(\.idx)?|index|state)$`), "<binlog>"})
	}
	if base := s.serverPath(s.relayLogBase()); base != "" {
		m.rules = append(m.rules, fileRule{regexp.MustCompile(`^` + regexp.QuoteMeta(base) + `\.(\d+|index)$`), "<relay_log>"})
	}
	for _, f := range []struct{ name, label string }{
		{s.MasterInfoFile, "<master.info>"},
		{s.RelayLogInfoFile, "<relay-log.info>"},
	} {
		if p := inDataDir(s.DataDir, f.name); p != "" {
			m.rules = append(m.rules, fileRule{regexp.MustCompile(`^` + regexp.QuoteMeta(p) + `$`), f.label})
		}
	}
	m.rules = append(m.rules, fixedFiles...)
	return m, nil
}

// Map returns the object name for path.
func (m *FileName) Map(path string) string {
	if name, ok := m.cache[path]; ok {
		return name
	}
	name := m.uncached(path)
	if name != tempTable { // temp table names are new each time
		m.cache[path] = name
	}
	return name
}

func (m *FileName) uncached(path string) string {
	p := cleanPath(decode(path))

	base := p[strings.LastIndexByte(p, '/')+1:]
	if strings.HasPrefix(base, "#sql") || reTempPool.MatchString(p) {
		return tempTable
	}
	for _, r := range m.rules {
		if r.re.MatchString(p) {
			return r.label
		}
	}
	if sm := reTableFile.FindStringSubmatch(p); sm != nil {
		return m.tableKey(sm[1], rePartition.ReplaceAllString(sm[2], ""))
	}
	name := p
	if m.datadir != "" && strings.HasPrefix(p, m.datadir) {
		name = "<datadir>/" + p[len(m.datadir):]
	}
	slog.Debug("no matching rule found for file name", "file", path, "name", name) // once per path: Map caches
	return name
}

// tableKey returns the key of schema and table, renamed by munge if set. A
// renamed name without a "." keeps its schema.
func (m *FileName) tableKey(schema, table string) string {
	if m.munge == nil {
		return snapshot.JoinKey(schema, table)
	}
	name := m.munge(schema + "." + table)
	if s, t, ok := strings.Cut(name, "."); ok {
		return snapshot.JoinKey(s, t)
	}
	return snapshot.JoinKey(schema, name)
}

// decode converts MySQL's @XXXX (hex code point) file name encoding.
func decode(s string) string {
	if !strings.Contains(s, "@") {
		return s
	}
	return reEncoded.ReplaceAllStringFunc(s, func(e string) string {
		n, err := strconv.ParseUint(e[1:], 16, 32)
		if err != nil {
			return e
		}
		return string(rune(n))
	})
}

// cleanPath removes "//", "/./" and "dir/../".
func cleanPath(p string) string {
	for {
		q := reParentDir.ReplaceAllString(reSlashDot.ReplaceAllString(p, "/"), "")
		if q == p {
			return p
		}
		p = q
	}
}

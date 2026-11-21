// Package filenamemap maps a FILE_NAME of
// performance_schema.file_summary_by_instance to a name of what the file
// represents, so that file I/O can be shown per table instead of per file:
//
//	/var/lib/mysql/sbtest/ptest#p#p2024.ibd  -> sbtest.ptest  (partitions merge into their table)
//	/var/lib/mysql/#innodb_temp/temp_7.ibt   -> <temp_table>  (all temporary tables)
//	/var/lib/mysql/ibtmp1                    -> <ibtmp>
//	/var/lib/mysql/mysqld-bin.000002         -> <binlog>
//
// Table files map to <schema>.<table>. Other server files map to a label in
// angle brackets, such as <binlog>, <redo_log> or <undo_log>. To create the
// appropriate names, the values of some global variables (datadir,
// log_bin_basename, relay_log_basename) are used where needed. Thus the
// Mapper must be built after connecting to the server. Its Map method is the
// key mapping of the file I/O view.
package filenamemap

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Settings are the server values the mapping depends on.
type Settings struct {
	DataDir      string // @@datadir, ending in "/"
	BinlogBase   string // @@log_bin_basename; empty: no binlog rule
	RelayLogBase string // @@relay_log_basename; empty: no relay log rule
}

const tempTable = "<temp_table>"

type rule struct {
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

	// fixed are checked in order, before the table rule.
	fixed = []rule{
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

// Mapper maps file names for one server. It is not safe for concurrent use.
type Mapper struct {
	datadir string
	rules   []rule
	cache   map[string]string
}

// New builds a mapper for a server with the given settings.
func New(s Settings) (*Mapper, error) {
	if s.DataDir != "" && !strings.HasSuffix(s.DataDir, "/") {
		return nil, fmt.Errorf("filenamemap: datadir %q must end in /", s.DataDir)
	}
	m := &Mapper{datadir: s.DataDir, cache: make(map[string]string)}
	if s.DataDir != "" {
		m.rules = append(m.rules, rule{regexp.MustCompile(`^` + regexp.QuoteMeta(s.DataDir) + `mysql\.ibd$`), "<data_dictionary>"})
	}
	if s.BinlogBase != "" {
		m.rules = append(m.rules, rule{regexp.MustCompile(`^` + regexp.QuoteMeta(s.BinlogBase) + `\.(\d+(\.idx)?|index|state)$`), "<binlog>"})
	}
	if s.RelayLogBase != "" {
		m.rules = append(m.rules, rule{regexp.MustCompile(`^` + regexp.QuoteMeta(s.RelayLogBase) + `\.(\d+|index)$`), "<relay_log>"})
	}
	m.rules = append(m.rules, fixed...)
	return m, nil
}

// Map returns the object name for path.
func (m *Mapper) Map(path string) string {
	if name, ok := m.cache[path]; ok {
		return name
	}
	name := m.uncached(path)
	if name != tempTable { // temp table names are new each time
		m.cache[path] = name
	}
	return name
}

func (m *Mapper) uncached(path string) string {
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
		return sm[1] + "." + rePartition.ReplaceAllString(sm[2], "")
	}
	if m.datadir != "" && strings.HasPrefix(p, m.datadir) {
		return "<datadir>/" + p[len(m.datadir):]
	}
	return p
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

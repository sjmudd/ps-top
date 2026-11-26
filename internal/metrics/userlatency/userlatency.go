// Package userlatency is activity per user from the processlist. Each
// connection is one entry, keyed by user, host (without port), db and
// connection ID. The user view maps the key to the user and merges a user's
// connections: most columns are added, Hosts and DBs count distinct values,
// and their totals are distinct over all connections, not sums.
//
// The rules are worked out in Go from the text columns (see Row), not in
// SQL, so they can be tested and changed in one place. Values are current
// (CurrentValue), as the processlist shows the server now.
//
// performance_schema.processlist is read if the server has it (MySQL), else
// INFORMATION_SCHEMA.PROCESSLIST (MariaDB). On some MySQL versions (seen on
// 8.0.34, bug 113930) performance_schema.processlist shows replication
// threads under the user and host that started replication instead of
// "system user", so replication threads are recognised by their command and
// state, not by their user.
package userlatency

import (
	"strconv"
	"strings"

	"github.com/sjmudd/ps-top/v2/internal/metric"
)

// Column is a value column of Spec. Each connection's values are 0 or 1
// (counts) or its TIME in seconds.
type Column int

// The value columns, in Spec.Columns order.
const (
	RunTime     Column = iota // TIME of a busy client connection
	SleepTime                 // TIME of an idle client connection
	TotalTime                 // RunTime + SleepTime
	Connections               // every connection: 1
	Active                    // a busy client connection
	Repl                      // a replication thread
	Hosts                     // distinct hosts (key part HOST)
	Dbs                       // distinct databases (key part DB)
	Selects                   // a busy client connection whose statement contains SELECT
	Inserts                   // ... INSERT
	Updates                   // ... UPDATE
	Deletes                   // ... DELETE
	Others                    // a busy client connection whose statement contains none of the four

	columns // number of value columns
)

// The key parts, in Spec.Keys order.
const (
	partUser = iota
	partHost
	partDB
	partID
)

// The fields read, in Spec.Fields order.
const (
	fieldID = iota
	fieldUser
	fieldHost
	fieldDB
	fieldCommand
	fieldTime
	fieldState
	fieldInfo
)

// Spec describes the metric.
var Spec = metric.Spec[Column]{
	Name:   "user_latency",
	Tables: []string{"performance_schema.processlist", "INFORMATION_SCHEMA.PROCESSLIST"},
	Keys:   []string{"USER", "HOST", "DB", "ID"},
	Columns: []metric.Column{
		RunTime:     {Name: "RUN_TIME", Kind: metric.Gauge},
		SleepTime:   {Name: "SLEEP_TIME", Kind: metric.Gauge},
		TotalTime:   {Name: "TOTAL_TIME", Kind: metric.Gauge},
		Connections: {Name: "CONNECTIONS", Kind: metric.Gauge},
		Active:      {Name: "ACTIVE", Kind: metric.Gauge},
		Repl:        {Name: "REPL", Kind: metric.Gauge},
		Hosts:       {Name: "HOSTS", Kind: metric.Gauge, Merge: metric.DistinctPart, Part: partHost},
		Dbs:         {Name: "DBS", Kind: metric.Gauge, Merge: metric.DistinctPart, Part: partDB},
		Selects:     {Name: "SELECTS", Kind: metric.Gauge},
		Inserts:     {Name: "INSERTS", Kind: metric.Gauge},
		Updates:     {Name: "UPDATES", Kind: metric.Gauge},
		Deletes:     {Name: "DELETES", Kind: metric.Gauge},
		Others:      {Name: "OTHERS", Kind: metric.Gauge},
	},
	Fields: []string{"ID", "USER", "HOST", "DB", "COMMAND", "TIME", "STATE", "INFO"},
	Row:    Row,
	Types:  []metric.MetricType{metric.CurrentValue},
}

// systemUser is the user the server shows for its own threads.
const systemUser = "system user"

// Commands of replication threads: binlog dump threads on a source, and
// MariaDB's replica threads. MySQL's replica threads show Connect or Query.
var replicationCommands = map[string]bool{
	"Binlog Dump":      true,
	"Binlog Dump GTID": true, // MySQL with GTID
	"Slave_IO":         true, // MariaDB replica I/O thread
	"Slave_SQL":        true, // MariaDB replica SQL thread
	"Slave_worker":     true, // MariaDB parallel replication worker
}

// States only replication threads show, in normalised form (see normalise):
// a state contains one of these. From the replication harness on MySQL 8.0
// to 26.10 and MariaDB 10.6 to 12.3. Add any found missing.
var replicationStates = []string{
	"has sent all binlog to replica",                           // binlog dump, idle
	"sending binlog event to replica",                          // binlog dump, busy
	"waiting for source to send event",                         // replica I/O, idle
	"queueing source event to the relay log",                   // replica I/O, busy
	"replica has read all relay log; waiting for more updates", // replica SQL, idle
	"reading event from the relay log",                         // replica SQL, busy
	"waiting for an event from coordinator",                    // MySQL worker, idle
	"waiting for dependent transaction to commit",              // MySQL worker
	"applying batch of row changes",                            // MySQL worker, ROW events
	"waiting for work from sql thread",                         // MariaDB worker, idle
	"update_rows_log_event::",                                  // MariaDB worker, ROW events
	"write_rows_log_event::",                                   // MariaDB worker, ROW events
	"delete_rows_log_event::",                                  // MariaDB worker, ROW events
}

// idleStates are, by command, the normalised states of a thread that is
// waiting, not working. Sleep is idle whatever its state. Any other
// combination is busy. Add any found missing.
var idleStates = map[string][]string{
	"Daemon":           {"waiting on empty queue", "suspending"}, // event scheduler, MySQL 9.x daemon
	"Binlog Dump":      {"has sent all binlog to replica"},
	"Binlog Dump GTID": {"has sent all binlog to replica"},
	"Connect":          {"waiting for source to send event", "waiting for an event from coordinator"},
	"Query":            {"replica has read all relay log; waiting for more updates", "waiting for an event from coordinator"},
	"Slave_IO":         {"waiting for source to send event"},
	"Slave_SQL":        {"replica has read all relay log; waiting for more updates"},
	"Slave_worker":     {"waiting for work from sql thread"},
}

// statements are the statement types counted, in Column order from Selects.
var statements = []string{"SELECT", "INSERT", "UPDATE", "DELETE"}

// normalise returns state in lower case with the older terms replaced, so
// one list matches every version: "master" is "source", "slave" is
// "replica".
func normalise(state string) string {
	s := strings.ToLower(state)
	s = strings.ReplaceAll(s, "master", "source")
	return strings.ReplaceAll(s, "slave", "replica")
}

// IsReplication reports whether a processlist row is a replication thread,
// by its command or its state.
//
// The user does not decide it: "system user" is not always replication. The
// server shows "system user" for any of its own threads that has no user
// (MySQL performance_schema.processlist: any system thread except a named
// singleton such as event_scheduler). MySQL's system thread types
// (include/mysql/thread_type.h, checked on 26.10) are replica I/O, SQL and
// workers, but also the NDB binlog, event scheduler and workers, info
// repository, GTID table compression, background (e.g. clone), data
// dictionary, initialise, init file, upgrade and group replication
// connection threads. MariaDB's (sql/sql_class.h, checked on 12.3) are
// delayed insert, replica I/O and SQL, event scheduler and workers, binlog
// background, replica background, generic and semi-sync master background.
// A replica worker caught while applying an event shows an ordinary state
// (e.g. "updating") and is then not counted as replication.
func IsReplication(command, state string) bool {
	if replicationCommands[command] {
		return true
	}
	s := normalise(state)
	for _, r := range replicationStates {
		if strings.Contains(s, r) {
			return true
		}
	}
	return false
}

// IsIdle reports whether a processlist row is waiting rather than working:
// Sleep, or a command and state listed in idleStates.
func IsIdle(command, state string) bool {
	if command == "Sleep" {
		return true
	}
	s := normalise(state)
	for _, i := range idleStates[command] {
		if strings.Contains(s, i) {
			return true
		}
	}
	return false
}

// host returns HOST without its port. A port is removed only if HOST has
// one ":", so an IPv6 address is left as it is.
func host(h string) string {
	if strings.Count(h, ":") == 1 {
		return h[:strings.IndexByte(h, ':')]
	}
	return h
}

// Row turns one processlist row (Spec.Fields: ID, USER, HOST, DB, COMMAND,
// TIME, STATE, INFO, NULL read as "") into its key parts and values.
//
//   - Connections is 1 for every row.
//   - A replication thread (IsReplication) has Repl 1 and no other values.
//   - Another "system user" thread is an internal thread, not a normal user's
//     connection: Connections only.
//   - Any other row is a client connection. Busy (not IsIdle), it has
//     Active 1, its TIME as RunTime, and 1 in each statement type its INFO
//     contains, in any case (INSERT ... SELECT counts as both), or in Others
//     if none. Idle, its TIME is SleepTime. A busy client includes the
//     connection reading the processlist, counted as a select, and a
//     connection still logging in ("unauthenticated user").
//   - The host part is HOST without its port, as the server shows it. The
//     servers are not consistent: the MySQL replica I/O thread has an empty
//     HOST in performance_schema.processlist but "connecting host" in
//     INFORMATION_SCHEMA.PROCESSLIST, and on 8.0.34 (bug 113930) replica
//     threads show the host that started replication. Values are used as
//     seen, not corrected.
func Row(f []string) (key []string, values []uint64) {
	user, command, state := f[fieldUser], f[fieldCommand], f[fieldState]
	key = make([]string, 4)
	key[partUser], key[partHost], key[partDB], key[partID] = user, host(f[fieldHost]), f[fieldDB], f[fieldID]
	values = make([]uint64, columns)
	values[Connections] = 1
	switch {
	case IsReplication(command, state):
		values[Repl] = 1
	case user == systemUser:
	case IsIdle(command, state):
		values[SleepTime] = seconds(f[fieldTime])
	default:
		values[Active] = 1
		values[RunTime] = seconds(f[fieldTime])
		info := strings.ToUpper(f[fieldInfo])
		other := uint64(1)
		for i, s := range statements {
			if strings.Contains(info, s) {
				values[Selects+Column(i)] = 1
				other = 0
			}
		}
		values[Others] = other
	}
	values[TotalTime] = values[RunTime] + values[SleepTime]
	return key, values
}

// seconds returns TIME as a number, 0 if it is empty or not a non-negative
// integer.
func seconds(t string) uint64 {
	n, err := strconv.ParseUint(t, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

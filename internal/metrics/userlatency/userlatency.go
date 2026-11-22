// Package userlatency is activity per user from the processlist: time spent
// running and sleeping, connections, active connections, distinct hosts and
// databases, and statements by type. A row's key is the user. Values are
// current (CurrentValue), as the processlist shows the server now.
//
// The server groups the rows (GROUP BY USER), so counts of distinct hosts and
// databases are per user and need no merging afterwards. The rules are the
// old program's (model/userlatency):
//
//   - Time and active counts leave out "system user", connections with no
//     host, and Binlog Dump threads. A Binlog Dump thread counts as active
//     while it is sending binlog events.
//   - A host is HOST without its port.
//   - A statement type is counted if INFO contains its keyword, in any case.
//
// performance_schema.processlist is read if the server has it (MySQL), else
// INFORMATION_SCHEMA.PROCESSLIST (MariaDB). With
// performance_schema_show_processlist ON, MySQL shows replication threads
// in performance_schema.processlist under the user who started replication,
// not "system user" (MySQL bugs 113930 and 77293).
package userlatency

import "github.com/sjmudd/ps-top/internal/metric"

// Column is a value column of Spec.
type Column int

// The value columns, in Spec.Columns order.
const (
	RunTime Column = iota
	SleepTime
	TotalTime // RunTime + SleepTime
	Connections
	Active
	Hosts
	Dbs
	Selects
	Inserts
	Updates
	Deletes
)

// counted is true for rows whose time is counted: not "system user", with a
// host, and not a Binlog Dump thread.
const counted = "COALESCE(USER, '') <> 'system user' AND COALESCE(SUBSTRING_INDEX(HOST, ':', 1), '') <> '' AND COALESCE(COMMAND, '') <> 'Binlog Dump'"

// contains counts rows whose INFO contains a keyword given in two parts. The
// parts are joined by the server, so this query's own INFO does not contain
// INSERT, UPDATE or DELETE. It starts with SELECT, so the connection reading
// it counts as one select, as in the old program.
func contains(a, b string) string {
	return "SUM(IF(UPPER(INFO) LIKE CONCAT('%" + a + "', '" + b + "%'), 1, 0))"
}

// Spec describes the metric.
var Spec = metric.Spec[Column]{
	Name:   "user_latency",
	Tables: []string{"performance_schema.processlist", "INFORMATION_SCHEMA.PROCESSLIST"},
	Keys:   []string{"USER"},
	Columns: []metric.Column{
		RunTime:     {Name: "RUN_TIME", Kind: metric.Gauge, Expr: "SUM(IF(" + counted + " AND COALESCE(COMMAND, '') <> 'Sleep', TIME, 0))"},
		SleepTime:   {Name: "SLEEP_TIME", Kind: metric.Gauge, Expr: "SUM(IF(" + counted + " AND COMMAND = 'Sleep', TIME, 0))"},
		TotalTime:   {Name: "TOTAL_TIME", Kind: metric.Gauge, Expr: "SUM(IF(" + counted + ", TIME, 0))"},
		Connections: {Name: "CONNECTIONS", Kind: metric.Gauge, Expr: "COUNT(*)"},
		Active: {Name: "ACTIVE", Kind: metric.Gauge, Expr: "SUM(IF((" + counted + " AND COALESCE(COMMAND, '') <> 'Sleep')" +
			" OR (COMMAND = 'Binlog Dump' AND STATE LIKE '%Sending binlog event to slave%'), 1, 0))"},
		Hosts: {Name: "HOSTS", Kind: metric.Gauge, Expr: "COUNT(DISTINCT NULLIF(SUBSTRING_INDEX(HOST, ':', 1), ''))"},
		Dbs:   {Name: "DBS", Kind: metric.Gauge, Expr: "COUNT(DISTINCT NULLIF(DB, ''))"},
		// The names must not contain the keywords either.
		Selects: {Name: "SEL_STMTS", Kind: metric.Gauge, Expr: contains("SEL", "ECT")},
		Inserts: {Name: "INS_STMTS", Kind: metric.Gauge, Expr: contains("INS", "ERT")},
		Updates: {Name: "UPD_STMTS", Kind: metric.Gauge, Expr: contains("UPD", "ATE")},
		Deletes: {Name: "DEL_STMTS", Kind: metric.Gauge, Expr: contains("DEL", "ETE")},
	},
	GroupBy: "USER",
	Types:   []metric.MetricType{metric.CurrentValue},
}

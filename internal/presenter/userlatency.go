package presenter

import (
	"github.com/sjmudd/ps-top/internal/formatter"
	"github.com/sjmudd/ps-top/internal/mapper"
	"github.com/sjmudd/ps-top/internal/metrics/userlatency"
)

// UserLatency shows activity per user from the processlist: one row per user,
// every user shown, ordered by total time, then connections, then name. The
// columns (see the userlatency package for the rules):
//
//   - Run Time, %: time of busy client connections, and its share.
//   - Sleeping, %: time of idle client connections, and its share.
//   - Conn: all connections. Actv: busy client connections. Repl:
//     replication threads. Idle connections are Conn - Actv - Repl.
//   - Hosts, DBs: distinct hosts (without port) and databases. On the totals
//     line they are distinct over all connections, not the sum of the column.
//   - Sel, Ins, Upd, Del: busy client connections whose statement contains
//     the keyword. Other: busy client connections whose statement contains
//     none of them.
//
// Zero values are blank. With --database-filter, only connections whose db
// is one of those named, or that have no db, are counted.
var UserLatency = View[userlatency.Column]{
	Name:      "user_latency",
	Title:     "Activity by Username",
	Spec:      userlatency.Spec,
	MapKey:    mapper.Part(0), // a connection's key to its user
	Anonymise: anonymiseUser,
	Schema:    mapper.Part(2), // a connection's db, "" if it has none
	Cells: [][]Cell[userlatency.Column]{
		{Value("Run Time", userlatency.RunTime, formatter.Seconds, 10), ShareOfTotal("%", userlatency.RunTime, 6)},
		{Value("Sleeping", userlatency.SleepTime, formatter.Seconds, 10), ShareOfTotal("%", userlatency.SleepTime, 6)},
		{Value("Conn", userlatency.Connections, formatter.Count, 4), Value("Actv", userlatency.Active, formatter.Count, 4),
			Value("Repl", userlatency.Repl, formatter.Count, 4)},
		{Value("Hosts", userlatency.Hosts, formatter.Count, 5), Value("DBs", userlatency.Dbs, formatter.Count, 3)},
		{Value("Sel", userlatency.Selects, formatter.Count, 3), Value("Ins", userlatency.Inserts, formatter.Count, 3),
			Value("Upd", userlatency.Updates, formatter.Count, 3), Value("Del", userlatency.Deletes, formatter.Count, 3),
			Value("Other", userlatency.Others, formatter.Count, 5)},
		{Key[userlatency.Column]("User", 0)},
	},
	SortColumn: userlatency.TotalTime,
	ThenBy:     []userlatency.Column{userlatency.Connections},
	ShowZero:   true,
}

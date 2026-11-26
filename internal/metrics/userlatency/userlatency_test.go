package userlatency

import (
	"slices"
	"testing"
)

// row returns the processlist fields of one connection, as Spec.Fields.
func row(id, user, host, db, command, time, state, info string) []string {
	return []string{id, user, host, db, command, time, state, info}
}

// TestRow checks the rules on processlist rows as the replication harness
// shows them (MySQL 8.0 to 26.10, MariaDB 10.6 to 12.3).
func TestRow(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		key    []string
		want   map[Column]uint64 // values not listed are 0; Connections is always 1
	}{
		{"idle client", row("21", "app", "10.0.0.5:59216", "shop", "Sleep", "40", "", ""),
			[]string{"app", "10.0.0.5", "shop", "21"}, map[Column]uint64{SleepTime: 40, TotalTime: 40}},
		{"select", row("22", "app", "10.0.0.5:59218", "shop", "Query", "3", "executing", "select * from orders"),
			[]string{"app", "10.0.0.5", "shop", "22"}, map[Column]uint64{Active: 1, RunTime: 3, TotalTime: 3, Selects: 1}},
		{"insert select counts both", row("23", "app", "10.0.0.5:1", "shop", "Query", "1", "executing", "INSERT INTO a SELECT * FROM b"),
			nil, map[Column]uint64{Active: 1, RunTime: 1, TotalTime: 1, Inserts: 1, Selects: 1}},
		{"update", row("24", "app", "h:1", "shop", "Execute", "0", "updating", "UPDATE t SET k = k + 1"),
			nil, map[Column]uint64{Active: 1, Updates: 1}},
		{"delete", row("25", "app", "h:1", "shop", "Query", "0", "updating", "delete from t"),
			nil, map[Column]uint64{Active: 1, Deletes: 1}},
		{"other statement", row("26", "app", "h:1", "shop", "Query", "2", "executing", "CALL refresh()"),
			nil, map[Column]uint64{Active: 1, RunTime: 2, TotalTime: 2, Others: 1}},
		{"busy without statement", row("27", "app", "h:1", "", "Prepare", "0", "", ""),
			nil, map[Column]uint64{Active: 1, Others: 1}},
		{"login in progress", row("30", "unauthenticated user", "10.0.0.6:1234", "", "Connect", "0", "login", ""),
			[]string{"unauthenticated user", "10.0.0.6", "", "30"}, map[Column]uint64{Active: 1, Others: 1}},
		{"event scheduler idle", row("5", "event_scheduler", "localhost", "", "Daemon", "46", "Waiting on empty queue", ""),
			[]string{"event_scheduler", "localhost", "", "5"}, map[Column]uint64{SleepTime: 46, TotalTime: 46}},
		{"MySQL 9.x daemon", row("7", "system user", "", "", "Daemon", "27", "Suspending", ""),
			[]string{"system user", "", "", "7"}, map[Column]uint64{}},
		{"dump, MySQL GTID, idle", row("14", "repl", "10.0.0.4:36406", "", "Binlog Dump GTID", "41", "Source has sent all binlog to replica; waiting for more updates", ""),
			[]string{"repl", "10.0.0.4", "", "14"}, map[Column]uint64{Repl: 1}},
		{"dump, MySQL, busy", row("15", "repl", "10.0.0.3:1", "", "Binlog Dump", "3", "Sending binlog event to replica", ""),
			nil, map[Column]uint64{Repl: 1}},
		{"dump, MariaDB", row("10", "repl", "10.0.0.4:59674", "", "Binlog Dump", "21", "Master has sent all binlog to slave; waiting for more updates", ""),
			nil, map[Column]uint64{Repl: 1}},
		{"replica I/O, MySQL, performance_schema", row("10", "system user", "", "", "Connect", "41", "Waiting for source to send event", ""),
			[]string{"system user", "", "", "10"}, map[Column]uint64{Repl: 1}},
		// INFORMATION_SCHEMA shows the same thread with HOST "connecting
		// host": used as seen.
		{"replica I/O, MySQL, INFORMATION_SCHEMA", row("10", "system user", "connecting host", "", "Connect", "41", "Waiting for source to send event", ""),
			[]string{"system user", "connecting host", "", "10"}, map[Column]uint64{Repl: 1}},
		{"replica SQL, MySQL", row("11", "system user", "", "", "Query", "7", "Replica has read all relay log; waiting for more updates", ""),
			nil, map[Column]uint64{Repl: 1}},
		// An ordinary state does not show replication. As "system user" it is
		// an internal thread, not a client: Connections only.
		{"worker applying a statement, MySQL", row("16", "system user", "", "", "Connect", "0", "updating", "UPDATE t SET k = 1"),
			nil, map[Column]uint64{}},
		{"replica I/O, MariaDB", row("7", "system user", "", "", "Slave_IO", "7", "Waiting for master to send event", ""),
			nil, map[Column]uint64{Repl: 1}},
		{"worker, MariaDB", row("9", "system user", "", "", "Slave_worker", "0", "Waiting for work from SQL thread", ""),
			nil, map[Column]uint64{Repl: 1}},
		// MySQL bug 113930 (seen on 8.0.34): replica threads under the user
		// that started replication. Recognised by their state.
		{"replica I/O under the starting user", row("13", "repl_admin", "localhost", "", "Connect", "20", "Waiting for source to send event", ""),
			[]string{"repl_admin", "localhost", "", "13"}, map[Column]uint64{Repl: 1}},
		{"worker under the starting user", row("15", "repl_admin", "172.21.0.2", "", "Query", "1", "Waiting for an event from Coordinator", ""),
			nil, map[Column]uint64{Repl: 1}},
	}
	for _, tt := range tests {
		key, values := Row(tt.fields)
		if tt.key != nil && !slices.Equal(key, tt.key) {
			t.Errorf("%s: key = %q, want %q", tt.name, key, tt.key)
		}
		if len(values) != len(Spec.Columns) {
			t.Fatalf("%s: %d values, want %d", tt.name, len(values), len(Spec.Columns))
		}
		want := make([]uint64, len(Spec.Columns))
		want[Connections] = 1
		for c, v := range tt.want {
			want[c] = v
		}
		if !slices.Equal(values, want) {
			t.Errorf("%s: values\n got %v\nwant %v", tt.name, values, want)
		}
	}
}

func TestHost(t *testing.T) {
	for h, want := range map[string]string{
		"10.0.0.5:59216":    "10.0.0.5",
		"app.example.com:1": "app.example.com",
		"localhost":         "localhost",
		"":                  "",
		"::1":               "::1", // IPv6: more than one ":", left as it is
	} {
		if got := host(h); got != want {
			t.Errorf("host(%q) = %q, want %q", h, got, want)
		}
	}
}

func TestSpec(t *testing.T) {
	if err := Spec.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(Spec.Columns) != int(columns) {
		t.Errorf("%d columns, the constants say %d", len(Spec.Columns), columns)
	}
}

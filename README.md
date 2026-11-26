# ps-top - a top-like program for MySQL

ps-top collects information from the `performance_schema` database of
MySQL and MariaDB and shows where the server is spending its time, in
real time: by table, by file, by user, by mutex, by SQL stage and by
memory area. Values are shown as the difference over a recent window
(by default the last 10 seconds), or as the current values.

It runs full screen, one view at a time, or prints the views as text
with `--batch`, for logging or for sharing.

This README describes ps-top v2, a rewrite. For v1, see the
[v1.3.3 README](https://github.com/sjmudd/ps-top/blob/v1.3.3/README.md).

## Installation

```sh
go install github.com/sjmudd/ps-top/v2@latest
```

This needs Go 1.26 or later. The binary is installed in `$(go env GOPATH)/bin`.

## Usage

```sh
ps-top                                         # full screen: one view at a time, h for the keys
ps-top --interval=5s --window=30s              # redraw every 5s, differences over 30s
ps-top --batch                                 # all views as text, every --interval, until interrupted
ps-top --batch --interval=10s --count=6        # every 10s, 6 outputs
ps-top --batch --view=table_io_latency:current,file_io_latency
ps-top --batch --view=user_latency --limit=10  # top 10 lines
ps-top --batch --totals                        # totals lines only
ps-top --batch --anonymise                     # schema1.table2, user1: output to share
ps-top --batch --database-filter=shop,crm      # only the shop and crm databases
```

Metrics are collected every `--collect-interval` (default 1s). The
screen or output is redrawn every `--interval` (default 2s). Differences
cover `--window` (default 10s), a multiple of the collect interval.

With `--batch`, the first output shows values up to now, as `vmstat`'s
first line does. The following ones show each view in its style: the
difference over the window, or current values. `--view` takes a list of
view names, each optionally followed by `:current` or `:difference`.

`ps-top --help` lists every option.

## Connecting

```sh
ps-top                                         # ~/.my.cnf if it exists, else $USER on 127.0.0.1:3306
ps-top --defaults-file=/path/to/my.cnf         # the [client] group of this option file
ps-top --host=db1 --port=3307 --user=app --askpass
ps-top --socket=/run/mysqld/mysqld.sock
ps-top --dsn='app:secret@tcp(db1:3306)/performance_schema?interpolateParams=true'
MYSQL_DSN='app:secret@tcp(db1:3306)/performance_schema' ps-top --use-environment
```

- `--askpass` asks for the password at startup, so it is not given on
  the command line.
- `--dsn` and `--use-environment` (`MYSQL_DSN`) take a
  [go-sql-driver DSN](https://github.com/go-sql-driver/mysql#dsn-data-source-name),
  used as given, and cannot be combined with the other connection
  options. Use them for settings the other options do not cover, such
  as TLS: `?tls=skip-verify`.
- The DSN built from the option file and options sets
  `interpolateParams=true`, so a query with arguments is one round trip
  rather than a prepared statement's three. Add it to a DSN you give.

## Configuration

Tables named alike, such as one table per day, can be shown as one row in
the File I/O view. Put the rules in the `[munge]` section of `~/.pstoprc`:
a regular expression, then its replacement.

```ini
[munge]
_[0-9]{8}$ = _YYYYMMDD
_[0-9]{6}$ = _YYYYMM
```

With these, `shop.orders_20260101` is shown as `shop.orders_YYYYMMDD`.
The rules are applied in no fixed order, not the order written: a known
issue.

## Server setup and grants

`performance_schema` must be enabled. It is on by default in MySQL and
off by default in MariaDB. To enable it, set this in the server's
configuration and restart the server:

```ini
[mysqld]
performance_schema = 1
```

The user ps-top connects as needs:

- `SELECT` on `performance_schema`.
- `UPDATE` on `performance_schema.setup_instruments`, for the Mutex
  Latency and SQL Stage Latency views. ps-top enables the instruments
  these views need if they are off, and restores their settings when it
  stops. Without this grant, or on a read-only server, these views may
  be empty.
- `PROCESS`, for the Activity by Username view to show other users'
  connections. Without it only ps-top's own connection is shown.

## Views

| Name | Title | Shows |
|---|---|---|
| `table_io_latency` | Table I/O Latency | time spent on table operations, by table |
| `table_io_ops` | Table I/O Ops | the number of table operations, by table |
| `file_io_latency` | File I/O Latency | time spent on file I/O, by table or file |
| `table_lock_latency` | Locks by Table Name | time spent waiting for table locks, by table |
| `user_latency` | Activity by Username | connections, run and sleep times and statements, by user |
| `mutex_latency` | Mutex Latency | time spent waiting for mutexes |
| `stages_latency` | SQL Stage Latency | time spent in each SQL stage |
| `memory_usage` | Memory Usage | memory in use, by memory area |

A view whose table the server does not have is skipped, and ps-top says so.

ps-top also shows its own costs: for each metric, the count, time and
rows of its collection stages, the time of each render, and its own
memory, garbage collection and goroutines. With `--batch` these are three
views, `ps_top_stages`, `ps_top_renders` and `ps_top_process`, printed
after the others. Full screen they are one view, `ps_top`, after Memory
Usage.

## Keys

| Key | Action |
|---|---|
| `+` / `-` | redraw less or more often: the interval 1s longer or shorter, down to the collect interval |
| Tab, Right / Shift-Tab, Left | next or previous view |
| `w` / `W` | shorter or longer window: 5, 10, 30 or 60s |
| `r` | show the difference over the window, or current values |
| `a` | anonymise schema, table and user names, on or off |
| `z` | reset: windows start again from now |
| `h`, `?` | help, on or off |
| `q`, Esc, Ctrl-C | quit |
| Ctrl-Z | suspend: `fg` returns to ps-top |

## Tested versions

| Server  | Versions             |
|---------|----------------------|
| MySQL   | 8.0, 8.4, 9.7, 26.10 |
| MariaDB | 10.6, 11.8, 12.3     |

See `test/integration/README.md` for the Docker test harness.

## Licensing

BSD 2-Clause License

## Feedback

Feedback and patches welcome. I am especially interested in hearing
from you if you are using ps-top, or if you have ideas of how I can
better use other information from the `performance_schema` tables
to provide a more complete vision of what MySQL is doing or where
it's busy. The tool has been used by myself and colleagues and
helped quickly identify bottlenecks and problems in several systems.

Simon J Mudd
<sjmudd@pobox.com>

## Code Documentation

[pkg.go.dev/github.com/sjmudd/ps-top/v2](https://pkg.go.dev/github.com/sjmudd/ps-top/v2)

# ps-top integration test harness

Runs a real MariaDB/MySQL server in Docker, loads it (sysbench, or the Go
loadgen fallback on MySQL 9.x), and validates ps-top against it. Isolated
project/network (`ps-top-test`, `ps-top-test-net`) — does not touch other
containers on the host.

## Usage

```
make integration-up      # start db, wait for healthcheck
make integration-load    # sysbench prepare + run (~3 min default)
make integration-test    # go test -tags=integration ./app/...
make integration-down    # stop and remove
```

Or the full cycle: `make integration-full`.

## Config (env vars)

- `CONTAINER_IMAGE` — image:tag to run (default `mariadb:12.3`).
- `CONTAINER_HOST_PORT` — host port to map (default `3307`).
- `CONTAINER_ROOT_PASSWORD` — root password (default `rootpw`).
- `CONTAINER_HEALTHCHECK` — command to check readiness (default
  `healthcheck.sh --connect --innodb_initialized`, MariaDB's helper). For
  MySQL images, which don't ship that helper: `mysqladmin ping -h 127.0.0.1
  -uroot -p"$MYSQL_ROOT_PASSWORD"`.
- `SYSBENCH_DB_USER` / `SYSBENCH_DB_PASSWORD` — sysbench's db user (default
  `sysbench` / `sysbenchpw`), scoped to the `sbtest` database only.
- `SYSBENCH_SCRIPT` — sysbench Lua script (default `oltp_read_write`).
- `SYSBENCH_TABLES`, `SYSBENCH_TABLE_SIZE`, `SYSBENCH_THREADS`,
  `SYSBENCH_TIME` — workload shape (defaults: 8, 10000, 2, 60s).
- `LOAD_DRIVER` — `sysbench` (default) or `go` to use `load/loadgen/` instead:
  a small Go program using `go-sql-driver/mysql`, which supports
  `caching_sha2_password` natively. Needed on MySQL 9.x, where
  `mysql_native_password` was removed entirely and the sysbench Docker
  image's client library can't speak `caching_sha2_password` at all - see
  "Running against MySQL 9.x" below.
- `PSTOP_TEST_MAX_WAIT_SECONDS` — per-view max seconds `TestViewsRenderRealData`
  polls for a non-zero metric before failing (default 20).
- `PSTOP_TEST_LIVE_SAMPLES` — number of one-second samples
  `TestTableIoLatencyUpdatesLive` takes while checking the screen actually
  refreshes (default 5).

The ps-top monitoring user (`ps_top_monitor` / `pstoppw`, created in
`initdb/01-pstop-user.sh`) has exactly the grants README.md documents ps-top
needing: `SELECT` on `performance_schema.*`, `UPDATE` on
`setup_instruments`, and `PROCESS`. This is the user the integration
tests, and any manual `./ps-top` run against the container, should connect
as — it exercises the realistic restricted-grant case, not root.

The script uses each engine's default auth plugin: `mysql_native_password`
on MariaDB, `caching_sha2_password` on MySQL. ps-top's Go driver supports
both.

## Manual exploration

```
./ps-top --host=127.0.0.1 --port=${CONTAINER_HOST_PORT:-3307} \
  --user=ps_top_monitor --password=pstoppw
```

## Running against MySQL instead of MariaDB

```
CONTAINER_IMAGE=mysql:8.4 \
CONTAINER_HEALTHCHECK='mysqladmin ping -h 127.0.0.1 -uroot -p"$MYSQL_ROOT_PASSWORD"' \
COMPOSE_EXTRA_FILE=test/integration/docker-compose.mysql.yml \
  make integration-up
```

`docker-compose.mysql.yml` enables `mysql_native_password` at startup so the
sysbench client can authenticate (see below). Only needed for 8.x.

## Running against MySQL 9.x

MySQL 9.x removed `mysql_native_password` entirely (not just disabled, as
in 8.4) - `initdb/01-pstop-user.sh` handles this fine for `ps_top_monitor`
(caching_sha2_password), but the sysbench Docker image's client library
still can't speak that plugin, so sysbench itself cannot connect at all on
9.x. Don't use `docker-compose.mysql.yml`'s command override (it's now an
invalid startup flag and will prevent the server starting); use the Go
load generator instead:

```
CONTAINER_IMAGE=mysql:9.7 \
CONTAINER_HEALTHCHECK='mysqladmin ping -h 127.0.0.1 -uroot -p"$MYSQL_ROOT_PASSWORD"' \
  make integration-up

LOAD_DRIVER=go make integration-load
make integration-test
```

## Replication: a source and two replicas

`replication.sh` starts three servers of the same image
(`docker-compose.replication.yml`, project `ps-top-repl`, network
`ps-top-repl-net`), identical apart from `server_id`, and sets up
replication from `source` to `replica1` and `replica2`. It shows what
replication threads look like in the processlist: Binlog Dump threads on the
source, from two hosts, and the I/O, SQL and worker threads on each replica.

```
make repl-up CONTAINER_IMAGE=mysql:8.4 GTID=on
make repl-load      # Go load generator on the source, LOAD_SECONDS (30)
make repl-capture   # SHOW PROCESSLIST, performance_schema and INFORMATION_SCHEMA processlists
make repl-states    # counts of USER, COMMAND, STATE over SAMPLES (300)
make repl-down
```

The targets run `replication.sh up|load|capture|states|down`, which can also
be run directly. Settings are environment variables, the same on all three
servers. Unset, each keeps the server's default.

- `CONTAINER_IMAGE`: `mysql:<tag>` or `mariadb:<tag>` (default `mariadb:12.3`).
- `GTID`: `off` (default) replicates by binlog file and position, `on` by
  GTID (`gtid_mode=ON` and auto-positioning on MySQL,
  `MASTER_USE_GTID=slave_pos` on MariaDB).
- `BINLOG_FORMAT`: `ROW`, `STATEMENT` or `MIXED`. Unset keeps the server's
  default.
- `PARALLEL`: replica worker threads. Unset keeps the server's default.
- `SHOW_PROCESSLIST`: `ON` or `OFF`, sets
  `performance_schema_show_processlist` (MySQL only).
- `START_USER`: `root` (default) or `repl_admin`, the user that runs
  `CHANGE ...` and `START ...` on the replicas. `repl_admin` has only the
  privilege needed for that.
- `START_FROM`: `local` (default) runs them on the replica itself, `remote`
  from the source container over TCP, as a tool such as orchestrator does.
  On some MySQL versions (seen on 8.0.34, bug 113930) replica threads then
  show that user and host instead of `system user`.
- Host ports: `REPL_SOURCE_PORT` (3307), `REPL_REPLICA1_PORT` (3308),
  `REPL_REPLICA2_PORT` (3309).

`make repl-matrix` (`replication-matrix.sh`) runs every combination, one at
a time: GTID off and on, `ROW` and `STATEMENT`, replication started by root
locally and by `repl_admin` locally and remotely, and on MySQL
`performance_schema_show_processlist` `OFF` and `ON`, all with
`PARALLEL=4`. Each run starts the servers, loads the source while sampling
states, captures the processlists and stops the servers. It writes one log
per run and `summary.txt` to `MATRIX_OUT`:

```
make repl-matrix
make repl-matrix MATRIX_IMAGES="mysql:8.4 mariadb:12.3" MATRIX_OUT=/some/dir
```

The default images are MySQL 8.0, 8.4, 9.7 and 26.10 and MariaDB 10.6, 11.8
and 12.3: 132 runs, about 100 minutes.

The replication commands follow the server's version: `CHANGE REPLICATION
SOURCE TO` and `START REPLICA` from MySQL 8.0.23, `CHANGE MASTER TO` and
`START SLAVE` before that and on MariaDB. Each server runs initdb itself, so
replication starts from the source's position after initdb.

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

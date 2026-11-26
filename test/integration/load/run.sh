#!/bin/bash
# Runs a load against the db service started by docker-compose.yml in this
# directory: sysbench by default, or LOAD_DRIVER=go for the loadgen/
# fallback when sysbench's client can't authenticate (e.g. MySQL 9.x, which
# removed mysql_native_password - see loadgen/main.go).
#
# Adjust the workload via env vars, e.g.:
#   SYSBENCH_SCRIPT=oltp_update_index SYSBENCH_THREADS=8 ./run.sh
#   LOAD_DRIVER=go SYSBENCH_TIME=30 ./run.sh
set -euo pipefail

cd "$(dirname "$0")"

: "${LOAD_DRIVER:=sysbench}"

if [ "$LOAD_DRIVER" = "go" ]; then
	exec go run ./loadgen
fi

if [ -z "${SYSBENCH_IMAGE:-}" ]; then
	# Upstream severalnines/sysbench can't authenticate with
	# caching_sha2_password (its bundled libmariadb3 is missing the client
	# plugin - see sysbench/Dockerfile). Build our patched image locally
	# instead; set SYSBENCH_IMAGE to use a different image.
	SYSBENCH_IMAGE=ps-top-test-sysbench
	docker build -q -t "$SYSBENCH_IMAGE" sysbench >/dev/null
fi
: "${SYSBENCH_SCRIPT:=oltp_read_write}"
: "${SYSBENCH_TABLES:=8}"
: "${SYSBENCH_TABLE_SIZE:=10000}"
: "${SYSBENCH_THREADS:=2}"
: "${SYSBENCH_TIME:=60}"
: "${SYSBENCH_DB_USER:=sysbench}"
: "${SYSBENCH_DB_PASSWORD:=sysbenchpw}"

run_sysbench() {
	docker run --rm --network ps-top-test-net "${SYSBENCH_IMAGE}" \
		"/usr/share/sysbench/${SYSBENCH_SCRIPT}.lua" \
		--db-driver=mysql \
		--mysql-host=db \
		--mysql-port=3306 \
		--mysql-user="${SYSBENCH_DB_USER}" \
		--mysql-password="${SYSBENCH_DB_PASSWORD}" \
		--mysql-db=sbtest \
		--tables="${SYSBENCH_TABLES}" \
		--table-size="${SYSBENCH_TABLE_SIZE}" \
		--threads="${SYSBENCH_THREADS}" \
		--time="${SYSBENCH_TIME}" \
		"$@"
}

run_sysbench prepare
run_sysbench run

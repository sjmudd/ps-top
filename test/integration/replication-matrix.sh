#!/bin/bash
# Runs replication.sh over each image and combination of settings, one at a
# time, and writes one log per run to MATRIX_OUT, plus summary.txt.
#
#   ./replication-matrix.sh
#   MATRIX_IMAGES="mysql:8.4 mariadb:12.3" MATRIX_OUT=/some/dir ./replication-matrix.sh
#
# Per image: GTID off and on, binlog_format ROW and STATEMENT, replication
# started by root locally, repl_admin locally and repl_admin remotely, and on
# MySQL performance_schema_show_processlist OFF and ON.
# All runs use PARALLEL=4. Each run starts the servers, writes on the source
# for LOAD_SECONDS while sampling states, captures the processlists and stops
# the servers.
set -uo pipefail

cd "$(dirname "$0")"

: "${MATRIX_IMAGES:=mysql:8.0 mysql:8.4 mysql:9.7 mysql:26.10 mariadb:10.6 mariadb:11.8 mariadb:12.3}"
: "${MATRIX_OUT:=$PWD/replication-matrix.out}"
export LOAD_SECONDS=${LOAD_SECONDS:-15} SAMPLES=${SAMPLES:-100} PARALLEL=4

mkdir -p "$MATRIX_OUT"
summary=$MATRIX_OUT/summary.txt
: >"$summary"

run() {
	local name log
	name=$(echo "$CONTAINER_IMAGE-gtid_$GTID-$BINLOG_FORMAT-showpl_${SHOW_PROCESSLIST:-na}-$START_USER-$START_FROM" | tr ':' '_')
	log=$MATRIX_OUT/$name.log
	echo "######## $name"
	./replication.sh down >/dev/null 2>&1
	{
		./replication.sh up
		./replication.sh load >/dev/null 2>&1 &
		sleep 5
		./replication.sh states
		wait
		./replication.sh capture
	} >"$log" 2>&1
	local running errors
	running=$(grep -c -E '_Running: Yes' "$log")
	errors=$(grep -c -E 'ERROR|Last_.*Error: .' "$log")
	echo "$name running=$running/4 errors=$errors" | tee -a "$summary"
	./replication.sh down >/dev/null 2>&1
}

for CONTAINER_IMAGE in $MATRIX_IMAGES; do
	export CONTAINER_IMAGE
	show="na"
	case "$CONTAINER_IMAGE" in mysql:*) show="OFF ON" ;; esac
	for GTID in off on; do
		for BINLOG_FORMAT in ROW STATEMENT; do
			for showpl in $show; do
				if [ "$showpl" = na ]; then
					unset SHOW_PROCESSLIST
				else
					export SHOW_PROCESSLIST=$showpl
				fi
				for start in root:local repl_admin:local repl_admin:remote; do
					START_USER=${start%:*} START_FROM=${start#*:}
					export GTID BINLOG_FORMAT START_USER START_FROM
					run
				done
			done
		done
	done
done
echo "######## done: $summary"

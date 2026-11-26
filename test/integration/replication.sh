#!/bin/bash
# Starts a source and two replicas (docker-compose.replication.yml), sets up
# replication, and shows the processlist of each server.
#
#   CONTAINER_IMAGE=mysql:8.4 GTID=on ./replication.sh up
#   ./replication.sh load      # writes on the source for LOAD_SECONDS
#   ./replication.sh capture   # processlist of each server
#   ./replication.sh states    # counts of USER, COMMAND, STATE over SAMPLES
#   ./replication.sh down
#
# Settings, the same on all servers. Unset, each keeps the server's default.
#
#   GTID=off|on          off (default) replicates by binlog file and position,
#                        on by GTID: gtid_mode=ON and auto-positioning on
#                        MySQL, MASTER_USE_GTID=slave_pos on MariaDB.
#   BINLOG_FORMAT        ROW, STATEMENT or MIXED.
#   PARALLEL=N           replica worker threads: slave_parallel_threads on
#                        MariaDB, replica_parallel_workers (from 8.0.26) or
#                        slave_parallel_workers on MySQL.
#   SHOW_PROCESSLIST     ON or OFF: performance_schema_show_processlist
#                        (MySQL only).
#   START_USER           root (default) or repl_admin: the user that runs
#                        CHANGE ... and START ... on the replicas. repl_admin
#                        has only the privilege needed for that.
#   START_FROM           local (default): run them on the replica itself.
#                        remote: run them from the source container over TCP,
#                        as a tool such as orchestrator does (MySQL bugs 113930
#                        and 77293: replica threads may show that user and
#                        host).
#
# Replication commands use the syntax of the server's version: CHANGE
# REPLICATION SOURCE TO and START REPLICA from MySQL 8.0.23, else CHANGE
# MASTER TO and START SLAVE.
#
# Each server runs initdb itself, so replication starts from the source's
# position after initdb: those events are not replicated.
set -euo pipefail

cd "$(dirname "$0")"

: "${CONTAINER_IMAGE:=mariadb:12.3}"
: "${GTID:=off}"
: "${START_USER:=root}"
: "${START_FROM:=local}"
: "${CONTAINER_ROOT_PASSWORD:=rootpw}"
: "${LOAD_SECONDS:=30}"
: "${REPL_SOURCE_PORT:=3307}"
export CONTAINER_IMAGE CONTAINER_ROOT_PASSWORD REPL_SOURCE_PORT

REPL_USER=repl
REPL_PASSWORD=replpw
ADMIN_USER=repl_admin
ADMIN_PASSWORD=repladminpw
SERVERS="source replica1 replica2"

case "$CONTAINER_IMAGE" in
mariadb:*) engine=mariadb ;;
mysql:*) engine=mysql ;;
*)
	echo "CONTAINER_IMAGE must be mariadb:<tag> or mysql:<tag>" >&2
	exit 1
	;;
esac
case "$START_USER" in
root | "$ADMIN_USER") ;;
*)
	echo "START_USER must be root or $ADMIN_USER" >&2
	exit 1
	;;
esac
case "$START_FROM" in
local | remote) ;;
*)
	echo "START_FROM must be local or remote" >&2
	exit 1
	;;
esac

client=mysql
[ "$engine" = mariadb ] && client=mariadb

compose() {
	docker compose -f docker-compose.replication.yml "$@"
}

# sql runs statements as root on server $1.
sql() {
	local server=$1
	shift
	docker exec -i -e MYSQL_PWD="$CONTAINER_ROOT_PASSWORD" "ps-top-repl-$server" \
		"$client" -uroot "$@"
}

# admin_sql runs statements as START_USER on server $1: on the server itself,
# or with START_FROM=remote from the source container over TCP.
admin_sql() {
	local server=$1 password=$CONTAINER_ROOT_PASSWORD
	shift
	[ "$START_USER" = "$ADMIN_USER" ] && password=$ADMIN_PASSWORD
	if [ "$START_FROM" = remote ]; then
		docker exec -i -e MYSQL_PWD="$password" ps-top-repl-source \
			"$client" -h "$server" -u"$START_USER" "$@"
	else
		docker exec -i -e MYSQL_PWD="$password" "ps-top-repl-$server" \
			"$client" -u"$START_USER" "$@"
	fi
}

# val prints the result of statement $2 on server $1, with no column names.
val() {
	sql "$1" -N -e "$2"
}

# version_at_least returns true if server $1's version is at least $2.$3.$4.
version_at_least() {
	local v
	v=$(val "$1" "SELECT VERSION()")
	v=${v%%-*}
	[ "$(printf '%s\n%s\n' "$2.$3.$4" "$v" | sort -V | head -n1)" = "$2.$3.$4" ]
}

up() {
	local args=""
	if [ "$engine" = mysql ]; then
		export CONTAINER_HEALTHCHECK='mysqladmin ping -h 127.0.0.1 -uroot -p"$MYSQL_ROOT_PASSWORD"'
		[ "$GTID" = on ] && args="--gtid-mode=ON --enforce-gtid-consistency=ON"
	fi
	[ -n "${BINLOG_FORMAT:-}" ] && args="$args --binlog-format=$BINLOG_FORMAT"
	export REPL_ARGS="$args"
	compose up -d --wait

	local s
	if [ -n "${SHOW_PROCESSLIST:-}" ]; then
		if [ "$engine" != mysql ]; then
			echo "SHOW_PROCESSLIST is for MySQL only" >&2
			exit 1
		fi
		for s in $SERVERS; do
			sql "$s" -e "SET GLOBAL performance_schema_show_processlist = $SHOW_PROCESSLIST"
		done
	fi

	# Terms and statements for this version.
	local new=0 change start status reset show
	if [ "$engine" = mysql ] && version_at_least source 8 0 23; then
		new=1
	fi
	if [ $new = 1 ]; then
		change="CHANGE REPLICATION SOURCE TO SOURCE_HOST='source', SOURCE_USER='$REPL_USER', SOURCE_PASSWORD='$REPL_PASSWORD'"
		start="START REPLICA"
		show="SHOW REPLICA STATUS"
	else
		change="CHANGE MASTER TO MASTER_HOST='source', MASTER_USER='$REPL_USER', MASTER_PASSWORD='$REPL_PASSWORD'"
		start="START SLAVE"
		show="SHOW SLAVE STATUS"
	fi
	status="SHOW MASTER STATUS"
	reset="RESET MASTER"
	if [ "$engine" = mysql ] && version_at_least source 8 2 0; then
		status="SHOW BINARY LOG STATUS"
		reset="RESET BINARY LOGS AND GTIDS"
	fi
	# caching_sha2_password without TLS needs the source's public key.
	if [ "$engine" = mysql ] && version_at_least source 8 0 0; then
		if [ $new = 1 ]; then
			change="$change, GET_SOURCE_PUBLIC_KEY=1"
		else
			change="$change, GET_MASTER_PUBLIC_KEY=1"
		fi
	fi

	# Not logged, so the replicas do not replicate it.
	sql source -e "SET sql_log_bin = 0; CREATE USER '$REPL_USER'@'%' IDENTIFIED BY '$REPL_PASSWORD'; GRANT REPLICATION SLAVE ON *.* TO '$REPL_USER'@'%'"

	local pos file offset gtids
	if [ "$GTID" = on ]; then
		if [ "$engine" = mysql ]; then
			gtids=$(val source "SELECT @@GLOBAL.gtid_executed" | tr -d '\n\\')
			pos="SET GLOBAL gtid_purged = '$gtids'"
			if [ $new = 1 ]; then
				change="$change, SOURCE_AUTO_POSITION=1"
			else
				change="$change, MASTER_AUTO_POSITION=1"
			fi
		else
			gtids=$(val source "SELECT @@GLOBAL.gtid_binlog_pos")
			pos="SET GLOBAL gtid_slave_pos = '$gtids'"
			change="$change, MASTER_USE_GTID=slave_pos"
		fi
	else
		read -r file offset _ < <(val source "$status")
		pos="DO 0"
		if [ $new = 1 ]; then
			change="$change, SOURCE_LOG_FILE='$file', SOURCE_LOG_POS=$offset"
		else
			change="$change, MASTER_LOG_FILE='$file', MASTER_LOG_POS=$offset"
		fi
		[ "$engine" = mariadb ] && change="$change, MASTER_USE_GTID=no"
	fi

	local parallel="DO 0"
	if [ -n "${PARALLEL:-}" ]; then
		if [ "$engine" = mariadb ]; then
			parallel="SET GLOBAL slave_parallel_threads = $PARALLEL"
		elif version_at_least source 8 0 26; then
			parallel="SET GLOBAL replica_parallel_workers = $PARALLEL"
		else
			parallel="SET GLOBAL slave_parallel_workers = $PARALLEL"
		fi
	fi

	# The privilege to run CHANGE ... and START ...
	local priv="REPLICATION_SLAVE_ADMIN"
	if [ "$engine" = mariadb ]; then
		priv="REPLICATION SLAVE ADMIN"
	elif ! version_at_least source 8 0 0; then
		priv="SUPER"
	fi

	local r
	for r in replica1 replica2; do
		sql "$r" -e "$reset; $pos; $parallel"
		if [ "$START_USER" = "$ADMIN_USER" ]; then
			sql "$r" -e "SET sql_log_bin = 0; CREATE USER '$ADMIN_USER'@'%' IDENTIFIED BY '$ADMIN_PASSWORD'; GRANT $priv ON *.* TO '$ADMIN_USER'@'%'"
		fi
		echo "$r: as $START_USER ($START_FROM): $change"
		admin_sql "$r" -e "$change; $start"
	done
	sleep 3
	for r in replica1 replica2; do
		echo "== $r"
		sql "$r" -E -e "$show" | grep -E '_Running:|Last_.*Error:|Auto_Position|Using_Gtid'
	done
}

# load writes on the source for LOAD_SECONDS, from the host, with the Go
# load generator.
load() {
	CONTAINER_HOST_PORT=$REPL_SOURCE_PORT SYSBENCH_TIME=$LOAD_SECONDS \
		LOAD_DRIVER=go load/run.sh
}

# settings prints server $1's version and the settings above.
settings() {
	local q="CONCAT(VERSION(), ' binlog_format=', @@GLOBAL.binlog_format"
	if [ "$engine" = mysql ]; then
		q="$q, ' gtid_mode=', @@GLOBAL.gtid_mode, ' replica_parallel_workers=', @@GLOBAL.replica_parallel_workers"
		q="$q, ' performance_schema_show_processlist=', @@GLOBAL.performance_schema_show_processlist"
	else
		q="$q, ' slave_parallel_threads=', @@GLOBAL.slave_parallel_threads"
	fi
	val "$1" "SELECT $q, ' START_USER=$START_USER START_FROM=$START_FROM')"
}

capture() {
	local s cols="ID, USER, HOST, DB, COMMAND, TIME, STATE"
	for s in $SERVERS; do
		echo "== $s $(settings "$s")"
		if [ "$engine" = mysql ]; then
			echo "-- performance_schema.processlist"
			sql "$s" -t -e "SELECT $cols FROM performance_schema.processlist ORDER BY ID" || true
		fi
		echo "-- INFORMATION_SCHEMA.PROCESSLIST"
		sql "$s" -t -e "SELECT $cols FROM INFORMATION_SCHEMA.PROCESSLIST ORDER BY ID"
		echo "-- SHOW PROCESSLIST"
		sql "$s" -t -e "SHOW PROCESSLIST"
	done
}

# states samples USER, COMMAND and STATE of each server's other connections
# SAMPLES times and counts each combination: a single capture misses short
# states such as "Sending binlog event to ...". On MySQL it reads
# performance_schema.processlist, else INFORMATION_SCHEMA.PROCESSLIST.
states() {
	local s table=INFORMATION_SCHEMA.PROCESSLIST
	[ "$engine" = mysql ] && table=performance_schema.processlist
	local q="SELECT USER, COMMAND, STATE FROM $table WHERE ID <> CONNECTION_ID()"
	for s in $SERVERS; do
		echo "== $s ($table)"
		docker exec -e MYSQL_PWD="$CONTAINER_ROOT_PASSWORD" "ps-top-repl-$s" sh -c \
			"for i in \$(seq ${SAMPLES:-300}); do $client -uroot -N -e \"$q\"; done" | sort | uniq -c
	done
}

down() {
	compose down -v
}

case "${1:-}" in
up | load | capture | states | down) "$1" ;;
*)
	echo "usage: $0 up|load|capture|states|down" >&2
	exit 1
	;;
esac

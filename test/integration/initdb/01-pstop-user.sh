#!/bin/bash
# Creates the ps-top monitoring user with each engine's default auth plugin:
# mysql_native_password on MariaDB, caching_sha2_password on MySQL.
set -euo pipefail

PSTOP_USER="ps_top_monitor"
PSTOP_PASSWORD="pstoppw"
ROOT_PASSWORD="${MARIADB_ROOT_PASSWORD:-${MYSQL_ROOT_PASSWORD:-}}"

sql_client() {
	if command -v mariadb >/dev/null 2>&1; then
		MYSQL_PWD="$ROOT_PASSWORD" mariadb -uroot "$@"
	else
		MYSQL_PWD="$ROOT_PASSWORD" mysql -uroot "$@"
	fi
}

is_mariadb=$(sql_client -N -e "SELECT VERSION() LIKE '%MariaDB%'")

if [ "$is_mariadb" = "1" ]; then
	sql_client <<-SQL
		CREATE USER IF NOT EXISTS '${PSTOP_USER}'@'%' IDENTIFIED BY '${PSTOP_PASSWORD}';
		GRANT SELECT ON performance_schema.* TO '${PSTOP_USER}'@'%';
		GRANT UPDATE ON performance_schema.setup_instruments TO '${PSTOP_USER}'@'%';
		GRANT PROCESS ON *.* TO '${PSTOP_USER}'@'%';
	SQL
else
	sql_client <<-SQL
		CREATE USER IF NOT EXISTS '${PSTOP_USER}'@'%' IDENTIFIED WITH caching_sha2_password BY '${PSTOP_PASSWORD}';
		GRANT SELECT ON performance_schema.* TO '${PSTOP_USER}'@'%';
		GRANT UPDATE ON performance_schema.setup_instruments TO '${PSTOP_USER}'@'%';
		GRANT PROCESS ON *.* TO '${PSTOP_USER}'@'%';
	SQL

	# sysbench's client library can't load caching_sha2_password at all (it's
	# built against MariaDB's libmysqlclient); keep the sysbench account on
	# mysql_native_password so it can still connect. Requires
	# --mysql-native-password=ON at server startup (see docker-compose.mysql.yml).
	# mysql_native_password was removed entirely in MySQL 9.x (not just
	# disabled like 8.4), so this is a no-op there and sysbench simply can't
	# connect on that version - `|| true` so that doesn't take down the whole
	# container init (ps_top_monitor above is what actually matters).
	sql_client -e "ALTER USER 'sysbench'@'%' IDENTIFIED WITH mysql_native_password BY 'sysbenchpw'" || true
fi

#!/bin/bash
# Records SHOW CREATE TABLE for performance_schema tables, per server version,
# into docs/schema/<flavour>-<major.minor>/<table>.sql. Uses the harness in
# this directory; starts and removes one server per image, in turn.
#
# Usage: ./schema-dump.sh <image>... -- <table>...
#   ./schema-dump.sh mysql:9.7 mariadb:12.3 -- file_summary_by_instance
set -euo pipefail

cd "$(dirname "$0")"
out=../../docs/schema

images=()
while [ $# -gt 0 ] && [ "$1" != "--" ]; do
	images+=("$1")
	shift
done
[ "${1:-}" = "--" ] && shift
tables=("$@")
if [ ${#images[@]} -eq 0 ] || [ ${#tables[@]} -eq 0 ]; then
	echo "usage: $0 <image>... -- <table>..." >&2
	exit 2
fi

for image in "${images[@]}"; do
	export CONTAINER_IMAGE=$image
	case $image in
	mysql:*) export CONTAINER_HEALTHCHECK='mysqladmin ping -h 127.0.0.1 -uroot -p"$MYSQL_ROOT_PASSWORD"' ;;
	*) unset CONTAINER_HEALTHCHECK ;;
	esac
	docker compose up -d --wait >/dev/null 2>&1
	cli=$(docker exec ps-top-test-db sh -c 'command -v mariadb || command -v mysql')
	q() { docker exec ps-top-test-db "$cli" -uroot -prootpw -N -B -r -e "$1" 2>/dev/null; }

	version=$(q "SELECT @@version")
	flavour=mysql
	case $version in *MariaDB*) flavour=mariadb ;; esac
	dir=$out/$flavour-$(echo "$version" | cut -d. -f1,2)
	mkdir -p "$dir"
	for t in "${tables[@]}"; do
		# A qualified name (schema.table) is used as given; otherwise performance_schema.
		case $t in
		*.*) name=$t ;;
		*) name=performance_schema.$t ;;
		esac
		ddl=$(q "SHOW CREATE TABLE $name" || true)
		{
			echo "-- $version"
			if [ -n "$ddl" ]; then
				echo "$ddl" | sed -e '1s/^[^\t]*\t//' -e '$s/$/;/'
			else
				echo "-- $name not present"
			fi
		} >"$dir/$t.sql"
	done
	echo "$image: $version -> $dir"
	docker compose down -v >/dev/null 2>&1
done

// loadgen is a sysbench-alternative load generator using go-sql-driver/mysql,
// which supports caching_sha2_password natively. Used when the sysbench
// Docker image's client library can't authenticate (e.g. MySQL 9.x, which
// removed mysql_native_password entirely).
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func main() {
	host := getenv("CONTAINER_HOST", "127.0.0.1")
	port := getenvInt("CONTAINER_HOST_PORT", 3307)
	user := getenv("SYSBENCH_DB_USER", "sysbench")
	password := getenv("SYSBENCH_DB_PASSWORD", "sysbenchpw")
	dbname := getenv("SYSBENCH_DB", "sbtest")
	tables := getenvInt("SYSBENCH_TABLES", 8)
	seconds := getenvInt("SYSBENCH_TIME", 60)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, dbname)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	for i := 1; i <= tables; i++ {
		schema := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS sbtest%d (
			id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			k INT NOT NULL DEFAULT 0,
			c CHAR(120) NOT NULL DEFAULT '',
			pad CHAR(60) NOT NULL DEFAULT '',
			KEY k_%d (k)
		) ENGINE=InnoDB`, i, i)
		if _, err := db.Exec(schema); err != nil {
			log.Fatalf("create sbtest%d: %v", i, err)
		}
		if _, err := db.Exec(fmt.Sprintf("INSERT IGNORE INTO sbtest%d (id, k, c, pad) VALUES (1, 1, 'seed', 'seed')", i)); err != nil {
			log.Fatalf("seed sbtest%d: %v", i, err)
		}
	}

	log.Printf("loadgen: running for %ds against %d tables", seconds, tables)

	deadline := time.Now().Add(time.Duration(seconds) * time.Second)
	table := 1
	ops := 0
	for time.Now().Before(deadline) {
		_, _ = db.Exec(fmt.Sprintf("UPDATE sbtest%d SET k = k + 1 WHERE id = 1", table))
		_, _ = db.Exec(fmt.Sprintf("INSERT INTO sbtest%d (k, c, pad) VALUES (?, '', '')", table), table)
		var count int
		_ = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM sbtest%d", table)).Scan(&count)
		ops += 3
		table = table%tables + 1
		time.Sleep(20 * time.Millisecond)
	}

	log.Printf("loadgen: done, %d operations", ops)
}

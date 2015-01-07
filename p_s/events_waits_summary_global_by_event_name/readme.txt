This screen is inspired by this post:

http://www.percona.com/blog/2015/01/06/getting-mutex-information-from-mysqls-performance_schema/i

root@localhost [performance_schema]> select * from setup_instruments limit 10;
+---------------------------------------------------------+---------+-------+
| NAME                                                    | ENABLED | TIMED |
+---------------------------------------------------------+---------+-------+
| wait/synch/mutex/sql/TC_LOG_MMAP::LOCK_tc               | NO      | NO    |
| wait/synch/mutex/sql/LOCK_des_key_file                  | NO      | NO    |
| wait/synch/mutex/sql/MYSQL_BIN_LOG::LOCK_commit         | NO      | NO    |
| wait/synch/mutex/sql/MYSQL_BIN_LOG::LOCK_commit_queue   | NO      | NO    |
| wait/synch/mutex/sql/MYSQL_BIN_LOG::LOCK_done           | NO      | NO    |
| wait/synch/mutex/sql/MYSQL_BIN_LOG::LOCK_flush_queue    | NO      | NO    |
| wait/synch/mutex/sql/MYSQL_BIN_LOG::LOCK_index          | NO      | NO    |
| wait/synch/mutex/sql/MYSQL_BIN_LOG::LOCK_log            | NO      | NO    |
| wait/synch/mutex/sql/MYSQL_BIN_LOG::LOCK_binlog_end_pos | NO      | NO    |
| wait/synch/mutex/sql/MYSQL_BIN_LOG::LOCK_sync           | NO      | NO    |
+---------------------------------------------------------+---------+-------+
10 rows in set (0.00 sec)

root@localhost [performance_schema]> update setup_instruments set enabled = 'YES', timed = 'YES' where NAME like 'wait/synch/mutex/innodb/%';
Query OK, 49 rows affected (0.02 sec)
Rows matched: 49  Changed: 49  Warnings: 0


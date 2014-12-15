## pstop

pstop - a top-like program for MySQL

pstop is a program which collects information from MySQL 5.6+'s
performance_schema database and uses this information to display
server load in real-time. Data is shown by table or filename and
the metrics also show how this is split between select, insert,
update or delete activity.  User activity is now shown showing the
number of different hosts that connect with the same username and
the actiity of those users.

### Installation

Install and update this go package with `go get -u github.com/sjmudd/pstop`

### Dependencies

The following Non-core Go dependencies are:
* github.com/sjmudd/mysql_defaults_file for connecting to MySQL via
a defaults file.
* github.com/nsf/termbox-go a library for creating cross-platform
text-based interfaces.

### Configuration

Access to MySQL can be made by one of the following methods:
* Default: use a defaults-file named `~/.my.cnf`.
* use an explicit defaults-file with `--defaults-file=/path/to/.my.cnf`.
* connect to a host with `--host=somehost --port=999 --user=someuser --password=somepass`, or
* connect via a socket with `--socket=/path/to/mysql.sock --user=someuser --password=somepass`

The user if not specified will default to the contents of `$USER`.

### Grants

`pstop` needs `SELECT` access to `performance_schema` tables. It
will not run if access to the required tables is not available.

### Screens

pstop has 5 different screens which get updated by default every second.
* Latency mode: order table activity by the time waiting to perform operations on them.
* Ops (operations) mode: order table activity by the number of operations MySQL performs on them.
* I/O mode: show where MySQL is spending it's time in file I/O.
* Locks mode: show order based on table locks
* User mode: show ordering based on how long users are running
queries, or the number of connections they have to MySQL. This is
really missing a feature in MySQL (see: http://bugs.mysql.com/75156)
to provide higher resolution query times than seconds. It gives
some info but if the queries are very short then the integer runtime
in seconds makes the output far less interesting.

You can change the polling interval and switch between modes (see below).

### Keys

The following keys allow you to navigate around the different pstop displays or to change it's behaviour.

* h - gives you a help screen.
* - - reduce the poll interval by 1 second (minimum 1 second)
* + - increase the poll interval by 1 second
* q - quit
* t - toggle between showing the statistics since resetting pstop started or you explicitly reset them (with 'z') [REL] or showing the statistics as collected from MySQL [ABS].
* z - reset statistics. That is counters you see are relative to when you "reset" statistics.
* <tab> - change display modes between: latency, ops, file I/O, lock modes and user modes.
* left arrow - change to previous screen
* right arrow - change to next screen

### See also

See also:
* [BUGS](https://github.com/sjmudd/pstop/blob/master/BUGS) currently known issues
* [NEW_FEATURES](https://github.com/sjmudd/pstop/blob/master/NEW_FEATURES) which describe things that probably need looking at
* [screen_samples.txt](https://github.com/sjmudd/pstop/blob/master/screen_samples.txt) provides some sample output from my own system.

### Contributing

This program was started as a simple project to allow me (Simon) to learn
go, which I'd been following for a while, but hadn't used in earnest.
This probably shows in the code so suggestions on improvement are
most welcome.

You may find "Contributing to Open Source Git Repositories in Go"
by Katrina Owen to be useful:
https://blog.splice.com/contributing-open-source-git-repositories-go/

### Licensing

BSD 2-Clause License

### Feedback

Feedback and patches welcome. I am especially interested in hearing
from you if you are using pstop, or if you have ideas of how I can
better use other information from the performance_schema tables to
provide a more complete vision of what MySQL is doing or where it's
busy.

Simon J Mudd
<sjmudd@pobox.com>

### Code Documenton
[godoc.org/github.com/sjmudd/pstop](http://godoc.org/github.com/sjmudd/pstop)

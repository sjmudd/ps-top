## pstop

pstop - a top-like program for MySQL

pstop is a program which collects information from MySQL 5.6+'s
performance_schema database and uses this information to display
server load in real-time. Data is shown by table or filename and
the metrics also show how this is split between select, insert,
update or delete activity.  User activity is now shown showing the
number of different hosts that connect with the same username and
the activity of those users.  There are also statistics on mutex
and sql stage timings.

If required pstop can be used in stdout mode behaving similarly
to other tools like vmstat.

### Installation

Install and update this go package with `go get -u github.com/sjmudd/pstop`.

Note if you are looking for binaries for pstop then look here
http://gobuild.io/github.com/sjmudd/pstop as a possible location.
I have not tried this service but it looks ok.

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
The port if not specified will default to 3306.

### Grants

`pstop` needs `SELECT` access to `performance_schema` tables. It
will not run if access to the required tables is not available.

setup_instruments: To view the Mutex or Stage page pstop will try to
change the configuration if needed and if you have grants to do this.
If the server is --read-only or you don't have sufficient grants
and the changes can not be made these pages may be empty.
Pior to stopping pstop will restore the setup_instruments
configuration back to its original settings if it had successfully
updated the table when starting up.

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
in seconds makes the output far less interesting. Total idle time is also
shown as this gives an indication of perhaps overly long idle queries,
and the sum of the values here if there's a pile up may be interesting.
* Mutex mode: show the ordering by mutex latency [1].
* Stages mode: show the ordering by time in the different SQL query stages [1].

You can change the polling interval and switch between modes (see below).

[1] pstop will try to configure the mutex and staging settings in
setup_consumers if it can, and restore them when exiting if it
changes something.

### Keys

The following keys allow you to navigate around the different pstop displays or to change it's behaviour.

* h - gives you a help screen.
* - - reduce the poll interval by 1 second (minimum 1 second)
* + - increase the poll interval by 1 second
* q - quit
* t - toggle between showing the statistics since resetting pstop started or you explicitly reset them (with 'z') [REL] or showing the statistics as collected from MySQL [ABS].
* z - reset statistics. That is counters you see are relative to when you "reset" statistics.
* <tab> - change display modes between: latency, ops, file I/O, lock, user, mutex and stage modes.
* left arrow - change to previous screen
* right arrow - change to next screen

### Stdout mode

This mode is intended to be used for watching and maybe collecting data
from pstop using stdout as the output medium.

Relevant command line options are:

--count=<count>        Limit the number of iterations (default: runs forever)
--interval=<seconds>   Set the default poll interval (in seconds)
--limit=<rows>         Limit the number of lines of output (excluding headers)
--stdout               Send output to stdout (not a screen)
--view=<view>          Determine the view you want to see when pstop starts (default: table_io_latency)
                       Possible values: table_io_latency table_io_ops file_io_latency table_lock_latency
                       user_latency mutex_latency stages_latency

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

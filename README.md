## ps-top

ps-top - a top-like program for MySQL

ps-top is a program which collects information from MySQL 5.6+'s
performance_schema database and uses this information to display
server load in real-time. Data is shown by table or filename and
the metrics also show how this is split between select, insert,
update or delete activity.  User activity is now shown showing the
number of different hosts that connect with the same username and
the activity of those users.  There are also statistics on mutex
and sql stage timings.

ps-stats is a similar utility which provides output in stdout mode.

### Installation

Install and update this go package with `go get -u github.com/sjmudd/ps-top`.

Each binary should be built from the directory representing its name
under cmd/.

e.g.
* `cd cmd/ps-top && go build`
* `cd cmd/ps-stats && go build`

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

`ps-top` needs `SELECT` access to `performance_schema` tables. It
will not run if access to the required tables is not available.

setup_instruments: To view the Mutex or Stage page ps-top will try to
change the configuration if needed and if you have grants to do this.
If the server is --read-only or you don't have sufficient grants
and the changes can not be made these pages may be empty.
Pior to stopping ps-top will restore the setup_instruments
configuration back to its original settings if it had successfully
updated the table when starting up.

### Views

`ps-top` can show 5 different views of data, the views are updated every second by default.
The views are named:

* table_io_latency: Show activity by table by the time waiting to perform operations on them.
* table_io_ops: Show activity by number of operations MySQL performs on them.
* file_io_latency: Show where MySQL is spending it's time in file I/O.
* table_lock_latency: Show order based on table locks
* user_latency: Show ordering based on how long users are running
queries, or the number of connections they have to MySQL. This is
really missing a feature in MySQL (see: http://bugs.mysql.com/75156)
to provide higher resolution query times than seconds. It gives
some info but if the queries are very short then the integer runtime
in seconds makes the output far less interesting. Total idle time is also
shown as this gives an indication of perhaps overly long idle queries,
and the sum of the values here if there's a pile up may be interesting.
* mutex_latency: Show the ordering by mutex latency [1].
* stages_latency: Show the ordering by time in the different SQL query stages [1].

You can change the polling interval and switch between modes (see below).

[1] ps-top will try to configure the mutex and staging settings in
setup_consumers if it can, and restore them when exiting if it
changes something.

### Keys

When in `ps-top` mode the following keys allow you to navigate around the different ps-top displays or to change it's behaviour.

* h - gives you a help screen.
* - - reduce the poll interval by 1 second (minimum 1 second)
* + - increase the poll interval by 1 second
* q - quit
* t - toggle between showing the statistics since resetting ps-top started or you explicitly reset them (with 'z') [REL] or showing the statistics as collected from MySQL [ABS].
* z - reset statistics. That is counters you see are relative to when you "reset" statistics.
* <tab> - change display modes between: latency, ops, file I/O, lock, user, mutex and stage modes.
* left arrow - change to previous screen
* right arrow - change to next screen

### Stdout mode

`ps-stats` has the same views as `ps-top` but the output is sent periodically to stdout.
If you don't specify the view to use it will default to table_io_latency.
You can adjust the interval of collection and the number of times to collect
data in the same way as using vmstat. That is the first parameter is delay
(default 1 second) and the second parameter is the number of iterations to make,
which if not provided means run forever.
This mode is intended to be used for watching and maybe collecting data
from ps-top using stdout as the output medium.

Relevant command line options are:

--count=<count>       Limit the number of iterations (default: runs forever)
--interval=<seconds>  Set the default poll interval (in seconds)
--limit=<rows>        Limit the number of lines of output (excluding headers)
--stdout              Send output to stdout (not a screen)
--view=<view>         Determine the view you want to see when ps-top starts (default: table_io_latency)
                      Possible values: table_io_latency table_io_ops file_io_latency table_lock_latency
                      user_latency mutex_latency stages_latency

### See also

See also:
* [BUGS](https://github.com/sjmudd/ps-top/blob/master/BUGS) currently known issues
* [NEW_FEATURES](https://github.com/sjmudd/ps-top/blob/master/NEW_FEATURES) which describe things that probably need looking at
* [screen_samples.txt](https://github.com/sjmudd/ps-top/blob/master/screen_samples.txt) provides some sample output from my own system.

### Incompatible Changes

As of v0.5.0 the original utility was renamed from pstop which could
work in stdout or top mode info two utilities named ps-top and ps-stats.
This change of name was triggered to avoid the name conflict with 
the Oracle command pstop(1). See https://docs.oracle.com/cd/E19683-01/816-0210/6m6nb7mii/index.html.
While the two commands are not related it was felt better to avoid
the name overload, and while ps-top is reasonably young this change
should not yet cause too much trouble.

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
from you if you are using ps-top, or if you have ideas of how I can
better use other information from the performance_schema tables to
provide a more complete vision of what MySQL is doing or where it's
busy.

Simon J Mudd
<sjmudd@pobox.com>

### Code Documenton
[godoc.org/github.com/sjmudd/ps-top](http://godoc.org/github.com/sjmudd/ps-top)

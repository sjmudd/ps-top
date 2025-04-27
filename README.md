## ps-top - a top-like program for MySQL

ps-top is a program which collects information from MySQL 5.6+'s
performance_schema database and uses this information to display
server load in real-time. Data is shown by table or filename and
the metrics also show how this is split between select, insert,
update or delete activity.  User activity is now shown showing the
number of different hosts that connect with the same username and
the activity of those users.  There are also statistics on mutex
and sql stage timings.

### Installation

Install each binary by doing:
`$ go install github.com/sjmudd/ps-top@latest`

Check the version of go you are using as older versions of GO may
not work.

The sources will be downloaded together with the dependencies and
the binary will be built and installed into `$GOPATH/bin/`. If
this path is in your `PATH` setting then the program can be run
directly without having to specify any specific path.

### Configuration

Sometimes you may want to combine different tables together and show
the combined output. A typical example might be if you have lots
of similarly named tables. Should you want to do this you can
use the following configuration file `~/.pstoprc` to hole the
configuration information:

```
[munge]
<regexp_match> = <replacement_string>
_[0-9]{8}$ = _YYYYMMDD
_[0-9]{6}$ = _YYYYMM
```

#### MySQL Access

Access to MySQL can be made by one of the following methods:
* Default: use a defaults-file named `~/.my.cnf`.
* use an explicit defaults-file with `--defaults-file=/path/to/.my.cnf`.
* connect to a host with `--host=somehost --port=999 --user=someuser --password=somepass`, or
* connect via a socket with `--socket=/path/to/mysql.sock --user=someuser --password=somepass`
* to avoid the password being stored or provided as a command-line
  argument you can use `--askpass` which will request this from the
  user on startup

The user if not specified will default to the contents of `$USER`.
The port if not specified will default to 3306.

* If you use the command-line option `--use-environment` `ps-top`
will look for the credentials in the environment
variable `MYSQL_DSN` and connect with that.  This is a GO DSN and
is expected to be in the format:
`user:pass@tcp(host:port)/performance_schema` and currently ALL
fields must be filled in. With a suitable wrapper function this
allows you to access one of many different servers without making
the credentials visible on the command-line.

An example setting could be to use TLS which is not fully supported
at the moment with command-line parameters:


```
$ export MYSQL_DSN='user:pass@tcp(host:3306)/performance_schema?tls=skip-verify&allowCleartextPasswords=1'
$ ps-top
```

#### MySQL/MariaDB configuration

The `performance_schema` database **MUST** be enabled for `ps-top` to work.
By default on MySQL this is enabled, but on MariaDB >= 10.0.12 it is disabled.
So please check your settings. Simply configure in `/etc/my.cnf`:

`performance_schema = 1`

If you change this setting you'll need to restart MariaDB for it to take
effect.

### Grants

`ps-top` needs `SELECT` grants to access `performance_schema`
tables. It will not run if access is not available.

`setup_instruments`: To view `mutex_latency` or `stages_latency`
`ps-top` will try to change the configuration if needed and if you
have grants to do this.  If the server is `--read-only` or you do not
have sufficient grants to change these tables these views may be empty.
Pior to stopping `ps-top` will restore the `setup_instruments` configuration
back to its original settings if it had successfully updated the table
when starting up.

### Views

`ps-top` can show 7 different views of data, the views
are updated every second by default.  The views are named:

* `table_io_latency`: Show activity by table by the time waiting to perform operations on them.
* `table_io_ops`: Show activity by number of operations MySQL performs on them.
* `file_io_latency`: Show where MySQL is spending it's time in file I/O.
* `table_lock_latency`: Show order based on table locks
* `user_latency`: Show ordering based on how long users are running
queries, or the number of connections they have to MySQL. This is
really missing a feature in MySQL (see: [bug#75156](http://bugs.mysql.com/75156))
to provide higher resolution query times than seconds. It gives
some info but if the queries are very short then the integer runtime
in seconds makes the output far less interesting. Total idle time is also
shown as this gives an indication of perhaps overly long idle queries,
and the sum of the values here if there's a pile up may be interesting.
* `mutex_latency`: Show the ordering by mutex latency [1].
* `stages_latency`: Show the ordering by time in the different SQL query stages [1].

You can change the polling interval and switch between modes (see below).

[1] See Grants above. These views may appear empty if `setup_instruments` is not
configured correctly.

### Keys

When in `ps-top` mode the following keys allow you to navigate around the different ps-top displays or to change it's behaviour.

* h - gives you a help screen.
* - - reduce the poll interval by 1 second (minimum 1 second)
* + - increase the poll interval by 1 second
* q - quit
* t - toggle between showing the statistics since resetting ps-top started or you explicitly reset them (with 'z') [REL] or showing the statistics as collected from MySQL [ABS].
* z - reset statistics. That is counters you see are relative to when you "reset" statistics.
* `<tab>` - change display modes between: latency, ops, file I/O, lock, user, mutex, stages and memory modes.
* left arrow - change to previous screen
* right arrow - change to next screen

### See also

See also:
* [screen_samples.txt](https://github.com/sjmudd/ps-top/blob/master/screen_samples.txt) provides some sample output from my own system.

### Contributing

This program was started as a simple project to allow me to learn
go, which I'd been following for a while, but hadn't used in earnest.
This probably shows in the code so suggestions on improvement are
most welcome.

### Licensing

BSD 2-Clause License

### Feedback

Feedback and patches welcome. I am especially interested in hearing
from you if you are using ps-top, or if you have ideas of how I can
better use other information from the `performance_schema` tables
to provide a more complete vision of what MySQL is doing or where
it's busy.  The tool has been used by myself and colleagues and
helped quickly identify bottlenecks and problems in several systems.

Simon J Mudd
<sjmudd@pobox.com>

### Code Documenton
[godoc.org/github.com/sjmudd/ps-top](http://godoc.org/github.com/sjmudd/ps-top)

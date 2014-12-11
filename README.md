## pstop

pstop - a top-like program for MySQL

pstop is a program which collects information from MySQL 5.6+'s
performance_schema database and uses this information to display
server load in real-time. Data is shown by table or filename and
the metrics also show how this is split between select, insert,
update or delete activity.  User activity is now shown showing the
number of different hosts that connect with the same username and
the actiity of those users.

This program was started as a simple project to allow me to learn
go, which I'd been following for a while, but hadn't used in earnest.
This probably shows in the code so suggestions on improvement are
most welcome.

### Installation

Install and update this go package with `go get -u github.com/sjmudd/pstop`

### Configuration

Access to MySQL is currently via a defaults-file which is assumed
to be ~/.my.cnf. This should probably be made more configurable.
If you see a need for this please let me know.

### Grants

Do not forget to ensure that the MySQL user you configure has access
to the performance_schema tables.

### Screens

pstop has 5 different screens:
* Latency mode: order table activity by the time waiting to perform operations on them.
* Ops (operations) mode: order table activity by the number of operations MySQL performs on them.
* I/O mode: show where MySQL is spending it's time in file I/O.
* Locks mode: show order based on table locks
* User mode: show ordering based on how long users are running queries, or the number of connections they have to MySQL.

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

See also BUGS and NEW_FEATURES which describe things that probably
need looking at and screen_samples.txt which provides some sample
output from my own system.

### Feedback

Feedback and patches welcome.

Simon J Mudd
<sjmudd@pobox.com>

### Code Documenton
[godoc.org/github.com/sjmudd/pstop](http://godoc.org/github.com/sjmudd/pstop)

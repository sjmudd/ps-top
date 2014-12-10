pstop
=====

pstop - a top-like program for MySQL

pstop is a program which collects information from MySQL 5.6+'s
performance_schema database and uses this information to display server
load in real-time. Data is shown by table or filename and the
metrics also show how this is split between select, insert, update or
delete activity.  User activity is now shown showing the number of
different hosts that connect with the same username and the actiity
of those users.

This program was started as a simple project to allow me to learn go,
which I'd been following for a while, but hadn't used in earnest.  This
probably shows in the code so suggestions on improvement are most welcome.

Access to MySQL is currently via a defaults-file which is assumed to be
~/.my.cnf. I should probably make this more configurable.

Do not forget to ensure that the MySQL user you configure has access
to the performance_schema tables.

See also BUGS and NEW_FEATURES which describe things that probably need
looking at, keys.txt which describes the keys used inside pstop, and
screen_samples.txt which provides some sample output from my own system.

Feedback and patches welcome.

Simon J Mudd
<sjmudd@pobox.com>

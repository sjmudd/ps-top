## mysql_defaults_file

Access in Go to MySQL via a defaults_file.

If using the MySQL command line utilities such as `mysql` or
`mysqladmin` you can provide a defaults-file option which stores
the credentials of the MySQL server you want to connect to. If no
specific defaults file is mentioned these utilities look in `~/.my.cnf`
for this file.

The Go sql interface does not support this functionality yet it can
be quite convenient as it avoids the need to explicitly provide credentials.

This small module fills in that gap by providing a function to allow you
to connect to MySQL using a specified defaults-file, or using the
`~/.my.cnf` if you do not specify a defaults-file path.

There is also a function BuildDSN which allows you to build up a Go
dsn for MySQL using various string to string map components such as
host, user, password, port or socket, and thus removing the need to
figure out the Go specific DSN format.

Both these functions are used by [pstop](http://github.com/sjmudd/pstop)
to simplify the connectivity and have been split off from it as they
may be useful for other programs that connect to  MySQL.

### Usage

Usage:

```
import (
	...
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sjmudd/mysql_defaults_file"
	...
)

// open the connection to the database using the default defaults-file.
dbh, err = mysql_defaults_file.OpenUsingDefaultsFile("mysql", "", "performance_schema")
```

The errors you get back will be the same as calling `sql.Open( "mysql",..... )`.

### Licensing

BSD 2-Clause License

### Feedback

Feedback and patches welcome.

Simon J Mudd
<sjmudd@pobox.com>

### Code Documenton
[godoc.org/github.com/sjmudd/mysql_defaults_file](http://godoc.org/github.com/sjmudd/mysql_defaults_file)

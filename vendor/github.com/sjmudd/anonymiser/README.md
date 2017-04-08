# anonymiser
Anonymise some values for showing to the "public"

# Description.

Allows a group of strings to be anonymised that is converted into
another set of string values. The anonymised strings consist of
the group name plus a number which increases with each new string
that needs anonymising.

Can be used to convert private names into a public anonymous set (e.g. [`ps-top`](https://godoc.org/github.com/sjmudd/ps-top]))

In `ps-top` I wanted to anonymise the host, database and table names
which were shown as they might expose internal information to third parties.
This package made that easy.

There is basically one routine `anonymise.Anonymise( "group", "some_name" )`.

The first parameter is the name of the group of strings to be
anonyised, The second parameter is the name to anonymise and basically
each new name gets an id starting at 1. This id is added to the end
of the group name and that's what is returned as the anonymised
name.

e.g.
To anonymise some database names:
* anonymise.Anonymise( "db",    "my_db" )   --> db1
* anonymise.Anonymise( "db",    "otherdb" ) --> db2
* anonymise.Anonymise( "db",    "otherdb" ) --> db2
* anonymise.Anonymise( "db",    "my_db" )   --> db1

To anonymise some table names:
* anonymise.Anonymise( "table", "important_name" )  --> table1
* anonymise.Anonymise( "table", "something_else" )  --> table2
* anonymise.Anonymise( "table", "important_name" )  --> table1

You can use as many prefixes as you like.

I guess in real code you'd do something like this:
```
var secret []string { ... } // holds strings of secrent information (maybe with duplicates)

... // fill secret with useful data

for i := range secret {
	fmt.Println( "secret:", secret[i], "==>", anonymise.Anonymise( "anonymised", secret[i] ) )
}
``` 

# Installation

Install by doing:
* `go get github.com/sjmudd/ps-top/anonymiser`

# Documentation

Documentation can be found using `godoc` or at [https://godoc.org/github.com/sjmudd/anonymiser]

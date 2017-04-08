package anonymiser

/*
Designed to anonymise a group of strings.
We provide a group name and string and get back an anonymised string based on the group name.
Can be used to convert private names into a public anonymous set (e.g. ps-top)
e.g.
Anonymise( "table", "tablea" )  --> table1
Anonymise( "table", "tableb" )  --> table2
Anonymise( "table", "tablea" )  --> table1
Anonymise( "db",    "my_db" )   --> db1
Anonymise( "db",    "otherdb" ) --> db2
Anonymise( "db",    "otherdb" ) --> db2
Anonymise( "db",    "my_db" )   --> db1
*/

var (
	groupMap map[string]*onegroup
	enabled  bool
)

// initialise structures - a global per app set of string groups
func init() {
	enabled = true
	Clear()
}

// Clear removes any previous data that might have been stored.
func Clear() {
	// no need to clean up explicitly if we had old data?
	// I guess go garbage collects but might be nice to do this???
	groupMap = make(map[string]*onegroup)
}

// Enable the anonymiser. We provide a boolean to signal intent
func Enable(setEnabled bool) {
	enabled = setEnabled
}

// Enabled returns if anonymising is enabled
func Enabled() bool {
	return enabled
}

// Anonymise takes a group and name and returns a string consisting
// of the group plus a number. If the string has been seen before
// then the same name is returned.  Use different groups if you want
// to store different sets of names.
func Anonymise(group, name string) string {
	if !enabled {
		return name
	}
	if name == "" {
		return name // empty string shouldn't be anonymised I think.
	}
	// does the group exist?
	if _, ok := groupMap[group]; !ok {
		newGroup := &onegroup{group: group, id: make(map[string]int)}
		newGroup.add(name)
		groupMap[group] = newGroup
	}

	// does the entry exist?
	return groupMap[group].name(name)
}

// expose for testing but should not really otherwise.
// func Self() map[string]*onegroup {
// 	return groupMap
// }

// Groups returns a slice of strings with the known groups
func Groups() []string {
	groups := make([]string, 0)
	for grp := range groupMap {
		groups = append(groups, grp)
	}

	return groups
}

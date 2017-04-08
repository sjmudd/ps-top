package anonymiser

import (
	"fmt"
)

type onegroup struct {
	group string
	last  int
	id    map[string]int
}

// does the name exist already?
func (a onegroup) exists(name string) bool {
	_, ok := a.id[name]
	return ok
}

// return the anonymised name
func (a *onegroup) name(orig string) string {
	if a.exists(orig) {
		return fmt.Sprintf("%s%d", a.group, a.id[orig])
	}
	return a.add(orig)
}

// add a new value and return the anonymised name
func (a *onegroup) add(orig string) string {
	a.last++
	a.id[orig] = a.last
	return a.name(orig)
}

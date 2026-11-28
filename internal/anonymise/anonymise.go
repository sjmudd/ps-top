// Package anonymise replaces the names of objects, such as schemas, tables
// and users, with anonymous equivalents, so output can be shared without
// showing them.
package anonymise

import (
	"log/slog"
	"strconv"
	"sync"
)

// Anonymiser replaces the name of an object, such as a schema, table or
// user, with an anonymous equivalent: Anonymise("shop", "schema") is
// "schema1", and the next schema seen is "schema2". A name keeps its
// anonymous equivalent for the life of the Anonymiser, so repeated output
// stays comparable. Each group (e.g. schema, table, user) is numbered on its
// own, in the order names are first seen. It is safe for concurrent use.
type Anonymiser struct {
	mu       sync.Mutex
	groups   map[string]map[string]int  // group to name to number
	excluded map[string]map[string]bool // group to names left as they are
}

// New returns an Anonymiser that has seen no names.
func New() *Anonymiser {
	return &Anonymiser{groups: map[string]map[string]int{}, excluded: map[string]map[string]bool{}}
}

// Exclude leaves name as it is in group: Anonymise returns it unchanged,
// and it takes no number. Call it before name is first anonymised, else
// output already made shows the name's anonymous equivalent.
func (a *Anonymiser) Exclude(name, group string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.excluded[group] == nil {
		a.excluded[group] = map[string]bool{}
	}
	a.excluded[group][name] = true
}

// Excluded reports whether name is left as it is in group (see Exclude).
func (a *Anonymiser) Excluded(name, group string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.excluded[group][name]
}

// Anonymise returns name's anonymous equivalent in group: the group
// followed by the name's number in it. An empty name, or one excluded in
// group, is returned as it is.
func (a *Anonymiser) Anonymise(name, group string) string {
	if name == "" {
		return ""
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.excluded[group][name] {
		return name
	}
	ids := a.groups[group]
	if ids == nil {
		ids = map[string]int{}
		a.groups[group] = ids
	}
	id, ok := ids[name]
	if !ok {
		id = len(ids) + 1
		ids[name] = id
		slog.Debug("anonymised name", "group", group, "name", name, "alias", group+strconv.Itoa(id))
	}
	return group + strconv.Itoa(id)
}

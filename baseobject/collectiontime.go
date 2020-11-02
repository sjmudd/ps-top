// Package baseobject contains the library
// routines for base stuff of an object
package baseobject

import (
	"time"
)

// CollectionTime holds information about the first and last collection times
type CollectionTime struct {
	first time.Time
	last  time.Time
}

// LastTime returns the last time a collection took place
func (c CollectionTime) LastTime() time.Time {
	return c.last
}

// FirstTime returns the first time a collection took place
func (c CollectionTime) FirstTime() time.Time {
	return c.first
}

// SetFirstTime sets the first time a collection happened
func (c *CollectionTime) SetFirstTime(first time.Time) {
	c.first = first
}

// SetLastTime sets the last time a collection happened
func (c *CollectionTime) SetLastTime(last time.Time) {
	c.last = last
}

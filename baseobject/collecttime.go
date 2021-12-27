// Package baseobject contains the library
// routines for base stuff of an object
package baseobject

import (
	"time"
)

// CollectTime records the first and last time we have collected data
type CollectTime struct {
	FirstCollected time.Time // the first collection time (for relative data)
	LastCollected  time.Time // the last collection time
}

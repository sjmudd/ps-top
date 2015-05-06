// manage the time the statistics were taken
package p_s

import (
	"time"
)

// object to hold the last collection time
type CollectionTime struct {
	collection_time time.Time
}

// reflect we've just collected statistics
func (t *CollectionTime) SetCollected() {
	t.collection_time = time.Now()
}

// return the last time we collected statistics
func (t CollectionTime) Last() time.Time {
	return t.collection_time
}

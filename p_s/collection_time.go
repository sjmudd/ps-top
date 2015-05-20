package p_s

import (
	"time"
)

// CollectionTime stores the time of the last collection time
type CollectionTime struct {
	collectionTime time.Time
}

// SetCollected records the time the data was collected (now)
func (t *CollectionTime) SetCollected() {
	t.collectionTime = time.Now()
}

// Last return the last time we collected statistics
func (t CollectionTime) Last() time.Time {
	return t.collectionTime
}

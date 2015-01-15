// manage the time the statistics were taken
package p_s

import (
	"time"
)

// object to hold the last collection time
type InitialTime struct {
	initial_time time.Time
}

// reflect we've just collected statistics
func (t *InitialTime) SetNow() {
	t.initial_time = time.Now()
}

// return the last time we collected statistics
func (t InitialTime) Last() time.Time {
	return t.initial_time
}

package performance_schema

import (
	"time"
)

type InitialTime struct {
	initial_time time.Time
}

func (t *InitialTime) SetNow() {
	t.initial_time = time.Now()
}

func (t InitialTime) Last() time.Time {
	return t.initial_time
}

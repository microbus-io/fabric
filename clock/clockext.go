package clock

import "time"

// NewMockAt returns an instance of a mock clock initialized to a given time.
func NewMockAt(time time.Time) *Mock {
	return &Mock{now: time}
}

// NewMockAtNow returns an instance of a mock clock initialized to the current real time.
func NewMockAtNow() *Mock {
	return &Mock{now: time.Now()}
}

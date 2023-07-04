/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package clock

import "time"

// NewMockAt returns an instance of a mock clock initialized to a given time.
func NewMockAt(t time.Time) *Mock {
	return &Mock{now: t}
}

// NewMockAtNow returns an instance of a mock clock initialized to the current real time.
func NewMockAtNow() *Mock {
	return &Mock{now: time.Now()}
}

// NewMockAtNow returns an instance of a mock clock initialized to a specified date.
func NewMockAtDate(year int, month time.Month, day, hour, min, sec, nsec int, loc *time.Location) *Mock {
	return &Mock{now: time.Date(year, month, day, hour, min, sec, nsec, loc)}
}

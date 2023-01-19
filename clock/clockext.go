/*
Copyright 2023 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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

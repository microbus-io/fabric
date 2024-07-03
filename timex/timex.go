/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

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

package timex

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"
)

// Ensure interfaces
var (
	_ = json.Marshaler(&Timex{})
	_ = json.Unmarshaler(&Timex{})
	_ = driver.Valuer(&Timex{})
	_ = sql.Scanner(&Timex{})
)

/*
Timex is an extension of the standard time.Time.
It improves on parsing and overrides JSON serialization and database formatting for the zero time.

To create a new timex.Timex from time.Time:

	timex := New(stdTime)

To obtain time.Time from a Timex:

	stdTime := timex.Time
*/
type Timex struct {
	time.Time `dv8:"main"`
}

// New creates a new timex from a standard time.Time.
func New(t time.Time) Timex {
	return Timex{t}
}

/*
Date returns the Time corresponding to

	yyyy-mm-dd hh:mm:ss + nsec nanoseconds

in the appropriate zone for that time in the given location.

The month, day, hour, min, sec, and nsec values may be outside
their usual ranges and will be normalized during the conversion.
For example, October 32 converts to November 1.

A daylight savings time transition skips or repeats times.
For example, in the United States, March 13, 2011 2:15am never occurred,
while November 6, 2011 1:15am occurred twice. In such cases, the
choice of time zone, and therefore the time, is not well-defined.
Date returns a time that is correct in one of the two zones involved
in the transition, but it does not guarantee which.

Date panics if loc is nil.
*/
func Date(year int, month time.Month, day, hour, min, sec, nsec int, loc *time.Location) Timex {
	tm := time.Date(year, month, day, hour, min, sec, nsec, loc)
	return Timex{tm}
}

// Now returns the current local time.
func Now() Timex {
	return Timex{time.Now()}
}

// inferLayout infers the layout of a time string given its length.
// RFC3339Nano "2006-01-02T15:04:05.999999999Z07:00",
// RFC3339 "2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05", "2006-01-02 15:04:05",
// or "2006-01-02".
func inferLayout(value string) (layout string) {
	if len(value) == 10 &&
		value[4] == '-' && value[7] == '-' {
		layout = "2006-01-02"
	} else if len(value) == 19 &&
		value[4] == '-' && value[7] == '-' &&
		value[10] == 'T' && value[13] == ':' && value[16] == ':' {
		layout = "2006-01-02T15:04:05"
	} else if len(value) == 19 &&
		value[4] == '-' && value[7] == '-' &&
		value[10] == ' ' && value[13] == ':' && value[16] == ':' {
		layout = "2006-01-02 15:04:05"
	} else if len(value) >= 20 &&
		value[4] == '-' && value[7] == '-' &&
		value[10] == 'T' && value[13] == ':' && value[16] == ':' {
		layout = time.RFC3339Nano
		if value[19] != '.' {
			layout = time.RFC3339
		}
	}
	return layout
}

// Parse creates a new timex from a string.
// The empty string is parsed to the zero time. Layout is optional.
// If not provided, layout is assumed to be RFC3339Nano "2006-01-02T15:04:05.999999999Z07:00",
// RFC3339 "2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05", "2006-01-02 15:04:05",
// or "2006-01-02" based on the length of the value.
func Parse(layout string, value string) (Timex, error) {
	if value == "" {
		return Timex{}, nil
	}
	if layout == "" {
		layout = inferLayout(value)
	}
	t, err := time.Parse(layout, value)
	return Timex{t}, err
}

// MustParse is the same as Parse but panics on error.
func MustParse(layout string, value string) Timex {
	tx, err := Parse(layout, value)
	if err != nil {
		panic(err)
	}
	return tx
}

// ParseInLocation is like Parse but differs in two important ways.
// First, in the absence of time zone information, Parse interprets a time as UTC;
// ParseInLocation interprets the time as in the given location.
// Second, when given a zone offset or abbreviation, Parse tries to match it
// against the Local location; ParseInLocation uses the given location.
func ParseInLocation(layout string, value string, loc *time.Location) (Timex, error) {
	if value == "" {
		return Timex{}, nil
	}
	if layout == "" {
		layout = inferLayout(value)
	}
	t, err := time.ParseInLocation(layout, value, loc)
	return Timex{t}, err
}

// MustParse is the same as Parse but panics on error.
func MustParseInLocation(layout string, value string, loc *time.Location) Timex {
	tx, err := ParseInLocation(layout, value, loc)
	if err != nil {
		panic(err)
	}
	return tx
}

// Since returns the time elapsed since t.
// It is shorthand for timex.Now().Sub(t).
func Since(t Timex) time.Duration {
	return time.Since(t.Time)
}

// Unix returns the local Time corresponding to the given Unix time,
// sec seconds and nsec nanoseconds since January 1, 1970 UTC.
// It is valid to pass nsec outside the range [0, 999999999].
// Not all sec values have a corresponding time value. One such
// value is 1<<63-1 (the largest int64 value).
func Unix(sec int64, nsec int64) Timex {
	tm := time.Unix(sec, nsec)
	return Timex{tm}
}

// UnixMicro returns the local Time corresponding to the given Unix time,
// usec microseconds since January 1, 1970 UTC.
func UnixMicro(usec int64) Timex {
	tm := time.UnixMicro(usec)
	return Timex{tm}
}

// UnixMilli returns the local Time corresponding to the given Unix time,
// msec milliseconds since January 1, 1970 UTC.
func UnixMilli(msec int64) Timex {
	tm := time.UnixMilli(msec)
	return Timex{tm}
}

// Until returns the duration until t.
// It is shorthand for t.Sub(timex.Now()).
func Until(t Timex) time.Duration {
	return time.Until(t.Time)
}

// Add returns the time t+d.
func (tx Timex) Add(d time.Duration) Timex {
	tm := tx.Time.Add(d)
	return Timex{tm}
}

// AddDate returns the time corresponding to adding the
// given number of years, months, and days to t.
// For example, AddDate(-1, 2, 3) applied to January 1, 2011
// returns March 4, 2010.
//
// AddDate normalizes its result in the same way that Date does,
// so, for example, adding one month to October 31 yields
// December 1, the normalized form for November 31.
func (tx Timex) AddDate(years int, months int, days int) Timex {
	tm := tx.Time.AddDate(years, months, days)
	return Timex{tm}
}

// After reports whether the time instant t is after u.
func (tx Timex) After(u Timex) bool {
	return tx.Time.After(u.Time)
}

// Before reports whether the time instant t is before u.
func (tx Timex) Before(u Timex) bool {
	return tx.Time.Before(u.Time)
}

// Compare compares the time instant t with u. If t is before u, it returns -1;
// if t is after u, it returns +1; if they're the same, it returns 0.
func (tx Timex) Compare(u Timex) int {
	return tx.Time.Compare(u.Time)
}

// Equal reports whether t and u represent the same time instant.
// Two times can be equal even if they are in different locations.
// For example, 6:00 +0200 and 4:00 UTC are Equal.
// See the documentation on the Time type for the pitfalls of using == with
// Time values; most code should use Equal instead.
func (tx Timex) Equal(u Timex) bool {
	return tx.Time.Equal(u.Time)
}

// Format returns a textual representation of the time value formatted according
// to the layout defined by the argument. See the documentation for the
// constant called Layout to see how to represent the layout format.
//
// The executable example for Time.Format demonstrates the working
// of the layout string in detail and is a good reference.
//
// The zero time is formatted as an empty string.
func (tx Timex) Format(layout string) string {
	if tx.Time.IsZero() {
		return ""
	}
	return tx.Time.Format(layout)
}

// Local returns t with the location set to local time.
func (tx Timex) Local() Timex {
	tm := tx.Time.Local()
	return Timex{tm}
}

// In returns a copy of t representing the same time instant, but
// with the copy's location information set to loc for display
// purposes.
//
// In panics if loc is nil.
func (tx Timex) In(loc *time.Location) Timex {
	return New(tx.Time.In(loc))
}

// MarshalJSON overrides JSON serialization of the zero value to null.
func (tx Timex) MarshalJSON() ([]byte, error) {
	if tx.IsZero() {
		return []byte("null"), nil
	}
	return tx.Time.MarshalJSON()
}

// MarshalYAML overrides YAML serialization of the zero value to null.
func (tx Timex) MarshalYAML() (interface{}, error) {
	if tx.IsZero() {
		return nil, nil
	}
	b, err := tx.Time.MarshalJSON()
	if err != nil {
		return nil, err // No trace
	}
	return string(b[1 : len(b)-1]), nil
}

// Round returns the result of rounding t to the nearest multiple of d (since the zero time).
// The rounding behavior for halfway values is to round up.
// If d <= 0, Round returns t stripped of any monotonic clock reading but otherwise unchanged.
//
// Round operates on the time as an absolute duration since the
// zero time; it does not operate on the presentation form of the
// time. Thus, Round(Hour) may return a time with a non-zero
// minute, depending on the time's Location.
func (tx Timex) Round(d time.Duration) Timex {
	tm := tx.Time.Round(d)
	return Timex{tm}
}

// Sub returns the duration t-u. If the result exceeds the maximum (or minimum)
// value that can be stored in a Duration, the maximum (or minimum) duration
// will be returned.
// To compute t-d for a duration d, use t.Add(-d).
func (tx Timex) Sub(u Timex) time.Duration {
	return tx.Time.Sub(u.Time)
}

// Truncate returns the result of rounding t down to a multiple of d (since the zero time).
// If d <= 0, Truncate returns t stripped of any monotonic clock reading but otherwise unchanged.
//
// Truncate operates on the time as an absolute duration since the
// zero time; it does not operate on the presentation form of the
// time. Thus, Truncate(Hour) may return a time with a non-zero
// minute, depending on the time's Location.
func (tx Timex) Truncate(d time.Duration) Timex {
	tm := tx.Time.Truncate(d)
	return Timex{tm}
}

// UTC returns t with the location set to UTC.
func (tx Timex) UTC() Timex {
	tm := tx.Time.UTC()
	return Timex{tm}
}

// UnmarshalJSON overrides JSON deserialization, interpreting "null" and "" as the zero time.
func (tx *Timex) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "" || s == "null" {
		tx.Time = time.Time{}
		return nil
	}
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	p, err := Parse("", s)
	if err != nil {
		return err // No trace
	}
	tx.Time = p.Time
	return nil
}

// UnmarshalYAML overrides YAML deserialization, interpreting "null" and "" as the zero time.
func (tx *Timex) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err // No trace
	}
	p, err := Parse("", s)
	if err != nil {
		return err // No trace
	}
	*tx = p
	return nil
}

// ZoneBounds returns the bounds of the time zone in effect at time t.
// The zone begins at start and the next zone begins at end.
// If the zone begins at the beginning of time, start will be returned as a zero Time.
// If the zone goes on forever, end will be returned as a zero Time.
// The Location of the returned times will be the same as t.
func (tx Timex) ZoneBounds() (start, end Timex) {
	tm1, tm2 := tx.Time.ZoneBounds()
	return Timex{tm1}, Timex{tm2}
}

// Scan implements the Scanner interface used by SQL.
func (tx *Timex) Scan(value interface{}) error {
	tm, ok := value.(time.Time)
	if ok {
		tx.Time = tm
		return nil
	}
	str, ok := value.([]uint8)
	if ok {
		tm, err := Parse("", string(str))
		if err != nil {
			return err // No trace
		}
		tx.Time = tm.Time
		return nil
	}
	tx.Time = time.Time{} // Zero time
	return nil
}

// Value implements the driver Valuer interface used by SQL.
func (tx Timex) Value() (driver.Value, error) {
	if tx.Time.IsZero() {
		return nil, nil
	}
	return tx.Time, nil
}

// JSONSchemaAlias indicates to use the JSON schema of time.
func (tx Timex) JSONSchemaAlias() any {
	return time.Time{}
}

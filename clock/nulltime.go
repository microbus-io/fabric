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

import (
	"database/sql/driver"
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
)

/*
NullTime is a standard time.Time that overrides JSON serialization and database formatting
for the zero time.

To create a new NullTime from time.Time:

	nt := NullTime{time.Now()}
	nt := NewNullTime(time.Now())
	nt := NewNullTimeUTC(time.Now())

To obtain time.Time from a NullTime:

	t := nt.Time
*/
type NullTime struct {
	time.Time `dv8:"main"`
}

// NewNullTime creates a new null time.
func NewNullTime(t time.Time) NullTime {
	return NullTime{t}
}

// NewNullTimeUTC creates a new null time after converting the time to UTC.
func NewNullTimeUTC(t time.Time) NullTime {
	return NullTime{t.UTC()}
}

// ParseNullTime overrides the default parsing of the empty string to return zero time.
// If not provided, layout defaults to RFC3339Nano "2006-01-02T15:04:05.999999999Z07:00",
// RFC3339 "2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05", "2006-01-02 15:04:05",
// or "2006-01-02" based on the length of the value.
func ParseNullTime(layout string, value string) (NullTime, error) {
	if value == "" {
		return NullTime{}, nil
	}
	if layout == "" {
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
	}
	t, err := time.Parse(layout, value)
	return NullTime{t}, err
}

// MustParseNullTime is the same as MustParseNullTime but panics on error.
func MustParseNullTime(layout string, value string) NullTime {
	nt, err := ParseNullTime(layout, value)
	if err != nil {
		panic(err)
	}
	return nt
}

// ParseNullTimeUTC overrides the default parsing of the empty string to return zero time.
// If not provided, layout defaults to RFC3339 "2006-01-02T15:04:05Z07:00".
// Time is converted to UTC.
func ParseNullTimeUTC(layout string, value string) (NullTime, error) {
	nt, err := ParseNullTime(layout, value)
	if err != nil || nt.IsZero() {
		return nt, err
	}
	return NullTime{nt.Time.UTC()}, nil
}

// MustParseNullTimeUTC is the same as ParseNullTimeUTC but panics on error.
func MustParseNullTimeUTC(layout string, value string) NullTime {
	nt, err := ParseNullTimeUTC(layout, value)
	if err != nil {
		panic(err)
	}
	return nt
}

// MarshalJSON overrides JSON serialization of the zero time to null.
func (nt NullTime) MarshalJSON() ([]byte, error) {
	if nt.IsZero() {
		return []byte("null"), nil
	}
	return nt.Time.MarshalJSON()
}

// UnmarshalJSON overrides JSON deserialization,
// interpreting "null" and "" as the zero time.
func (nt *NullTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "" || s == "null" {
		nt.Time = time.Time{}
		return nil
	}
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	tm, err := ParseNullTime("", s)
	if err != nil {
		return errors.Trace(err)
	}
	nt.Time = tm.Time
	return nil
}

// MarshalYAML overrides YAML serialization.
func (nt NullTime) MarshalYAML() (interface{}, error) {
	if nt.IsZero() {
		return nil, nil
	}
	b, err := nt.Time.MarshalJSON()
	if err != nil {
		return nil, errors.Trace(err)
	}
	return string(b[1 : len(b)-1]), nil
}

// UnmarshalYAML overrides YAML deserialization.
func (nt *NullTime) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return errors.Trace(err)
	}
	p, err := ParseNullTime("", s)
	if err != nil {
		return errors.Trace(err)
	}
	*nt = p
	return nil
}

// Scan implements the Scanner interface.
func (nt *NullTime) Scan(value interface{}) error {
	m, ok := value.(time.Time)
	if ok {
		nt.Time = m
	} else {
		nt.Time = time.Time{} // Zero time
	}
	return nil
}

// Value implements the driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if nt.Time.IsZero() {
		return nil, nil
	}
	return nt.Time, nil
}

// Format overrides the default formatting of the zero time to an empty string.
func (nt NullTime) Format(layout string) string {
	if nt.Time.IsZero() {
		return ""
	}
	return nt.Time.Format(layout)
}

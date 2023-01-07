package clock

import (
	"bytes"
	"database/sql/driver"
	"time"
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
	time.Time
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
	if bytes.Equal(b, []byte(`""`)) || bytes.Equal(b, []byte("null")) {
		nt.Time = time.Time{}
		return nil
	}
	return nt.Time.UnmarshalJSON(b)
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

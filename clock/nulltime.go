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

	nt := NullTime{Time: time.Now()}

To obtain the time.Time from a NullTime:

	t := nt.Time
*/
type NullTime struct {
	time.Time
}

// NewNullTime creates a new NullTime.
func NewNullTime(t time.Time) NullTime {
	return NullTime{Time: t}
}

// MarshalJSON overrides JSON serialization of the zero time to null.
func (t NullTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return t.Time.MarshalJSON()
}

// UnmarshalJSON overrides JSON deserialization,
// interpreting "null" and "" as the zero time.
func (t *NullTime) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte(`""`)) || bytes.Equal(b, []byte("null")) {
		t.Time = time.Time{}
		return nil
	}
	return t.Time.UnmarshalJSON(b)
}

// Scan implements the Scanner interface.
func (t *NullTime) Scan(value interface{}) error {
	m, ok := value.(time.Time)
	if ok {
		t.Time = m
	} else {
		t.Time = time.Time{} // Zero time
	}
	return nil
}

// Value implements the driver Valuer interface.
func (t NullTime) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return t.Time, nil
}

// Format overrides the default formatting of the zero time to an empty string.
func (t NullTime) Format(layout string) string {
	if t.Time.IsZero() {
		return ""
	}
	return t.Time.Format(layout)
}

// Parse overrides the default parsing of the empty string to return zero time.
func Parse(layout string, value string) (NullTime, error) {
	if value == "" {
		return NullTime{}, nil
	}
	t, err := time.Parse(layout, value)
	return NullTime{Time: t}, err
}

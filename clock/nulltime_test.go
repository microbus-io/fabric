package clock

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNullTime_NewNullTime(t *testing.T) {
	t.Parallel()

	now := time.Now()
	actual := NewNullTime(now)
	assert.Equal(t, now, actual.Time)

	actual = NewNullTime(time.Time{})
	assert.Equal(t, time.Time{}, actual.Time)
}

func TestNullTime_SerializeNow(t *testing.T) {
	t.Parallel()

	jt1 := NullTime{time.Now()}
	b, err := json.Marshal(jt1)
	assert.NoError(t, err)
	var jt2 NullTime
	err = json.Unmarshal(b, &jt2)
	assert.NoError(t, err)
	assert.True(t, jt1.Equal(jt2.Time))
}

func TestNullTime_SerializeZero(t *testing.T) {
	t.Parallel()

	jt1 := NullTime{time.Time{}}
	b, err := json.Marshal(jt1)
	assert.NoError(t, err)
	assert.Equal(t, []byte("null"), b)
	var jt2 NullTime
	err = json.Unmarshal(b, &jt2)
	assert.NoError(t, err)
	assert.True(t, jt1.Equal(jt2.Time))
}

func TestNullTime_Format(t *testing.T) {
	t.Parallel()

	jt1 := NullTime{time.Time{}}
	s1 := jt1.Format(time.RFC3339)
	assert.Equal(t, "", s1)

	jt2, err := Parse(time.RFC3339, "")
	assert.NoError(t, err)
	assert.True(t, jt1.Equal(jt2.Time))
}

func TestNullTime_UnmarshalJSONEmpty(t *testing.T) {
	t.Parallel()

	var jt1 NullTime
	err := json.Unmarshal([]byte(`""`), &jt1)
	assert.NoError(t, err)
	assert.True(t, jt1.IsZero())

	err = json.Unmarshal([]byte(`null`), &jt1)
	assert.NoError(t, err)
	assert.True(t, jt1.IsZero())

	jt1 = NullTime{Time: time.Now()}
	err = json.Unmarshal([]byte(`""`), &jt1)
	assert.NoError(t, err)
	assert.True(t, jt1.IsZero())
}

func TestNullTime_UnmarshalJSONInvalid(t *testing.T) {
	t.Parallel()

	var jt2 NullTime
	err := json.Unmarshal(nil, &jt2)
	assert.Error(t, err)

	err = json.Unmarshal([]byte("not-a-time"), &jt2)
	assert.Error(t, err)
}

func TestNullTime_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	var jt1 NullTime
	err := json.Unmarshal([]byte(`"2021-08-11T10:00:00Z"`), &jt1)
	assert.NoError(t, err)
	assert.False(t, jt1.IsZero())

	jt2 := NullTime{Time: time.Now()}
	b, err := json.Marshal(jt2)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var jt3 NullTime
	err = json.Unmarshal(b, &jt3)
	assert.NoError(t, err)
	assert.True(t, jt2.Equal(jt3.Time))
}

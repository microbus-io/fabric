/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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

	jt2, err := ParseNullTime(time.RFC3339, "")
	assert.NoError(t, err)
	assert.True(t, jt1.Equal(jt2.Time))

	jt3 := MustParseNullTimeUTC("", "2015-01-14T11:12:13Z")
	s3 := jt3.Format(time.RFC3339)
	assert.Equal(t, "2015-01-14T11:12:13Z", s3)
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

func TestNullTime_ParseNullTime(t *testing.T) {
	t.Parallel()

	assert.Equal(t, time.Date(2015, 1, 2, 0, 0, 0, 0, time.UTC), MustParseNullTime("", "2015-01-02").Time)
	assert.Equal(t, time.Date(2015, 1, 2, 0, 0, 0, 0, time.UTC), MustParseNullTimeUTC("", "2015-01-02").Time)
	assert.Equal(t, time.Date(2015, 1, 2, 11, 12, 13, 0, time.UTC), MustParseNullTime("", "2015-01-02T11:12:13").Time)
	assert.Equal(t, time.Date(2015, 1, 2, 11, 12, 13, 0, time.UTC), MustParseNullTimeUTC("", "2015-01-02T11:12:13").Time)
	assert.Equal(t, time.Date(2015, 1, 2, 11, 12, 13, 0, time.UTC), MustParseNullTime("", "2015-01-02 11:12:13").Time)
	assert.Equal(t, time.Date(2015, 1, 2, 11, 12, 13, 0, time.UTC), MustParseNullTimeUTC("", "2015-01-02 11:12:13").Time)
	assert.Equal(t, time.Date(2015, 1, 2, 11, 12, 13, 0, time.UTC), MustParseNullTime("", "2015-01-02T11:12:13Z").Time)
	assert.Equal(t, time.Date(2015, 1, 2, 11, 12, 13, 500000, time.UTC), MustParseNullTime("", "2015-01-02T11:12:13.000500000Z").Time)
}

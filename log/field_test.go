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

package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestLog_Fields(t *testing.T) {
	t.Parallel()

	f := Int("int", 1)
	assert.Equal(t, zapcore.Int64Type, f.Type)
	assert.Equal(t, "int", f.Key)
	assert.Equal(t, int64(1), f.Integer)

	f = Float("float", 1)
	assert.Equal(t, zapcore.Float64Type, f.Type)
	assert.Equal(t, "float", f.Key)
	assert.NotZero(t, f.Integer)

	f = String("string", "foo")
	assert.Equal(t, zapcore.StringType, f.Type)
	assert.Equal(t, "string", f.Key)
	assert.Equal(t, "foo", f.String)

	f = Bool("bool", true)
	assert.Equal(t, zapcore.BoolType, f.Type)
	assert.Equal(t, "bool", f.Key)
	assert.Equal(t, int64(1), f.Integer)

	f = Duration("duration", time.Minute)
	assert.Equal(t, zapcore.DurationType, f.Type)
	assert.Equal(t, "duration", f.Key)
	assert.NotZero(t, f.Integer)
	f = Time("time", time.Now())
	assert.Equal(t, zapcore.TimeType, f.Type)
	assert.Equal(t, "time", f.Key)
	assert.NotZero(t, f.Integer)
}

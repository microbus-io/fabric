/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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

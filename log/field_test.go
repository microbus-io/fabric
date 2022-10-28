package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestLog_Fields(t *testing.T) {
	f := Uint("uint", 1)
	assert.Equal(t, zapcore.Uint64Type, f.Type)
	assert.Equal(t, "uint", f.Key)
	assert.Equal(t, int64(1), f.Integer)
	f = Uint32("uint32", 1)
	assert.Equal(t, zapcore.Uint32Type, f.Type)
	assert.Equal(t, "uint32", f.Key)
	assert.Equal(t, int64(1), f.Integer)
	f = Uint64("uint64", 1)
	assert.Equal(t, zapcore.Uint64Type, f.Type)
	assert.Equal(t, "uint64", f.Key)
	assert.Equal(t, int64(1), f.Integer)

	f = Int("int", 1)
	assert.Equal(t, zapcore.Int64Type, f.Type)
	assert.Equal(t, "int", f.Key)
	assert.Equal(t, int64(1), f.Integer)
	f = Int32("int32", 1)
	assert.Equal(t, zapcore.Int32Type, f.Type)
	assert.Equal(t, "int32", f.Key)
	assert.Equal(t, int64(1), f.Integer)
	f = Int64("int64", 1)
	assert.Equal(t, zapcore.Int64Type, f.Type)
	assert.Equal(t, "int64", f.Key)
	assert.Equal(t, int64(1), f.Integer)

	f = Float32("float32", 1)
	assert.Equal(t, zapcore.Float32Type, f.Type)
	assert.Equal(t, "float32", f.Key)
	assert.NotZero(t, f.Integer)
	f = Float64("float64", 1)
	assert.Equal(t, zapcore.Float64Type, f.Type)
	assert.Equal(t, "float64", f.Key)
	assert.NotZero(t, f.Integer)

	f = String("string", "foo")
	assert.Equal(t, zapcore.StringType, f.Type)
	assert.Equal(t, "string", f.Key)
	assert.Equal(t, "foo", f.String)
	f = ByteString("bytestring", []byte("foo"))
	assert.Equal(t, zapcore.ByteStringType, f.Type)
	assert.Equal(t, "bytestring", f.Key)
	assert.Equal(t, []byte("foo"), f.Interface)
	x := stringer{}
	f = Stringer("stringer", x)
	assert.Equal(t, zapcore.StringerType, f.Type)
	assert.Equal(t, "stringer", f.Key)
	assert.NotNil(t, f.Interface)

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

	var z struct{}
	f = Any("any", &z)
	assert.Equal(t, zapcore.ReflectType, f.Type)
	assert.Equal(t, "any", f.Key)
	assert.NotNil(t, f.Interface)
}

type stringer struct{}

func (s stringer) String() string { return "foo" }

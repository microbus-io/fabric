package log

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Field is an alias for zapcore.Field which are fields used for logging
type Field = zapcore.Field

// Uint creates a uint log field
func Uint(key string, val uint) Field {
	return zap.Uint(key, val)
}

// Uint32 creates a uint32 log field
func Uint32(key string, val uint32) Field {
	return zap.Uint32(key, val)
}

// Uint64 creates a uint64 log field
func Uint64(key string, val uint64) Field {
	return zap.Uint64(key, val)
}

// Int creates an int log field
func Int(key string, val int) Field {
	return zap.Int(key, val)
}

// Int32 creates an int32 log field
func Int32(key string, val int32) Field {
	return zap.Int32(key, val)
}

// Int64 creates an int64 log field
func Int64(key string, val int64) Field {
	return zap.Int64(key, val)
}

// Float32 creates a float32 log field
func Float32(key string, val float32) Field {
	return zap.Float32(key, val)
}

// Float64 creates a float64 log field
func Float64(key string, val float64) Field {
	return zap.Float64(key, val)
}

// String creates a string log field
func String(key string, val string) Field {
	return zap.String(key, val)
}

// ByteString creates a byte string log field
func ByteString(key string, val []byte) Field {
	return zap.ByteString(key, val)
}

// Stringer creates a stringer log field
func Stringer(key string, val fmt.Stringer) Field {
	return zap.Stringer(key, val)
}

// Bool creates a bool log field
func Bool(key string, val bool) Field {
	return zap.Bool(key, val)
}

// Duration creates a duration log field
func Duration(key string, val time.Duration) Field {
	return zap.Duration(key, val)
}

// Time creates a time log field
func Time(key string, val time.Time) Field {
	return zap.Time(key, val)
}

// Any creates an any log field
func Any(key string, val interface{}) Field {
	return zap.Any(key, val)
}

// Error creates an error log field
func Error(err error) Field {
	return zap.Error(err)
}

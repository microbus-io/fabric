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
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Field is an alias for zapcore.Field which are fields used for logging
type Field = zapcore.Field

// Int creates an int log field
func Int(key string, val int) Field {
	return zap.Int(key, val)
}

// Float creates a float log field
func Float(key string, val float64) Field {
	return zap.Float64(key, val)
}

// String creates a string log field
func String(key string, val string) Field {
	return zap.String(key, val)
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

// Error creates an error log field
func Error(err error) Field {
	return zap.Error(err)
}

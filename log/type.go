package log

import (
	"go.uber.org/zap/zapcore"
)

// Type is the field type used for logging.
type Type zapcore.FieldType

const (
	UnknownType         = Type(zapcore.UnknownType)
	ArrayMarshalerType  = Type(zapcore.ArrayMarshalerType)
	ObjectMarshalerType = Type(zapcore.ObjectMarshalerType)
	BinaryType          = Type(zapcore.BinaryType)
	BoolType            = Type(zapcore.BoolType)
	ByteStringType      = Type(zapcore.ByteStringType)
	Complex128Type      = Type(zapcore.Complex128Type)
	Complex64Type       = Type(zapcore.Complex64Type)
	DurationType        = Type(zapcore.DurationType)
	Float64Type         = Type(zapcore.Float64Type)
	Float32Type         = Type(zapcore.Float32Type)
	Int64Type           = Type(zapcore.Int64Type)
	Int32Type           = Type(zapcore.Int32Type)
	Int16Type           = Type(zapcore.Int16Type)
	Int8Type            = Type(zapcore.Int8Type)
	StringType          = Type(zapcore.StringType)
	TimeType            = Type(zapcore.TimeType)
	Uint64Type          = Type(zapcore.Uint64Type)
	Uint32Type          = Type(zapcore.Uint32Type)
	Uint16Type          = Type(zapcore.Uint16Type)
	Uint8Type           = Type(zapcore.Uint8Type)
	UintptrType         = Type(zapcore.UintptrType)
	ReflectType         = Type(zapcore.ReflectType)
	NamespaceType       = Type(zapcore.NamespaceType)
	StringerType        = Type(zapcore.StringerType)
	ErrorType           = Type(zapcore.ErrorType)
	SkipType            = Type(zapcore.SkipType)
)

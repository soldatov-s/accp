package logger

import (
	// stdlib
	"reflect"
)

// Field represents field information that should be passed to
// context.Logger.GetLogger function.
type Field struct {
	// Name is a field name.
	Name string
	// Type is a field value data type (like reflect.Int64,
	// reflect.String).
	Type  reflect.Kind
	Value interface{}
}

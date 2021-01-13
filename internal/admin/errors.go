package admin

import (
	"errors"
	"path/filepath"
	"reflect"
)

var (
	ErrEmptyDependencyName = errors.New("empty dependency name")
	ErrCheckFuncIsNil      = errors.New("pointer to checkFunc is nil")
	ErrEmptyMetricName     = errors.New("empty metric name")
)

func ErrInvalidProviderOptions(iface interface{}) error {
	return errors.New("passed provider options is not *" +
		ObjName(iface))
}

func ErrInvalidMetricOptions(iface interface{}) error {
	return errors.New("passed metric options is not *" +
		ObjName(iface))
}

// ObjName return object name for passing it to errors messages
func ObjName(iface interface{}) string {
	return filepath.Base(reflect.TypeOf(iface).PkgPath()) +
		"." + reflect.TypeOf(iface).Name()
}

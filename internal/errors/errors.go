package errors

import "errors"

func EmptyConfigParameter(name string) error {
	return errors.New("empty config parameter " + name)
}

func EmptyConfig(name string) error {
	return errors.New("empty config " + name)
}

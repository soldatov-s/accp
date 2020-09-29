package introspector

import (
	"errors"
	"fmt"
)

var (
	ErrBadAuthRequest = errors.New("bad authorization request")
)

type ErrTokenInactive struct {
	token string
}

func (e *ErrTokenInactive) Error() string {
	return fmt.Sprintf("token %s inactive", e.token)
}

package introspection

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
	if len(e.token) > 8 {
		return fmt.Sprintf("token %s inactive", e.token[0:4]+"****"+e.token[len(e.token)-4:])
	}
	return fmt.Sprintf("token %s inactive", e.token)
}

package routes

import "fmt"

type ErrDuplicatedRoute struct {
	route string
}

func (e *ErrDuplicatedRoute) Error() string {
	return fmt.Sprintf("duplicated route: %s", e.route)
}

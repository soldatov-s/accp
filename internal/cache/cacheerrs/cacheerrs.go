package cacheerrs

import "errors"

var (
	ErrNotFoundInCache = errors.New("not found in cache")
)

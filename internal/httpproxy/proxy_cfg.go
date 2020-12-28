package httpproxy

import "github.com/soldatov-s/accp/internal/routes"

type Config struct {
	Listen    string
	Hydration struct {
		RequestID bool
	}
	Routes   map[string]*routes.Config
	Excluded map[string]*routes.Config
}

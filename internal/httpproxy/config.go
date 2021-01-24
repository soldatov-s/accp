package httpproxy

import (
	"github.com/soldatov-s/accp/internal/errors"
	"github.com/soldatov-s/accp/internal/routes"
)

const (
	defaultListen = "0.0.0.0:9000"
)

type Config struct {
	Listen    string
	Hydration struct {
		RequestID bool
	}
	Routes   routes.MapConfig
	Excluded routes.MapConfig
}

func (c *Config) Validate() error {
	if c.Listen == "" {
		c.Listen = defaultListen
	}

	if len(c.Routes) == 0 {
		return errors.EmptyConfigParameter("routes")
	}

	return nil
}

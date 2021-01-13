package redis

import (
	"time"

	"github.com/soldatov-s/accp/internal/errors"
)

const (
	defaultMinIdleConnections   = 10
	defaultMaxOpenedConnections = 30
	defaultMaxConnLifetime      = time.Second * 10
)

type Config struct {
	DSN                   string
	MinIdleConnections    int
	MaxOpenedConnections  int
	MaxConnectionLifetime time.Duration
}

func (c *Config) Validate() error {
	if c.DSN == "" {
		return errors.EmptyConfigParameter("dsn")
	}

	if c.MaxConnectionLifetime == 0 {
		c.MaxConnectionLifetime = defaultMaxConnLifetime
	}

	if c.MinIdleConnections == 0 {
		c.MinIdleConnections = defaultMinIdleConnections
	}

	if c.MaxOpenedConnections == 0 {
		c.MaxOpenedConnections = defaultMaxOpenedConnections
	}

	return nil
}

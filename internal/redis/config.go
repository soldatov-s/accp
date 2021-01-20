package redis

import (
	"time"

	"github.com/go-redis/redis/v8"
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

func (c *Config) SetDefault() {
	if c.MaxConnectionLifetime == 0 {
		c.MaxConnectionLifetime = defaultMaxConnLifetime
	}

	if c.MinIdleConnections == 0 {
		c.MinIdleConnections = defaultMinIdleConnections
	}

	if c.MaxOpenedConnections == 0 {
		c.MaxOpenedConnections = defaultMaxOpenedConnections
	}
}

func (c *Config) Options() (*redis.Options, error) {
	// Connect to database.
	connOptions, err := redis.ParseURL(c.DSN)
	if err != nil {
		return nil, err
	}

	// Set connection pooling options.
	connOptions.MaxConnAge = c.MaxConnectionLifetime
	connOptions.MinIdleConns = c.MinIdleConnections
	connOptions.PoolSize = c.MaxOpenedConnections

	return connOptions, nil
}

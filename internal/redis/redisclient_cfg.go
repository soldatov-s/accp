package externalcache

import "time"

type RedisConfig struct {
	DSN                   string
	MinIdleConnections    int
	MaxOpenedConnections  int
	MaxConnectionLifetime time.Duration
}

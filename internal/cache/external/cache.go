package externalcache

type RedisConfig struct {
	DSN                   string
	KeyPrefix             string
	MinIdleConnections    int
	MaxOpenedConnections  int
	MaxConnectionLifetime string
	ClearTime             string
}

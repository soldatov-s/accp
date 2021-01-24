package refresh

import "time"

const (
	defaultMaxCount = 100
	defaultTime     = 10 * time.Second
)

type Config struct {
	// Conter - the maximum of requests after which will be refreshed cache
	MaxCount int
	// Time - the refresh period
	Time time.Duration
}

func (c *Config) SetDefault() {
	if c.MaxCount == 0 {
		c.MaxCount = defaultMaxCount
	}

	if c.Time == 0 {
		c.Time = defaultTime
	}
}

func (c *Config) Merge(target *Config) *Config {
	if c == nil {
		return target
	}

	result := &Config{
		MaxCount: c.MaxCount,
		Time:     c.Time,
	}

	if target == nil {
		return result
	}

	if target.MaxCount > 0 {
		result.MaxCount = target.MaxCount
	}

	if target.Time > 0 {
		result.Time = target.Time
	}

	return result
}

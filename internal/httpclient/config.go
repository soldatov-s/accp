package httpclient

import "time"

const (
	defaultSize    = 20
	defaultTimeout = 5 * time.Second
)

type Config struct {
	// Size - size of pool httpclients for introspection requests
	Size int
	// Timeout - timeout of httpclients for introspection requests
	Timeout time.Duration
}

func (c *Config) SetDefault() {
	if c.Size == 0 {
		c.Size = defaultSize
	}

	if c.Timeout == 0 {
		c.Timeout = defaultTimeout
	}
}

func (c *Config) Merge(target *Config) *Config {
	if c == nil {
		return target
	}

	result := &Config{
		Size:    c.Size,
		Timeout: c.Timeout,
	}

	if target == nil {
		return result
	}

	if target.Size > 0 {
		result.Size = target.Size
	}

	if target.Timeout > 0 {
		result.Timeout = target.Timeout
	}

	return result
}

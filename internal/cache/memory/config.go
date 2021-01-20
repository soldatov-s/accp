package memory

import "time"

const (
	defaultTTL    = 10 * time.Second
	defaultTTLErr = 5 * time.Second
)

type Config struct {
	TTL    time.Duration
	TTLErr time.Duration
}

func (c *Config) SetDefault() {
	if c.TTL == 0 {
		c.TTL = defaultTTL
	}

	if c.TTLErr == 0 {
		c.TTLErr = defaultTTLErr
	}
}

func (c *Config) Merge(target *Config) *Config {
	if c == nil {
		return target
	}

	result := &Config{
		TTL:    c.TTL,
		TTLErr: c.TTLErr,
	}

	if target == nil {
		return result
	}

	if target.TTL > 0 {
		result.TTL = target.TTL
	}

	if target.TTLErr > 0 {
		result.TTLErr = target.TTLErr
	}

	return result
}

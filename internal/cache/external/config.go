package external

import "time"

const (
	defaultKeyPrefix = "accp_"
	defaultTTL       = 10 * time.Second
	defaultTTLErr    = 5 * time.Second
)

type Config struct {
	KeyPrefix string
	TTL       time.Duration
	TTLErr    time.Duration
}

func (c *Config) SetDefault() {
	if c.KeyPrefix == "" {
		c.KeyPrefix = defaultKeyPrefix
	}

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
		KeyPrefix: c.KeyPrefix,
		TTL:       c.TTL,
		TTLErr:    c.TTLErr,
	}

	if target == nil {
		return result
	}

	if target.KeyPrefix != "" {
		result.KeyPrefix = target.KeyPrefix
	}

	if target.TTL > 0 {
		result.TTL = target.TTL
	}

	if target.TTLErr > 0 {
		result.TTLErr = target.TTLErr
	}

	return result
}

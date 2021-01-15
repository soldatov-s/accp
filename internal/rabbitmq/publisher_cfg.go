package rabbitmq

import (
	"time"

	"github.com/soldatov-s/accp/internal/errors"
)

type Config struct {
	DSN           string
	BackoffPolicy []time.Duration
	ExchangeName  string
}

func (c *Config) Validate() error {
	if c.DSN == "" {
		return errors.EmptyConfigParameter("dsn")
	}

	if len(c.BackoffPolicy) == 0 {
		c.BackoffPolicy = []time.Duration{
			2 * time.Second,
			5 * time.Second,
			10 * time.Second,
			15 * time.Second,
			20 * time.Second,
			25 * time.Second,
		}
	}

	return nil
}

func (c *Config) Merge(target *Config) *Config {
	result := &Config{
		DSN:           c.DSN,
		BackoffPolicy: c.BackoffPolicy,
		ExchangeName:  c.ExchangeName,
	}

	if target.DSN != "" {
		result.DSN = target.DSN
	}

	if len(target.BackoffPolicy) > 0 {
		result.BackoffPolicy = target.BackoffPolicy
	}

	if target.ExchangeName != "" {
		result.ExchangeName = target.ExchangeName
	}

	return result
}

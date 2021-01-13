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

func (pc *Config) Validate() error {
	if pc.Size == 0 {
		pc.Size = defaultSize
	}

	if pc.Timeout == 0 {
		pc.Timeout = defaultTimeout
	}

	return nil
}

func (pc *Config) Merge(target *Config) *Config {
	result := &Config{
		Size:    pc.Size,
		Timeout: pc.Timeout,
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

package refresh

import "time"

const (
	defaultCount = 100
	defaultTime  = 10 * time.Second
)

type Config struct {
	// Conter
	Count int
	// Time
	Time time.Duration
}

func (rc *Config) Initilize() error {
	if rc.Count == 0 {
		rc.Count = defaultCount
	}

	if rc.Time == 0 {
		rc.Time = defaultTime
	}

	return nil
}

func (rc *Config) Merge(target *Config) *Config {
	result := &Config{
		Count: rc.Count,
		Time:  rc.Time,
	}

	if target == nil {
		return result
	}

	if target.Count > 0 {
		result.Count = target.Count
	}

	if target.Time > 0 {
		result.Time = target.Time
	}

	return result
}

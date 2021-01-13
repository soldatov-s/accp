package external

import "time"

type Config struct {
	KeyPrefix string
	TTL       time.Duration
	TTLErr    time.Duration
}

func (cc *Config) Initilize() error {
	if cc.KeyPrefix == "" {
		cc.KeyPrefix = defaultKeyPrefix
	}

	if cc.TTL == 0 {
		cc.TTL = defaultTTL
	}

	return nil
}

func (cc *Config) Merge(target *Config) *Config {
	if cc == nil {
		return target
	}

	result := &Config{
		KeyPrefix: cc.KeyPrefix,
		TTL:       cc.TTL,
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

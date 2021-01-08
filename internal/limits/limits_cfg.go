package limits

import "time"

type Config struct {
	// Header is name of header in request for limit
	Header []string
	// Cookie is name of cookie in request for limit
	Cookie []string
	// Limit Count per Time period
	// Conter limits count of request to API
	Counter int
	// PT limits period of requests to API
	PT time.Duration
}

type MapConfig map[string]*Config

func NewMapConfig() MapConfig {
	l := make(MapConfig)
	l["token"] = &Config{
		Header: []string{"Authorization"},
	}
	l["ip"] = &Config{
		Header: []string{"X-Forwarded-For"},
	}

	return l
}

func (mc MapConfig) Merge(target MapConfig) MapConfig {
	result := make(MapConfig)
	for k, v := range mc {
		result[k] = v
	}

	for k, v := range target {
		if limit, ok := result[k]; !ok {
			result[k] = v
		} else {
			result[k] = limit.Merge(v)
		}
	}

	return result
}

func (lc *Config) Merge(target *Config) *Config {
	result := &Config{
		Header:  lc.Header,
		Cookie:  lc.Cookie,
		Counter: lc.Counter,
		PT:      lc.PT,
	}

	if target == nil {
		return result
	}

	if len(target.Header) > 0 {
		result.Header = append(result.Header, target.Header...)
	}

	if len(target.Cookie) > 0 {
		result.Cookie = append(result.Cookie, target.Cookie...)
	}

	if target.Counter > 0 {
		result.Counter = target.Counter
	}

	if target.PT > 0 {
		result.PT = target.PT
	}

	return result
}

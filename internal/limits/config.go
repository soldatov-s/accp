package limits

import (
	"time"

	"github.com/soldatov-s/accp/x/helper"
)

const (
	defaultCounter      = 1000
	defaultPT           = time.Minute
	authorizationHeader = "authorization"
	ipHeader            = "x-forwarded-for"
	// default name for default items in mapconfig
	defaultItemIP    = "ip"
	defaultItemToken = "token"
)

type Config struct {
	// Header is name of header in request for limit
	Header helper.Arguments
	// Cookie is name of cookie in request for limit
	Cookie helper.Arguments
	// Limit Count per Time period
	// Conter limits count of request to API
	Counter int
	// PT limits period of requests to API
	PT time.Duration
}

func (c *Config) SetDefault() {
	if c.Counter == 0 {
		c.Counter = defaultCounter
	}

	if c.PT == 0 {
		c.PT = defaultPT
	}
}

func (c *Config) Merge(target *Config) *Config {
	if c == nil {
		return target
	}

	result := &Config{
		Header:  c.Header,
		Cookie:  c.Cookie,
		Counter: c.Counter,
		PT:      c.PT,
	}

	if target == nil {
		return result
	}

	if len(target.Header) > 0 {
		for _, v := range target.Header {
			if result.Header.Matches(v) {
				continue
			}
			result.Header = append(result.Header, v)
		}
	}

	if len(target.Cookie) > 0 {
		for _, v := range target.Cookie {
			if result.Cookie.Matches(v) {
				continue
			}
			result.Cookie = append(result.Cookie, v)
		}
	}

	if target.Counter > 0 {
		result.Counter = target.Counter
	}

	if target.PT > 0 {
		result.PT = target.PT
	}

	return result
}

type MapConfig map[string]*Config

// NewMapConfig creates MapConfig with predefined items "token" and "ip"
func NewMapConfig() MapConfig {
	l := make(MapConfig)
	l[defaultItemToken] = &Config{
		Header: []string{authorizationHeader},
	}
	l[defaultItemIP] = &Config{
		Header: []string{ipHeader},
	}

	return l
}

// Merge merges MapConfig
func (c MapConfig) Merge(target MapConfig) MapConfig {
	if c == nil {
		return target
	}

	result := make(MapConfig)
	for k, v := range c {
		result[k] = v
	}

	if target == nil {
		return result
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

package limits

import (
	"time"

	"github.com/soldatov-s/accp/x/helper"
)

const (
	defaultCounter      = 1000
	defaultTTL          = time.Minute
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
	// MaxCounter limits count of request to API
	MaxCounter int
	// TTL limits period of requests to API
	TTL time.Duration
}

func (c *Config) SetDefault() {
	if c.MaxCounter == 0 {
		c.MaxCounter = defaultCounter
	}

	if c.TTL == 0 {
		c.TTL = defaultTTL
	}
}

func (c *Config) Merge(target *Config) *Config {
	if c == nil {
		return target
	}

	result := &Config{
		Header:     c.Header,
		Cookie:     c.Cookie,
		MaxCounter: c.MaxCounter,
		TTL:        c.TTL,
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

	if target.MaxCounter > 0 {
		result.MaxCounter = target.MaxCounter
	}

	if target.TTL > 0 {
		result.TTL = target.TTL
	}

	return result
}

type MapConfig map[string]*Config

// NewMapConfig creates MapConfig
func NewMapConfig() MapConfig {
	l := make(MapConfig)
	return l
}

// SetDefault sets default items "token" and "ip"
func (c MapConfig) SetDefault() {
	c[defaultItemToken] = &Config{
		Header: []string{authorizationHeader},
	}
	c[defaultItemIP] = &Config{
		Header: []string{ipHeader},
	}
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

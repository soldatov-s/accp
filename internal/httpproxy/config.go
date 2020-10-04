package httpproxy

import (
	"time"

	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cache/memory"
	"github.com/soldatov-s/accp/internal/httpclient"
)

type CacheConfig struct {
	Memory   *memory.CacheConfig
	External *external.CacheConfig
}

func (cc *CacheConfig) Merge(target *CacheConfig) *CacheConfig {
	result := &CacheConfig{
		Memory:   cc.Memory,
		External: cc.External,
	}

	if target == nil {
		return result
	}

	if target.Memory != nil {
		result.Memory = cc.Memory.Merge(target.Memory)
	}

	if target.External != nil {
		result.External = cc.External.Merge(target.External)
	}

	return result
}

type RefreshConfig struct {
	// Conter
	Count int
	// Time
	Time time.Duration
}

func (rc *RefreshConfig) Merge(target *RefreshConfig) *RefreshConfig {
	result := &RefreshConfig{
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

type LimitConfig struct {
	// Header is name of header in request for limit
	Header string
	// Cookie is name of cookie in request for limit
	Cookie string
	// Limit Count per Time period
	// Conter limits count of request to API
	Counter int
	// Time limits period of requests to API
	Time time.Duration
}

func (lc *LimitConfig) Merge(target *LimitConfig) *LimitConfig {
	result := &LimitConfig{
		Header:  lc.Header,
		Cookie:  lc.Cookie,
		Counter: lc.Counter,
		Time:    lc.Time,
	}

	if target == nil {
		return result
	}

	if target.Header != "" {
		result.Header = target.Header
	}

	if target.Cookie != "" {
		result.Cookie = target.Cookie
	}

	if target.Counter > 0 {
		result.Counter = target.Counter
	}

	if target.Time > 0 {
		result.Time = target.Time
	}

	return result
}

type RouteParameters struct {
	DSN             string
	TTL             time.Duration
	Limits          map[string]*LimitConfig
	Refresh         *RefreshConfig
	Cache           *CacheConfig
	Pool            *httpclient.PoolConfig
	PublishKeyRoute string
	// Introspect if true it means that necessary to introspect request
	Introspect bool
}

func (rp *RouteParameters) Merge(target *RouteParameters) *RouteParameters {
	result := &RouteParameters{
		DSN:             rp.DSN,
		TTL:             rp.TTL,
		Cache:           rp.Cache,
		Refresh:         rp.Refresh,
		Pool:            rp.Pool,
		Limits:          rp.Limits,
		PublishKeyRoute: rp.PublishKeyRoute,
	}

	if target == nil {
		return result
	}

	if target.DSN != "" {
		result.DSN = target.DSN
	}

	if target.TTL > 0 {
		result.TTL = target.TTL
	}

	if target.Cache != nil {
		result.Cache = rp.Cache.Merge(target.Cache)
	}

	if target.Refresh != nil {
		result.Refresh = rp.Refresh.Merge(target.Refresh)
	}

	if target.Pool != nil {
		result.Pool = rp.Pool.Merge(target.Pool)
	}

	if target.Limits != nil {
		result.Limits = make(map[string]*LimitConfig)
		for k, v := range rp.Limits {
			result.Limits[k] = v
		}

		for k, v := range target.Limits {
			if limit, ok := result.Limits[k]; !ok {
				result.Limits[k] = v
			} else {
				result.Limits[k] = limit.Merge(v)
			}
		}
	}

	if target.PublishKeyRoute != "" {
		result.PublishKeyRoute = target.PublishKeyRoute
	}

	return result
}

type RouteConfig struct {
	Parameters *RouteParameters
	Routes     map[string]*RouteConfig
}

type HTTPProxyConfig struct {
	Listen    string
	Hydration struct {
		RequestID  bool
		Introspect string
	}
	Routes   map[string]*RouteConfig
	Excluded map[string]*RouteConfig
}

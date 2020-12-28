package routes

import (
	"net/http"
	"time"

	"github.com/soldatov-s/accp/internal/cache"
	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/limits"
	"github.com/soldatov-s/accp/internal/routes/refresh"
	"github.com/soldatov-s/accp/x/helper"
)

type Parameters struct {
	DSN      string
	TTL      time.Duration
	Limits   map[string]*limits.LimitConfig
	Refresh  *refresh.Config
	Cache    *cache.Config
	Pool     *httpclient.PoolConfig
	Methods  helper.Arguments
	RouteKey string
	// Introspect if true it means that necessary to introspect request
	Introspect          bool
	IntrospectHydration string
}

func (rp *Parameters) Initilize() error {
	if len(rp.Methods) == 0 {
		rp.Methods = helper.Arguments{http.MethodGet, http.MethodPut, http.MethodDelete}
	}

	if rp.Cache == nil {
		rp.Cache = &cache.Config{}
	}

	if err := rp.Cache.Initilize(); err != nil {
		return err
	}

	if rp.Refresh == nil {
		rp.Refresh = &refresh.Config{}
	}

	if err := rp.Refresh.Initilize(); err != nil {
		return err
	}

	if rp.Pool == nil {
		rp.Pool = &httpclient.PoolConfig{}
	}

	if err := rp.Pool.Initilize(); err != nil {
		return err
	}

	if rp.Limits == nil {
		rp.Limits = make(map[string]*limits.LimitConfig)
		rp.Limits["token"] = &limits.LimitConfig{
			Header: []string{"Authorization"},
		}
		rp.Limits["ip"] = &limits.LimitConfig{
			Header: []string{"X-Forwarded-For"},
		}
	}

	return nil
}

func (rp *Parameters) Merge(target *Parameters) *Parameters {
	result := &Parameters{
		DSN:        rp.DSN,
		TTL:        rp.TTL,
		Cache:      rp.Cache,
		Refresh:    rp.Refresh,
		Pool:       rp.Pool,
		Limits:     rp.Limits,
		RouteKey:   rp.RouteKey,
		Introspect: rp.Introspect,
		Methods:    rp.Methods,
	}

	if target == nil {
		return result
	}

	if len(target.Methods) > 0 {
		result.Methods = target.Methods
	}

	if target.Introspect {
		result.Introspect = true
	}

	if target.IntrospectHydration != "" {
		result.IntrospectHydration = target.IntrospectHydration
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
		result.Limits = make(map[string]*limits.LimitConfig)
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

	if target.RouteKey != "" {
		result.RouteKey = target.RouteKey
	}

	return result
}

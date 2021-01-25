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

func defaultMethods() helper.Arguments {
	return helper.Arguments{http.MethodGet}
}

type Parameters struct {
	DSN      string
	TTL      time.Duration
	Limits   limits.MapConfig
	Refresh  *refresh.Config
	Cache    *cache.Config
	Pool     *httpclient.Config
	Methods  helper.Arguments
	RouteKey string
	// NotIntrospect if true it means that not necessary to introspect request
	NotIntrospect bool
	// IntrospectHydration describes hydrations format
	IntrospectHydration string
}

func (p *Parameters) SetDefault() {
	if len(p.Methods) == 0 {
		p.Methods = defaultMethods()
	}

	if p.Cache == nil {
		p.Cache = &cache.Config{}
	}

	p.Cache.SetDefault()

	if p.Refresh == nil {
		p.Refresh = &refresh.Config{}
	}

	p.Refresh.SetDefault()

	if p.Pool == nil {
		p.Pool = &httpclient.Config{}
	}

	p.Pool.SetDefault()

	if p.Limits == nil {
		p.Limits = limits.NewMapConfig()
	}
}

func (p *Parameters) Merge(target *Parameters) *Parameters {
	if p == nil {
		return target
	}

	result := &Parameters{
		DSN:                 p.DSN,
		TTL:                 p.TTL,
		Cache:               p.Cache,
		Refresh:             p.Refresh,
		Pool:                p.Pool,
		Limits:              p.Limits,
		RouteKey:            p.RouteKey,
		NotIntrospect:       p.NotIntrospect,
		IntrospectHydration: p.IntrospectHydration,
		Methods:             p.Methods,
	}

	if target == nil {
		return result
	}

	if len(target.Methods) > 0 {
		for _, v := range target.Methods {
			if result.Methods.Matches(v) {
				continue
			}
			result.Methods = append(result.Methods, v)
		}
	}

	if target.NotIntrospect {
		result.NotIntrospect = true
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
		result.Cache = p.Cache.Merge(target.Cache)
	}

	if target.Refresh != nil {
		result.Refresh = p.Refresh.Merge(target.Refresh)
	}

	if target.Pool != nil {
		result.Pool = p.Pool.Merge(target.Pool)
	}

	if target.Limits != nil {
		result.Limits = p.Limits.Merge(target.Limits)
	}

	if target.RouteKey != "" {
		result.RouteKey = target.RouteKey
	}

	return result
}

package httpproxy

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache"
	"github.com/soldatov-s/accp/internal/cache/cachedata"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cache/memory"
	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/publisher"
	accpmodels "github.com/soldatov-s/accp/models"
)

const (
	defaultCount = 100
	defaultTime  = 10 * time.Second
)

type RefreshConfig struct {
	// Conter
	Count int
	// Time
	Time time.Duration
}

func (rc *RefreshConfig) Initilize() error {
	if rc.Count == 0 {
		rc.Count = defaultCount
	}

	if rc.Time == 0 {
		rc.Time = defaultTime
	}

	return nil
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

type RouteParameters struct {
	DSN             string
	TTL             time.Duration
	Limits          map[string]*LimitConfig
	Refresh         *RefreshConfig
	Cache           *cache.Config
	Pool            *httpclient.PoolConfig
	PublishKeyRoute string
	// Introspect if true it means that necessary to introspect request
	Introspect bool
}

func (rp *RouteParameters) Initilize() error {
	if rp.Cache == nil {
		rp.Cache = &cache.Config{}
	}

	if err := rp.Cache.Initilize(); err != nil {
		return err
	}

	if rp.Refresh == nil {
		rp.Refresh = &RefreshConfig{}
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
		rp.Limits = make(map[string]*LimitConfig)
	}

	return nil
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
		Introspect:      rp.Introspect,
	}

	if target == nil {
		return result
	}

	if target.Introspect {
		result.Introspect = true
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

type Route struct {
	ctx            *context.Context
	log            zerolog.Logger
	parameters     *RouteParameters
	Routes         map[string]*Route
	Cache          *cache.Cache
	Pool           *httpclient.Pool
	WaitAnswerList map[string]chan struct{}
	WaiteAnswerMu  map[string]*sync.Mutex
	Introspect     bool
	Publisher      publisher.Publisher
	RefreshTimer   *time.Timer
	Limits         map[string]LimitTable
	Route          string
}

func (r *Route) Initilize(
	ctx *context.Context,
	route string,
	parameters *RouteParameters,
	externalStorage external.Storage,
	pub publisher.Publisher,
) error {
	var err error
	r.ctx = ctx
	r.log = ctx.GetPackageLogger(empty{})
	r.parameters = parameters

	r.Cache = &cache.Cache{}

	r.Cache.Mem, err = memory.NewCache(ctx, parameters.Cache.Memory)
	if err != nil {
		return err
	}

	r.Cache.External, err = external.NewCache(ctx, parameters.Cache.External, externalStorage)
	if err != nil {
		return err
	}

	r.Publisher = pub

	r.Pool = httpclient.NewPool(parameters.Pool.Size, parameters.Pool.Timeout)

	r.WaitAnswerList = make(map[string]chan struct{})
	r.WaiteAnswerMu = make(map[string]*sync.Mutex)

	r.Introspect = parameters.Introspect

	if parameters.Refresh.Time > 0 {
		r.RefreshTimer = time.AfterFunc(parameters.Refresh.Time, r.RefreshHandler)
	}

	// Load limits
	r.Limits = make(map[string]LimitTable)
	for k := range r.parameters.Limits {
		r.Limits[k] = make(LimitTable)
	}

	r.Route = strings.ReplaceAll(route, "//", "/")

	return nil
}

func (r *Route) GetLimitsFromRequest(req *http.Request) map[string]interface{} {
	limitList := make(map[string]interface{})

	for k, v := range r.parameters.Limits {
		for _, vv := range v.Header {
			if h := req.Header.Get(vv); h != "" {
				limitList[k] = h
			}
		}

		for _, vv := range v.Cookie {
			if c, err := req.Cookie(vv); err == nil {
				limitList[k] = c.Value
			}
		}
	}

	return limitList
}

func (r *Route) CheckLimits(req *http.Request) (*bool, error) {
	result := true

	if len(r.parameters.Limits) == 0 {
		return &result, nil
	}

	limitList := r.GetLimitsFromRequest(req)
	if len(limitList) == 0 {
		return &result, nil
	}

	for k, v := range limitList {
		if vv, ok := (r.Limits[k])[v]; !ok {
			r.Limits[k][v] = &Limit{
				Counter:    1,
				LastAccess: time.Now().Unix(),
			}
			if err := vv.CreateLimit(r.Route, k, r.Cache.External); err != nil {
				return nil, err
			}
		} else {
			vv.Mu.Lock()
			defer vv.Mu.Unlock()

			if err := vv.LoadLimit(r.Route, k, r.Cache.External); err != nil {
				r.log.Err(err).Msgf("failed to get limit %s from external cache", k)
			}

			vv.Counter++

			if vv.Counter >= r.parameters.Limits[k].Counter &&
				time.Now().Add(-r.parameters.Limits[k].PT).Unix() < vv.LastAccess {
				result = result && false
				r.log.Debug().Msgf("limit reached: %s", k)
			} else if time.Now().Add(-r.parameters.Limits[k].PT).Unix() >= vv.LastAccess {
				go func() {
					vv.Counter = 1
					vv.LastAccess = time.Now().Unix()
					if err := vv.UpdateLimit(r.Route, k, r.Cache.External); err != nil {
						r.log.Err(err).Msgf("failed to update limit %s in external cache", k)
					}
				}()
			}
		}
	}

	return &result, nil
}

func (r *Route) RefreshHandler() {
	r.Cache.Mem.Range(func(k, v interface{}) bool {
		data := v.(*cachedata.CacheItem).Data.(*accpmodels.RRData)

		go func() {
			client := r.Pool.GetFromPool()
			if err := data.Update(client); err != nil {
				r.log.Err(err).Msg("failed to update inmemory cache")
			}

			if err := r.Cache.External.Update(k.(string), data); err != nil {
				r.log.Err(err).Msg("failed to update external cache")
			}
		}()

		return true
	})

	r.RefreshTimer.Reset(r.parameters.Refresh.Time)
}

// Publish publishes request from client to message queue
func (r *Route) Publish(message interface{}) error {
	return r.Publisher.SendMessage(message, r.parameters.PublishKeyRoute)
}

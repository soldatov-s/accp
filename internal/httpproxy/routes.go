package httpproxy

import (
	"encoding/json"
	"net/http"
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

type LimitMarshal struct {
	Counter    int
	LastAccess int64 // Unix time
}

type Limit struct {
	Mu         sync.Mutex
	Counter    int
	LastAccess int64 // Unix time
}

type LimitTable map[interface{}]*Limit

func (l *Limit) LoadLimit(name, key string, externalStorage *external.Cache) error {
	if externalStorage != nil {
		if err := externalStorage.JSONGet(key, name+".counter", &l.Counter); err != nil {
			return err
		}
		if err := externalStorage.JSONGet(key, name+".lastaccess", &l.LastAccess); err != nil {
			return err
		}
	}

	return nil
}

func (l *Limit) UpdateLimit(route, key string, externalStorage *external.Cache) error {
	if externalStorage != nil {
		data, err := json.Marshal(&l.Counter)
		if err != nil {
			return err
		}

		if err := externalStorage.JSONSet(route, key+".counter", string(data)); err != nil {
			return err
		}

		data, err = json.Marshal(&l.Counter)
		if err != nil {
			return err
		}

		if err := externalStorage.JSONSet(route, key+".lastaccess", string(data)); err != nil {
			return err
		}

	}

	return nil
}

func (l *Limit) CreateLimit(route, key string, externalStorage *external.Cache) error {
	if externalStorage != nil {
		data, err := json.Marshal(&l.Counter)
		if err != nil {
			return err
		}

		if err := externalStorage.JSONSetNX(route, key+".counter", string(data)); err != nil {
			return err
		}

		data, err = json.Marshal(&l.Counter)
		if err != nil {
			return err
		}

		if err := externalStorage.JSONSetNX(route, key+".lastaccess", string(data)); err != nil {
			return err
		}

	}

	return nil
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

func (r *Route) Initilize(ctx *context.Context, route string, parameters *RouteParameters, externalStorage external.ExternalStorage, publisher publisher.Publisher) error {
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

	r.Publisher = publisher

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

	r.Route = route

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

func (r *Route) Publish(message interface{}) error {
	return r.Publisher.SendMessage(message, r.parameters.PublishKeyRoute)
}

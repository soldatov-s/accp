package httpproxy

import (
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
}

func (r *Route) Initilize(ctx *context.Context, parameters *RouteParameters, externalStorage external.ExternalStorage, publisher publisher.Publisher) error {
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

	return nil
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

package routes

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache"
	"github.com/soldatov-s/accp/internal/cache/cachedata"
	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/httputils"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/limits"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/publisher"
	"github.com/soldatov-s/accp/internal/rabbitmq"
	rrdata "github.com/soldatov-s/accp/internal/request_response_data"
)

type empty struct{}

type Route struct {
	ctx            context.Context
	log            zerolog.Logger
	Parameters     *Parameters
	Routes         MapRoutes
	Cache          *cache.Cache
	Pool           *httpclient.Pool
	waitAnswerList map[string]chan struct{}
	waiteAnswerMu  map[string]*sync.Mutex
	publisher      publisher.Publisher
	RefreshTimer   *time.Timer
	Limits         map[string]*limits.LimitTable
	route          string
	introspector   introspection.Introspector
	excluded       bool
}

func NewRoute(ctx context.Context, routeName string, params *Parameters) *Route {
	return &Route{
		ctx:            ctx,
		log:            logger.GetPackageLogger(ctx, empty{}),
		Parameters:     params,
		route:          routeName,
		Routes:         make(MapRoutes),
		publisher:      rabbitmq.Get(ctx),
		introspector:   introspection.Get(ctx),
		waitAnswerList: make(map[string]chan struct{}),
		waiteAnswerMu:  make(map[string]*sync.Mutex),
	}
}

func (r *Route) Initilize() {
	r.Pool = httpclient.NewPool(r.Parameters.Pool)

	if r.excluded {
		return
	}

	r.Cache = cache.NewCache(r.ctx, r.Parameters.Cache)
	r.Limits = limits.NewLimits(r.route, r.Parameters.Limits, r.Cache.External)

	if r.Parameters.Refresh.Time > 0 {
		r.RefreshTimer = time.AfterFunc(r.Parameters.Refresh.Time, r.refreshByTime)
	}
}

func (r *Route) IsExcluded() bool {
	return r.excluded
}

// checkLimit checks limit
func (r *Route) checkLimit(limitName, limitValue string, result *bool) error {
	var (
		vv *limits.Limit
		ok bool
	)
	if vv, ok = (r.Limits[limitName]).List[limitValue]; !ok {
		if err := (r.Limits[limitName]).Inc(limitValue); err != nil {
			return err
		}
		return nil
	}

	if err := (r.Limits[limitName]).Inc(limitValue); err != nil {
		return err
	}

	if vv.Counter >= r.Parameters.Limits[limitName].Counter {
		*result = *result && false
		r.log.Debug().Msgf("limit reached: %s", limitName)
	}

	return nil
}

func (r *Route) CheckLimits(req *http.Request) (*bool, error) {
	result := true

	if len(r.Parameters.Limits) == 0 {
		return &result, nil
	}

	limitList := limits.NewLimitedParamsOfRequest(r.Parameters.Limits, req)
	if len(limitList) == 0 {
		return &result, nil
	}

	for k, v := range limitList {
		if err := r.checkLimit(k, v, &result); err != nil {
			return nil, err
		} else if !result {
			break
		}
	}

	return &result, nil
}

func (r *Route) refreshHandler(hk string, data *rrdata.RequestResponseData) error {
	data.Mu.Lock()
	defer data.Mu.RUnlock()

	req, err := data.Request.BuildRequest()
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}

	if err = r.HydrationIntrospect(req); err != nil {
		r.log.Err(err).Msg("")
		if err = r.Cache.Delete(hk); err != nil {
			return errors.Wrap(err, "introspection failed, delete key failed")
		}
		return errors.Wrap(err, "introspection failed")
	}

	client := r.Pool.GetFromPool()
	defer r.Pool.PutToPool(client)

	if err := data.UpdateByRequest(client, req); err != nil {
		if err = r.Cache.Delete(hk); err != nil {
			return errors.Wrap(err, "failed to update request/response data, delete key failed")
		}
		return errors.Wrap(err, "failed to update request/response data")
	}

	if r.Cache.External == nil {
		return nil
	}

	if err := r.Cache.External.Update(hk, data); err != nil {
		return errors.Wrap(err, "failed to update external cache")
	}

	r.log.Debug().Msgf("%s: cache refreshed", hk)
	return nil
}

func (r *Route) refreshByTime() {
	r.Cache.Mem.Range(func(k, v interface{}) bool {
		data := v.(*cachedata.CacheItem).Data.(*rrdata.RequestResponseData)
		hk := k.(string)
		go func() {
			if err := r.refreshHandler(hk, data); err != nil {
				r.log.Error().Err(err).Msgf("%s: refresh cache failed", hk)
			}
		}()

		return true
	})

	r.RefreshTimer.Reset(r.Parameters.Refresh.Time)
}

// Publish publishes request from client to message queue
func (r *Route) Publish(message interface{}) error {
	if r.publisher == nil {
		return nil
	}
	return r.publisher.SendMessage(message, r.Parameters.RouteKey)
}

func (r *Route) requestToBack(hk string, w http.ResponseWriter, req *http.Request) *rrdata.RequestResponseData {
	var err error
	// Proxy request to backend
	client := r.Pool.GetFromPool()
	defer r.Pool.PutToPool(client)

	var resp *http.Response
	req.URL, err = url.Parse(r.Parameters.DSN + req.URL.String())
	if err != nil {
		resp = httputils.ErrResponse(err.Error(), http.StatusServiceUnavailable)
	} else {
		// nolint
		resp, err = client.Do(req)
		if err != nil {
			resp = httputils.ErrResponse(err.Error(), http.StatusServiceUnavailable)
		}
	}
	defer resp.Body.Close()

	rrData := rrdata.NewRequestResponseData(hk, r.Parameters.Refresh.Count, r.Cache.External)
	if err := rrData.ReadAll(req, resp); err != nil {
		r.log.Err(err).Msg("failed to read request/response data")
	}

	if err := rrData.Response.Write(w); err != nil {
		r.log.Err(err).Msg("failed to write data to client from response")
	}

	return rrData
}

// NotCached is handler for proxy requests to excluded routes and routes which not need to cache
func (r *Route) NotCached(w http.ResponseWriter, req *http.Request) {
	client := r.Pool.GetFromPool()
	defer r.Pool.PutToPool(client)

	var err error
	req.URL, err = url.Parse(r.Parameters.DSN + req.URL.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	r.log.Debug().Msg(req.URL.String())

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	if err := httputils.CopyHTTPResponse(w, resp); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
}

func (r *Route) HydrationIntrospect(req *http.Request) error {
	if r.introspector == nil || !r.Parameters.Introspect {
		r.log.Debug().Msgf("no introspector or disabled introspection: %s", r.route)
		return nil
	}

	content, err := r.introspector.IntrospectRequest(req)
	if err != nil {
		return err
	}

	var str string
	switch r.Parameters.IntrospectHydration {
	case "nothing":
		return nil
	case "plaintext":
		str = strings.ReplaceAll(strings.ReplaceAll(string(content), "\"", "\\\""), "\n", "")
	case "base64":
		str = base64.StdEncoding.EncodeToString(content)
	}

	req.Header.Add("accp-introspect-body", str)
	r.log.Debug().Msgf("accp-introspect-body header: %s", str)

	return nil
}

// refresh incremets refresh count and checks that we not reached the limit
func (r *Route) refresh(data *rrdata.RequestResponseData, hk string) {
	// Check that we have refresh limit by request count
	if r.Parameters.Refresh.Count == 0 {
		return
	}

	r.log.Debug().Msgf("refresh cache, key %s, maxCount %d, current count %d", hk, r.Parameters.Refresh.Count, data.Response.Refresh.Current())

	if incResult, err := data.Response.Refresh.Inc(); err != nil {
		r.log.Error().Err(err).Msg("failed to check refresh counter")
		return
	} else if *incResult < r.Parameters.Refresh.Count {
		return
	}

	if err := r.refreshHandler(hk, data); err != nil {
		r.log.Error().Err(err).Msgf("%s: refresh cache failed", hk)
	}
}

func (r *Route) responseHandle(data *rrdata.RequestResponseData, w http.ResponseWriter, req *http.Request, hk string) {
	// If data was get from redis, the request will be empty
	if data.Request == nil {
		var err error
		if data.Request, err = rrdata.NewRequest(req); err != nil {
			r.log.Err(err).Msg("failed to read data from request")
		}
	}

	if err := data.Response.Write(w); err != nil {
		r.log.Err(err).Msg("failed to write data from cache")
	}

	if err := r.Publish(data.Request); err != nil {
		r.log.Err(err).Msg("failed to publish data")
	}

	go r.refresh(data, hk)
}

func (r *Route) waitAnswer(w http.ResponseWriter, req *http.Request, hk string, ch chan struct{}) {
	<-ch

	if data, err := r.Cache.Select(hk); err == nil {
		r.responseHandle(data, w, req, hk)
		return
	}

	http.Error(w, "failed to get data from cache", http.StatusServiceUnavailable)
}

func (r *Route) CachedHandler(w http.ResponseWriter, req *http.Request) {
	hk, err := httputils.HashRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Finding a response to a request in the memory cache
	if data, err1 := r.Cache.Select(hk); err1 == nil {
		r.responseHandle(data, w, req, hk)
		return
	}

	// Check that we not started to handle the request
	if waitCh, ok := r.waitAnswerList[hk]; !ok {
		// If we not started to handle the request we need to add lock-channel to map
		var (
			mu *sync.Mutex
			ok bool
		)
		// Create mutex for same requests
		if mu, ok = r.waiteAnswerMu[hk]; !ok {
			mu = &sync.Mutex{}
			r.waiteAnswerMu[hk] = mu
		}
		mu.Lock()
		if waitCh1, ok1 := r.waitAnswerList[hk]; !ok1 {
			ch := make(chan struct{})
			r.waitAnswerList[hk] = ch
			mu.Unlock() // unlock mutex fast as possible

			// Proxy request to backend
			rrData := r.requestToBack(hk, w, req)
			// Save answer to mem cache
			if err := r.Cache.Add(hk, rrData); err != nil {
				r.log.Err(err).Msg("failed to save data to cache")
			}

			close(ch)
			delete(r.waitAnswerList, hk)
			// Delete removes only item from map, GC remove mutex after removed all references to it.
			delete(r.waiteAnswerMu, hk)
		} else {
			mu.Unlock()
			r.waitAnswer(w, req, hk, waitCh1)
		}
	} else {
		r.waitAnswer(w, req, hk, waitCh)
	}
}

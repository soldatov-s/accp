package routes

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache"
	"github.com/soldatov-s/accp/internal/cache/cachedata"
	"github.com/soldatov-s/accp/internal/cache/external"
	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/httputils"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/limits"
	"github.com/soldatov-s/accp/internal/publisher"
	accpmodels "github.com/soldatov-s/accp/models"
)

const (
	defaultRefreshMutexExpire = 30 * time.Second
	checkInterval             = 100 * time.Millisecond
)

type empty struct{}

type Route struct {
	ctx                 *context.Context
	log                 zerolog.Logger
	Parameters          *Parameters
	Routes              map[string]*Route
	Cache               *cache.Cache
	Pool                *httpclient.Pool
	WaitAnswerList      map[string]chan struct{}
	WaiteAnswerMu       map[string]*sync.Mutex
	Introspect          bool
	IntrospectHydration string
	Publisher           publisher.Publisher
	RefreshTimer        *time.Timer
	Limits              map[string]limits.LimitTable
	Route               string
	Introspector        introspection.Introspector
}

func (r *Route) Initilize(
	ctx *context.Context,
	route string,
	parameters *Parameters,
	externalStorage external.Storage,
	pub publisher.Publisher,
	introspector introspection.Introspector,
) error {
	var err error
	r.InitilizeExcluded(ctx, route, parameters)

	if r.Cache, err = cache.NewCache(ctx, parameters.Cache, externalStorage); err != nil {
		return err
	}

	r.Publisher = pub
	r.Introspector = introspector
	r.Pool = httpclient.NewPool(parameters.Pool.Size, parameters.Pool.Timeout)
	r.WaitAnswerList = make(map[string]chan struct{})
	r.WaiteAnswerMu = make(map[string]*sync.Mutex)
	r.Limits = limits.NewLimits(r.Parameters.Limits)
	r.Introspect = parameters.Introspect

	if parameters.Refresh.Time > 0 {
		r.RefreshTimer = time.AfterFunc(parameters.Refresh.Time, r.RefreshHandler)
	}

	return nil
}

func (r *Route) InitilizeExcluded(
	ctx *context.Context,
	route string,
	parameters *Parameters,
) {
	r.ctx = ctx
	r.log = ctx.GetPackageLogger(empty{})
	r.Parameters = parameters
	r.Pool = httpclient.NewPool(parameters.Pool.Size, parameters.Pool.Timeout)
	r.Route = route
}

func (r *Route) getLimitsFromRequest(req *http.Request) map[string]interface{} {
	limitList := make(map[string]interface{})

	for k, v := range r.Parameters.Limits {
		for _, vv := range v.Header {
			if h := req.Header.Get(vv); h != "" {
				if strings.EqualFold(vv, "authorization") {
					splitToken := strings.Split(h, " ")
					if len(splitToken) < 2 {
						h = splitToken[0]
					} else {
						h = splitToken[1]
					}
				}
				// Always taken client IP
				if strings.EqualFold(vv, "x-forwarded-for") {
					splitIP := strings.Split(h, ",")
					h = splitIP[0]
				}
				limitList[strings.ToLower(k)] = h
			}
		}

		for _, vv := range v.Cookie {
			if c, err := req.Cookie(vv); err == nil {
				limitList[strings.ToLower(k)] = c.Value
			}
		}
	}

	return limitList
}

func (r *Route) checkLimit(k string, v interface{}, result *bool) error {
	if vv, ok := (r.Limits[k])[v]; !ok {
		r.Limits[k][v] = &limits.Limit{
			Counter:    1,
			LastAccess: time.Now().Unix(),
		}
		if err := vv.CreateLimit(r.Route, k, r.Cache.External); err != nil {
			return err
		}
	} else {
		mu, err := r.Cache.External.NewMutexByID(r.Route+"_"+k, defaultRefreshMutexExpire, checkInterval)
		if err != nil {
			r.log.Err(err).Msg("INTERNAL SERVER ERROR")

			return err
		}

		defer func() {
			err1 := mu.Unlock()
			if err1 != nil {
				r.log.Err(err1).Msg("INTERNAL SERVER ERROR")
			}
		}()

		err = mu.Lock()
		if err != nil {
			r.log.Err(err).Msg("INTERNAL SERVER ERROR")

			return err
		}

		if err := vv.LoadLimit(r.Route, k, r.Cache.External); err != nil {
			r.log.Err(err).Msgf("failed to get limit %s from external cache", k)
		}

		vv.Counter++

		if vv.Counter >= r.Parameters.Limits[k].Counter &&
			time.Now().Add(-r.Parameters.Limits[k].PT).Unix() < vv.LastAccess {
			*result = *result && false
			r.log.Debug().Msgf("limit reached: %s", k)
			return nil
		} else if time.Now().Add(-r.Parameters.Limits[k].PT).Unix() >= vv.LastAccess {
			vv.Counter = 1
			vv.LastAccess = time.Now().Unix()
		}

		go func() {
			if err := vv.UpdateLimit(r.Route, k, r.Cache.External); err != nil {
				r.log.Err(err).Msgf("failed to update limit %s in external cache", k)
			}
		}()
	}

	return nil
}

func (r *Route) CheckLimits(req *http.Request) (*bool, error) {
	result := true

	if len(r.Parameters.Limits) == 0 {
		return &result, nil
	}

	limitList := r.getLimitsFromRequest(req)
	if len(limitList) == 0 {
		return &result, nil
	}

	for k, v := range limitList {
		if err := r.checkLimit(k, v, &result); err != nil {
			return nil, err
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

			if r.Cache.External == nil {
				return
			}

			if err := r.Cache.External.Update(k.(string), data); err != nil {
				r.log.Err(err).Msg("failed to update external cache")
			}
		}()

		return true
	})

	r.RefreshTimer.Reset(r.Parameters.Refresh.Time)
}

// Publish publishes request from client to message queue
func (r *Route) Publish(message interface{}) error {
	if r.Publisher == nil {
		return nil
	}
	return r.Publisher.SendMessage(message, r.Parameters.RouteKey)
}

func (r *Route) RequestToBack(w http.ResponseWriter, req *http.Request) *accpmodels.RRData {
	var err error
	// Proxy request to backend
	client := r.Pool.GetFromPool()

	var resp *http.Response
	req.URL, err = url.Parse(r.Parameters.DSN + req.URL.String())
	if err != nil {
		resp = errResponse(err.Error(), http.StatusServiceUnavailable)
	} else {
		// nolint
		resp, err = client.Do(req)
		if err != nil {
			resp = errResponse(err.Error(), http.StatusServiceUnavailable)
		}
	}
	defer resp.Body.Close()

	rrData := accpmodels.NewRRData()
	if err := rrData.Request.Read(req); err != nil {
		r.log.Err(err).Msg("failed to read data from request")
	}

	rrData.Refresh.MaxCount = r.Parameters.Refresh.Count

	if err := rrData.Response.Read(resp); err != nil {
		r.log.Err(err).Msg("failed to read data from response")
	}

	if err := rrData.Response.Write(w); err != nil {
		r.log.Err(err).Msg("failed to write data to client from response")
	}

	return rrData
}

// NotCached is handler for proxy requests to excluded routes and routes which not need to cache
func (r *Route) NotCached(w http.ResponseWriter, req *http.Request) {
	// Proxy request to backend
	client := r.Pool.GetFromPool()

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

func errResponse(errormsg string, code int) *http.Response {
	resp := &http.Response{
		StatusCode: code,
		Body:       ioutil.NopCloser(bytes.NewBufferString(errormsg)),
	}

	resp.Header = make(http.Header)
	resp.Header.Set("Content-Type", "text/plain; charset=utf-8")
	resp.Header.Set("X-Content-Type-Options", "nosniff")

	return resp
}

func (r *Route) HydrationIntrospect(req *http.Request) error {
	if r.Introspector == nil || !r.Introspect {
		r.log.Debug().Msgf("no introspector or disabled introspection: %s", r.Route)
		return nil
	}

	content, err := r.Introspector.IntrospectRequest(req)
	if err != nil {
		return err
	}

	var str string
	switch r.IntrospectHydration {
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
func (r *Route) refresh(rrdata *accpmodels.RRData, hk string) {
	// Check that we have refresh limit by request count
	if rrdata.Refresh.MaxCount == 0 {
		return
	}

	r.log.Debug().Msgf("refresh cache, key %s, maxCount %d, current count %d", hk, rrdata.Refresh.MaxCount, rrdata.Refresh.Counter)

	rrdata.MuLock()
	defer rrdata.MuUnlock()

	if err := rrdata.LoadRefreshCounter(hk, r.Cache.External); err != nil {
		r.log.Err(err).Msg("failed to get refresh-counter from external cache")
	}

	rrdata.Refresh.Counter++

	if rrdata.Refresh.Counter < rrdata.Refresh.MaxCount {
		if err := rrdata.UpdateRefreshCounter(hk, r.Cache.External); err != nil {
			r.log.Err(err).Msg("failed to update refresh counter")
		}
		return
	}

	req, err := rrdata.Request.BuildRequest()
	if err != nil {
		r.log.Err(err).Msg("failed to build request")
		return
	}

	if err = r.HydrationIntrospect(req); err != nil {
		r.log.Err(err).Msg("introspection failed")
		if err = r.Cache.Delete(hk); err != nil {
			r.log.Err(err).Msg("delete key failed")
		}
		return
	}

	client := r.Pool.GetFromPool()
	if err := rrdata.UpdateByRequest(client, req); err != nil {
		r.log.Err(err).Msg("failed to update RRdata")
		if err = r.Cache.Delete(hk); err != nil {
			r.log.Err(err).Msg("delete key failed")
		}
		return
	}

	rrdata.Refresh.Counter = 0
	if err := rrdata.UpdateRefreshCounter(hk, r.Cache.External); err != nil {
		r.log.Err(err).Msg("failed to update refresh counter")
	}

	r.log.Debug().Msgf("cache refreshed")
}

func (r *Route) responseHandle(rrdata *accpmodels.RRData, w http.ResponseWriter, req *http.Request, hk string) {
	// If get rrdata from redis, request will be empty
	if rrdata.Request == nil {
		rrdata.Request = &accpmodels.Request{}
		if err := rrdata.Request.Read(req); err != nil {
			r.log.Err(err).Msg("failed to read data from request")
		}
	}

	if err := rrdata.Response.Write(w); err != nil {
		r.log.Err(err).Msg("failed to write data from cache")
	}

	if err := r.Publish(rrdata.Request); err != nil {
		r.log.Err(err).Msg("failed to publish data")
	}

	go r.refresh(rrdata, hk)
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
	if waitCh, ok := r.WaitAnswerList[hk]; !ok {
		// If we not started to handle the request we need to add lock-channel to map
		var (
			mu *sync.Mutex
			ok bool
		)
		// Create mutex for same requests
		if mu, ok = r.WaiteAnswerMu[hk]; !ok {
			mu = &sync.Mutex{}
			r.WaiteAnswerMu[hk] = mu
		}
		mu.Lock()
		if waitCh1, ok1 := r.WaitAnswerList[hk]; !ok1 {
			ch := make(chan struct{})
			r.WaitAnswerList[hk] = ch
			mu.Unlock() // unlock mutex fast as possible

			// Proxy request to backend
			rrData := r.RequestToBack(w, req)
			// Save answer to mem cache
			if err := r.Cache.Add(hk, rrData); err != nil {
				r.log.Err(err).Msg("failed to save data to cache")
			}

			close(ch)
			delete(r.WaitAnswerList, hk)
			// Delete removes only item from map, GC remove mutex after removed all references to it.
			delete(r.WaiteAnswerMu, hk)
		} else {
			mu.Unlock()
			r.waitAnswer(w, req, hk, waitCh1)
		}
	} else {
		r.waitAnswer(w, req, hk, waitCh)
	}
}

package routes

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache"
	"github.com/soldatov-s/accp/internal/cache/cachedata"
	cacheerrors "github.com/soldatov-s/accp/internal/cache/errors"
	"github.com/soldatov-s/accp/internal/captcha"
	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/httputils"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/limits"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/publisher"
	"github.com/soldatov-s/accp/internal/rabbitmq"
	"github.com/soldatov-s/accp/internal/redis"
	rrdata "github.com/soldatov-s/accp/internal/request_response_data"
)

const (
	hydrationIntrospectPlainText = "plaintext"
	hydrationIntrospectBase64    = "base64"
	hydrationIntrospectHeader    = "Accp-Introspect-Body"
	disabledCachedHeader         = "Accp-Cache-Disable"
	disabledCapchaHeader         = "Accp-Captcha-Disable"
)

type empty struct{}

type Route struct {
	Routes         MapRoutes
	ctx            context.Context
	log            zerolog.Logger
	parameters     *Parameters
	cache          *cache.Cache
	pool           *httpclient.Pool
	waitAnswerList map[string]chan struct{}
	waiteAnswerMu  map[string]*sync.Mutex
	publisher      publisher.Publisher
	refreshTimer   *time.Timer
	limits         map[string]*limits.LimitTable
	route          string
	introspector   introspection.Introspector
	captcher       *captcha.GoogleCaptcha
}

func NewRoute(ctx context.Context, routeName string, params *Parameters) *Route {
	if routeName == "" {
		return nil
	}

	if params == nil {
		params = &Parameters{}
	}

	params.SetDefault()
	r := &Route{
		ctx:            ctx,
		log:            logger.GetPackageLogger(ctx, empty{}),
		parameters:     params,
		route:          routeName,
		Routes:         make(MapRoutes),
		waitAnswerList: make(map[string]chan struct{}),
		waiteAnswerMu:  make(map[string]*sync.Mutex),
		pool:           httpclient.NewPool(params.Pool),
	}

	if !params.NotCaptcha {
		if c := captcha.Get(r.ctx); c != nil {
			r.captcher = c
		}
	}

	if !params.NotIntrospect {
		if i := introspection.Get(r.ctx); i != nil {
			r.introspector = i
		}
	}

	if !params.Cache.Disabled {
		if externalCache := redis.Get(r.ctx); externalCache != nil {
			r.cache = cache.NewCache(r.ctx, params.Cache, externalCache)
		} else {
			r.cache = cache.NewCache(r.ctx, params.Cache, nil)
		}

		if params.Refresh.Time > 0 {
			r.refreshTimer = time.AfterFunc(params.Refresh.Time, r.refreshByTime)
		}

		if r.parameters.RouteKey != "" {
			if p := rabbitmq.Get(r.ctx); p != nil {
				r.publisher = p
			}
		}
	}

	if len(params.Limits) != 0 {
		if r.cache != nil && r.cache.External != nil {
			r.limits = limits.NewLimits(r.route, params.Limits, r.cache.External)
		} else {
			r.limits = limits.NewLimits(r.route, params.Limits, nil)
		}
	}

	return r
}

// route is fully exluded if disabled cache, introspection and limits
func (r *Route) isExcluded() bool {
	return r.parameters.NotIntrospect &&
		r.parameters.Cache.Disabled &&
		len(r.parameters.Limits) == 0
}

func (r *Route) checkLimits(req *http.Request) (*bool, error) {
	result := false
	if len(r.parameters.Limits) == 0 {
		r.log.Debug().Msgf("limits disabled for route: %s", r.route)
		return &result, nil
	}

	limitList, err := limits.NewLimitedParamsOfRequest(r.parameters.Limits, req)
	if err != nil {
		return nil, err
	}
	if len(limitList) == 0 {
		return &result, nil
	}

	result = true
	for k, v := range limitList {
		if err := r.limits[k].Check(v, &result); err != nil {
			return nil, err
		} else if result {
			r.log.Debug().Str("requestID", httputils.GetRequestID(req)).Msgf("limit reached: %s:%s", k, v)
			break
		}
	}

	return &result, nil
}

func (r *Route) refreshHandler(hk string, data *rrdata.RequestResponseData) error {
	data.Mu.Lock()
	defer data.Mu.Unlock()

	if data.Request == nil {
		return nil
	}

	req, err := data.Request.BuildRequest()
	if err != nil {
		return errors.Wrap(err, "failed to build request")
	}

	client := r.pool.GetFromPool()
	defer r.pool.PutToPool(client)

	if err := data.UpdateByRequest(client, req); err != nil {
		if err = r.cache.Delete(hk); err != nil {
			return errors.Wrap(err, "failed to update request/response data, delete key failed")
		}
		return errors.Wrap(err, "failed to update request/response data")
	}

	r.log.Debug().Msgf("%s: cache refreshed", hk)

	if r.cache.External == nil {
		return nil
	}

	if err := r.cache.External.Update(hk, data); err != nil {
		return errors.Wrap(err, "failed to update external cache")
	}

	r.log.Debug().Msgf("%s: external cache refreshed", hk)
	return nil
}

func (r *Route) refreshByTime() {
	r.cache.Memory.Range(func(k, v interface{}) bool {
		data := v.(*cachedata.CacheItem).Data.(*rrdata.RequestResponseData)
		hk := k.(string)
		go func() {
			r.log.Debug().Msgf("try to refersh %s by time", hk)
			if err := r.refreshHandler(hk, data); err != nil {
				r.log.Error().Err(err).Msgf("%s: refresh cache failed", hk)
			}
		}()

		return true
	})

	r.refreshTimer.Reset(r.parameters.Refresh.Time)
}

// Publish publishes request from client to message queue
func (r *Route) Publish(message interface{}) error {
	if r.publisher == nil {
		return nil
	}
	return r.publisher.SendMessage(message, r.parameters.RouteKey)
}

func (r *Route) requestToBack(hk string, w http.ResponseWriter, req *http.Request) *rrdata.RequestResponseData {
	var err error
	// Proxy request to backend
	client := r.pool.GetFromPool()
	defer r.pool.PutToPool(client)

	rrData := rrdata.NewRequestResponseData(hk, r.parameters.Refresh.MaxCount, r.cache.External)

	var resp *http.Response
	proxyReq, err := httputils.CopyRequestWithDSN(req, r.parameters.DSN)
	if err != nil {
		resp = httputils.ErrResponse(err.Error(), http.StatusServiceUnavailable)
		rrData.Request = nil
	} else {
		// nolint : bodyclose
		if err = rrData.Request.Read(proxyReq); err != nil {
			resp = httputils.ErrResponse(err.Error(), http.StatusServiceUnavailable)
			rrData.Request = nil
		} else if resp, err = client.Do(proxyReq); err != nil {
			resp = httputils.ErrResponse(err.Error(), http.StatusServiceUnavailable)
		}
	}
	defer resp.Body.Close()

	if err := rrData.Response.Read(resp); err != nil {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("failed to read request/response data")
	}

	if err := rrData.Response.Write(w, rrdata.ResponseBack); err != nil {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("failed to write data to client from response")
	}

	return rrData
}

// notCached is handler for proxy requests to excluded routes and routes which not need to cache
func (r *Route) notCached(w http.ResponseWriter, req *http.Request) {
	client := r.pool.GetFromPool()
	defer r.pool.PutToPool(client)

	r.log.Debug().Msg(req.URL.String())

	var err error
	proxyReq, err := httputils.CopyRequestWithDSN(req, r.parameters.DSN)
	if err != nil {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("request duplication failed")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("request to back failed")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Mark that it is a proxy request
	resp.Header.Add(rrdata.ResponseSourceHeader, rrdata.ResponseBypass.String())

	if err := httputils.CopyHTTPResponse(w, resp); err != nil {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("copy http response failed")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
}

func (r *Route) hydrationIntrospect(req *http.Request) error {
	if r.parameters.NotIntrospect || r.introspector == nil {
		r.log.Debug().Str("requestID", httputils.GetRequestID(req)).Msgf("no introspector or disabled introspection: %s", r.route)
		return nil
	}

	content, err := r.introspector.IntrospectRequest(req)
	if err != nil {
		return err
	}

	var str string
	switch r.parameters.IntrospectHydration {
	case hydrationIntrospectPlainText:
		str = strings.ReplaceAll(strings.ReplaceAll(string(content), "\"", "\\\""), "\n", "")
	case hydrationIntrospectBase64:
		str = base64.StdEncoding.EncodeToString(content)
	default:
		return nil
	}

	req.Header.Add(hydrationIntrospectHeader, str)
	r.log.Debug().Str("requestID", httputils.GetRequestID(req)).Msgf("introspect header: %s", str)

	return nil
}

// Checking captcha
func (r *Route) validateCaptcha(w http.ResponseWriter, req *http.Request) {
	var (
		captchaSrc captcha.SrcCaptcha
		err        error
	)
	if !r.parameters.NotCaptcha && req.Header.Get(disabledCapchaHeader) != r.parameters.IgnoreCapchaKey && r.captcher != nil {
		captchaSrc, err = r.captcher.Validate(req)
		if errors.Is(err, captcha.ErrCaptchaFailed) {
			r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("captcha failed")
			http.Error(w, "limit reached", http.StatusBadRequest)
		}
		if err != nil {
			r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("captcha failed")
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
	} else {
		r.log.Debug().Str("requestID", httputils.GetRequestID(req)).Msgf("no captcher or disabled captcha: %s", r.route)
	}

	responseWrapper := httputils.NewResponseWrapper(w)

	r.proxyHandler(responseWrapper, req)

	if captchaSrc == captcha.CaptchaFromGoogle && responseWrapper.GetStatusCode() == http.StatusOK {
		if err := r.captcher.GenerateCaptchaJWT(w); err != nil {
			r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("captcha generate jwt failed")
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
	}
}

// refresh incremets refresh count and checks that we not reached the limit
func (r *Route) refresh(data *rrdata.RequestResponseData, hk string) {
	// Check that we have refresh limit by request count
	if r.parameters.Refresh.MaxCount == 0 {
		return
	}
	if err := data.Response.Refresh.Inc(); err != nil {
		r.log.Error().Err(err).Msg("failed to inc refresh counter")
		return
	} else if data.Response.Refresh.Check() {
		return
	}

	r.log.Debug().Msgf("try to refersh %s by counter", hk)
	if err := r.refreshHandler(hk, data); err != nil {
		r.log.Error().Err(err).Msgf("%s: refresh cache failed", hk)
	}
}

func (r *Route) responseHandle(data *rrdata.RequestResponseData, w http.ResponseWriter, req *http.Request, hk string) {
	// If data was getted from redis, the request will be empty
	if data.Request == nil {
		var err error
		if data.Request, err = rrdata.NewRequestData(req); err != nil {
			r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("failed to read data from request")
		}
	}

	if err := data.Response.Write(w, rrdata.ResponseCache); err != nil {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("failed to write data from cache")
	}

	if err := r.Publish(data.Request); err != nil {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("failed to publish data")
	}

	go r.refresh(data, hk)
}

func (r *Route) waitAnswer(w http.ResponseWriter, req *http.Request, hk string, ch chan struct{}) {
	<-ch

	var (
		data *rrdata.RequestResponseData
		err  error
	)
	if data, err = r.cache.Select(hk); err != nil {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("failed to get data from cache")
		http.Error(w, "failed to get data from cache", http.StatusServiceUnavailable)
		return
	}

	r.responseHandle(data, w, req, hk)
}

func (r *Route) cachedHandler(w http.ResponseWriter, req *http.Request) {
	hk, err := httputils.HashRequest(req)
	if err != nil {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("failed to calculate request hash")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Finding a response to a request in the memory cache
	if data, err1 := r.cache.Select(hk); err1 == nil {
		r.responseHandle(data, w, req, hk)
		return
	} else if err1 != cacheerrors.ErrNotFound {
		r.log.Err(err1).Str("requestID", httputils.GetRequestID(req)).Msg("failed to get data from cache")
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
			if err := r.cache.Add(hk, rrData); err != nil {
				r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("failed to save data to cache")
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
func (r *Route) proxyHandler(w http.ResponseWriter, req *http.Request) {
	// Checking an authorization token
	err := r.hydrationIntrospect(req)
	var e *introspection.ErrTokenInactive
	if errors.As(err, &e) || errors.Is(err, introspection.ErrBadAuthRequest) {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("intropsection failed")
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	} else if err != nil {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("intropsection failed")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Checking limits
	if res, err := r.checkLimits(req); err != nil {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("check limits failed")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	} else if *res {
		r.log.Err(err).Str("requestID", httputils.GetRequestID(req)).Msg("check limits failed")
		http.Error(w, "limit reached", http.StatusTooManyRequests)
		return
	}

	// It's a cached request, checking allowed methods, check header
	if !r.parameters.Cache.Disabled &&
		r.parameters.Methods.Has(req.Method) &&
		req.Header.Get(disabledCachedHeader) != "true" {
		r.cachedHandler(w, req)
		return
	}

	// Ooops, not allowed methods or nor cached request, pass request to backend
	r.notCached(w, req)
}

func (r *Route) ProxyHandler(w http.ResponseWriter, req *http.Request) {
	r.log.Debug().Msgf("proxy route: %s", r.route)

	if r.parameters.DSN == "" {
		r.log.Error().Msgf("route %s not found", req.URL.String())
		http.Error(w, "route "+req.URL.String()+" not found", http.StatusNotFound)
		return
	}

	r.validateCaptcha(w, req)
}

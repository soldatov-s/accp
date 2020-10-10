package httpproxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"net/http/pprof"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache/external"
	ctxint "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httputils"
	"github.com/soldatov-s/accp/internal/introspector"
	"github.com/soldatov-s/accp/internal/publisher"
	accpmodels "github.com/soldatov-s/accp/models"
)

type empty struct{}

type Config struct {
	Listen    string
	Hydration struct {
		RequestID  bool
		Introspect string
	}
	Routes   map[string]*RouteConfig
	Excluded map[string]*RouteConfig
}

type HTTPProxy struct {
	cfg          *Config
	ctx          *ctxint.Context
	log          zerolog.Logger
	srv          *http.Server
	routes       map[string]*Route
	excluded     map[string]*Route
	introspector introspector.Introspector
}

func NewHTTPProxy(
	ctx *ctxint.Context,
	cfg *Config,
	i introspector.Introspector,
	externalStorage external.Storage,
	publisher publisher.Publisher,
) (*HTTPProxy, error) {
	p := &HTTPProxy{
		ctx:          ctx,
		cfg:          cfg,
		introspector: i,
	}

	p.srv = &http.Server{
		Addr:           cfg.Listen,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	p.srv.Handler = http.HandlerFunc(p.proxyHandler)

	p.log = ctx.GetPackageLogger(empty{})
	p.routes = make(map[string]*Route)

	if err := p.fillRoutes(ctx, externalStorage, publisher, p.cfg.Routes, p.routes, nil, ""); err != nil {
		return nil, err
	}

	if err := p.fillExcludedRoutes(p.cfg.Excluded, p.excluded); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *HTTPProxy) fillRoutes(
	ctx *ctxint.Context,
	externalStorage external.Storage,
	publisher publisher.Publisher,
	rc map[string]*RouteConfig,
	r map[string]*Route,
	parentParameters *RouteParameters,
	parentRoute string,
) error {
	for k, v := range rc {
		routes := r
		strs := strings.Split(k, "/")
		for _, s := range strs {
			if route, ok := routes[s]; !ok {
				routes[k] = &Route{}
			} else {
				routes = route.Routes
			}
		}

		parameters := parentParameters
		if parameters != nil {
			parameters = parameters.Merge(v.Parameters)
		} else {
			parameters = v.Parameters
		}

		lastPartOfRoute := strs[len(strs)-1]
		if err := routes[lastPartOfRoute].Initilize(ctx, parentRoute+"/"+k, parameters, externalStorage, publisher); err != nil {
			return err
		}

		if err := p.fillRoutes(
			ctx,
			externalStorage,
			publisher,
			v.Routes,
			routes[lastPartOfRoute].Routes,
			parameters,
			parentRoute+"/"+k,
		); err != nil {
			return err
		}
	}

	return nil
}

func (p *HTTPProxy) fillExcludedRoutes(
	rc map[string]*RouteConfig,
	r map[string]*Route,
) error {
	for k, v := range rc {
		routes := r
		strs := strings.Split(k, "/")
		for _, s := range strs {
			if route, ok := routes[s]; !ok {
				routes[k] = &Route{}
			} else {
				routes = route.Routes
			}
		}

		lastPartOfRoute := strs[len(strs)-1]
		if err := p.fillExcludedRoutes(v.Routes, routes[lastPartOfRoute].Routes); err != nil {
			return err
		}
	}

	return nil
}

func (p *HTTPProxy) findRoute(r *http.Request) *Route {
	strs := strings.Split(r.URL.Path, "/")
	var (
		route *Route
		ok    bool
	)

	routes := p.routes
	for _, s := range strs {
		if route, ok = routes[s]; !ok {
			return route
		}
		routes = route.Routes
	}

	return route
}

func (p *HTTPProxy) findExcludedRoute(r *http.Request) *Route {
	strs := strings.Split(r.URL.Path, "/")
	var (
		route *Route
		ok    bool
	)

	routes := p.excluded
	for _, s := range strs {
		if route, ok = routes[s]; !ok {
			return route
		}
		routes = route.Routes
	}

	return route
}

func (p *HTTPProxy) waitAnswer(w http.ResponseWriter, r *http.Request, hk string, ch chan struct{}, route *Route) {
	<-ch

	if data, err := route.Cache.Select(hk); err == nil {
		rrdata, ok := data.(*accpmodels.RRData)
		if !ok {
			p.log.Err(err).Msg("failed to convert data from cache to RRData")
			return
		}

		// If get rrdata from redis, request will be empty
		if rrdata.Request == nil {
			if err := rrdata.Request.Read(r); err != nil {
				p.log.Err(err).Msg("failed to read data from request")
			}
		}

		if err := rrdata.Response.Write(w); err != nil {
			p.log.Err(err).Msg("failed to write data from cache")
		}

		if err := route.Publish(rrdata.Request); err != nil {
			p.log.Err(err).Msg("failed to publish data")
		}

		// Check that we have refresh limit by request count
		if rrdata.Refresh.MaxCount == 0 {
			return
		}

		go func() {
			rrdata.Refresh.Mu.Lock()
			defer rrdata.Refresh.Mu.Unlock()

			if err := rrdata.LoadRefreshCounter(hk, route.Cache.External); err != nil {
				p.log.Err(err).Msg("failed to get refresh-counter from external cache")
			}

			rrdata.Refresh.Counter++

			if rrdata.Refresh.Counter < rrdata.Refresh.MaxCount {
				if err := rrdata.UpdateRefreshCounter(hk, route.Cache.External); err != nil {
					p.log.Err(err).Msg("failed to update refresh counter")
				}
				return
			}

			client := route.Pool.GetFromPool()
			if err := rrdata.Update(client); err != nil {
				p.log.Err(err).Msg("failed to update RRdata")
			}

			rrdata.Refresh.Counter = 0
			if err := rrdata.UpdateRefreshCounter(hk, route.Cache.External); err != nil {
				p.log.Err(err).Msg("failed to update refresh counter")
			}
		}()

		return
	}

	http.Error(w, "failed to get data from cache", http.StatusServiceUnavailable)
}

func (p *HTTPProxy) getHandler(route *Route, w http.ResponseWriter, r *http.Request) {
	hk, err := p.hashKey(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Finding a response to a request in the memory cache
	if data, err := route.Cache.Select(hk); err == nil {
		rrdata, ok := data.(*accpmodels.RRData)
		if !ok {
			p.log.Err(err).Msg("failed to convert data from cache to RRData")
			return
		}

		// If get rrdata from redis, request will be empty
		if rrdata.Request == nil {
			if err := rrdata.Request.Read(r); err != nil {
				p.log.Err(err).Msg("failed to read data from request")
			}
		}

		if err := rrdata.Response.Write(w); err != nil {
			p.log.Err(err).Msg("failed to write data from cache")
		}

		if err := route.Publish(rrdata.Request); err != nil {
			p.log.Err(err).Msg("failed to publish data")
		}

		// Check that we have refresh limit by request count
		if rrdata.Refresh.MaxCount == 0 {
			return
		}

		go func() {
			rrdata.Refresh.Mu.Lock()
			defer rrdata.Refresh.Mu.Unlock()

			if err := rrdata.LoadRefreshCounter(hk, route.Cache.External); err != nil {
				p.log.Err(err).Msg("failed to get refresh-counter from external cache")
			}

			rrdata.Refresh.Counter++

			if rrdata.Refresh.Counter < rrdata.Refresh.MaxCount {
				if err := rrdata.UpdateRefreshCounter(hk, route.Cache.External); err != nil {
					p.log.Err(err).Msg("failed to update refresh counter")
				}
				return
			}

			client := route.Pool.GetFromPool()
			if err := rrdata.Update(client); err != nil {
				p.log.Err(err).Msg("failed to update RRdata")
			}

			rrdata.Refresh.Counter = 0
			if err := rrdata.UpdateRefreshCounter(hk, route.Cache.External); err != nil {
				p.log.Err(err).Msg("failed to update refresh counter")
			}
		}()
	}

	// Check that we not started to handle the request
	if waitCh, ok := route.WaitAnswerList[hk]; !ok {
		// If we not started to handle the request we need to add lock-channel to map
		var (
			mu *sync.Mutex
			ok bool
		)
		// Create mutex for same requests
		if mu, ok = route.WaiteAnswerMu[hk]; !ok {
			mu = &sync.Mutex{}
			route.WaiteAnswerMu[hk] = mu
		}
		mu.Lock()
		if waitCh1, ok1 := route.WaitAnswerList[hk]; !ok1 {
			ch := make(chan struct{})
			route.WaitAnswerList[hk] = ch
			mu.Unlock() // unlock mutex fast as possible

			// Proxy request to backend
			client := route.Pool.GetFromPool()

			resp, err := client.Do(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			defer resp.Body.Close()

			rrData := &accpmodels.RRData{}
			if err := rrData.Request.Read(r); err != nil {
				p.log.Err(err).Msg("failed to read data from request")
			}

			rrData.Refresh.MaxCount = route.parameters.Refresh.Count

			if err := rrData.Response.Read(resp); err != nil {
				p.log.Err(err).Msg("failed to read data from response")
			}

			if err := rrData.Response.Write(w); err != nil {
				p.log.Err(err).Msg("failed to write data from response")
			}

			// Save answer to mem cache
			if err := route.Cache.Add(hk, rrData); err != nil {
				p.log.Err(err).Msg("failed to save data to cache")
			}

			close(ch)
			delete(route.WaitAnswerList, hk)
			// Delete removes only item from map, GC remove mutex after removed all references to it.
			delete(route.WaiteAnswerMu, hk)
		} else {
			mu.Unlock()
			p.waitAnswer(w, r, hk, waitCh1, route)
		}
	} else {
		p.waitAnswer(w, r, hk, waitCh, route)
	}
}

func (p *HTTPProxy) defaultHandler(w http.ResponseWriter, r *http.Request) {
	// Proxy request to backend
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	httputils.CopyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
}

func (p *HTTPProxy) proxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		switch r.URL.Path {
		case "/health/alive":
			w.WriteHeader(http.StatusOK)
			w.Header().Add("Content-Type", "application/json")
			_, err := w.Write([]byte("{\"result\":\"ok\"}"))
			if err != nil {
				p.log.Err(err).Msg("failed write body")
			}
			return
		case "/metrics":
			promhttp.Handler().ServeHTTP(w, r)
			return
			// TODO: enable pprof via config
		case "/debug/pprof/":
			pprof.Index(w, r)
			return
		case "/debug/pprof/cmdline":
			pprof.Cmdline(w, r)
			return
		case "/debug/pprof/profile":
			pprof.Profile(w, r)
			return
		case "/debug/pprof/symbol":
			pprof.Symbol(w, r)
			return
		case "/debug/pprof/trace":
			pprof.Trace(w, r)
			return
		}
	}

	// Handle excluded routes
	route := p.findExcludedRoute(r)
	if route != nil {
		p.defaultHandler(w, r)
		return
	}

	route = p.findRoute(r)

	// Adding to request header the requestID
	p.hydrationID(r)

	// The check an authorization token
	if p.introspector != nil && route.Introspect {
		introspectBody, err := p.introspector.IntrospectRequest(r)
		if _, ok := err.(*introspector.ErrTokenInactive); ok {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		p.hydrationIntrospect(r, introspectBody)
	}

	// Check limits
	if res, err := route.CheckLimits(r); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	} else if !*res {
		http.Error(w, "limit reached", http.StatusTooManyRequests)
		return
	}

	switch r.Method {
	case http.MethodGet:
		p.getHandler(route, w, r)
	default:
		p.defaultHandler(w, r)
	}
}

func (p *HTTPProxy) hashKey(r *http.Request) (string, error) {
	buf := bytes.Buffer{}
	if _, err := io.Copy(&buf, r.Body); err != nil {
		return "", err
	}

	p.log.Debug().Msgf("request: %s; body: %s", r.URL.RequestURI(), buf.String())
	sum := sha256.New().Sum([]byte(r.URL.RequestURI() + buf.String()))
	return base64.URLEncoding.EncodeToString(sum), nil
}

func (p *HTTPProxy) hydrationID(r *http.Request) {
	if !p.cfg.Hydration.RequestID {
		return
	}

	requestID := r.Header.Get("x-request-id")
	if requestID == "" {
		newUUID, err := uuid.NewRandom()
		if err != nil {
			p.log.Err(err).Msg("failed to generate requesID")
			return
		}
		r.Header.Add("x-request-id", newUUID.String())
	}
}

func (p *HTTPProxy) hydrationIntrospect(r *http.Request, content []byte) {
	var str string
	switch p.cfg.Hydration.Introspect {
	case "nothing":
		return
	case "plaintext":
		str = strings.ReplaceAll(strings.ReplaceAll(string(content), "\"", "\\\""), "\n", "")
	case "base64":
		str = base64.StdEncoding.EncodeToString(content)
	}

	r.Header.Add("accp-introspect-body", str)
	p.log.Debug().Msgf("accp-introspect-body header: %s", r.Header.Get("accp-introspect-body"))
}

func (p *HTTPProxy) Start() {
	p.log.Debug().Msg("start proxy")
	p.log.Fatal().Err(p.srv.ListenAndServe()).Msg("failed to start proxy")
}

func (p *HTTPProxy) Shutdown() error {
	return p.srv.Shutdown(context.Background())
}

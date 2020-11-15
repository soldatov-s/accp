package httpproxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache/external"
	ctxint "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httputils"
	"github.com/soldatov-s/accp/internal/introspection"
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
	introspector introspection.Introspector
}

func NewHTTPProxy(
	ctx *ctxint.Context,
	cfg *Config,
	i introspection.Introspector,
	externalStorage external.Storage,
	pub publisher.Publisher,
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
	p.excluded = make(map[string]*Route)

	if err := p.fillRoutes(ctx, externalStorage, pub, p.cfg.Routes, p.routes, nil, ""); err != nil {
		return nil, err
	}

	return p, nil
}

// fillRoutes fill routes map
// r - routes
// er - excluded routes
func (p *HTTPProxy) fillRoutes(
	ctx *ctxint.Context,
	externalStorage external.Storage,
	pub publisher.Publisher,
	rc map[string]*RouteConfig,
	r map[string]*Route,
	parentParameters *RouteParameters,
	parentRoute string,
) error {
	// Sort config map
	keys := make([]string, 0)

	for k := range rc {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, configKey := range keys {
		k := strings.Trim(configKey, "/")
		p.log.Debug().Msgf("parse route \"%s\"", k)
		// Check for duplicate
		if p.FindRouteByPath(parentRoute+"/"+k) != nil {
			p.log.Warn().Msgf("duplicated route: %s", parentRoute+"/"+k)
			return nil
		}

		var previousLevelRoutes map[string]*Route
		routes := r
		strs := strings.Split(k, "/")

		// For example:
		// root/
		//   - level1_node1
		//     - level2_node1
		//     - level2_node2
		//   - level1_node2
		for _, s := range strs {
			if s == "" {
				continue
			}
			p.log.Debug().Msgf("parse path item \"%s\"", s)
			if route, ok := routes[s]; !ok {
				p.log.Debug().Msgf("create route node \"%s\"", s)
				routes[s] = &Route{
					Routes: make(map[string]*Route),
				}
				previousLevelRoutes = routes
				routes = routes[s].Routes
			} else {
				routes = route.Routes
			}
		}

		parameters := parentParameters
		if parameters != nil {
			parameters = parameters.Merge(rc[configKey].Parameters)
		} else {
			parameters = rc[configKey].Parameters
			if err := parameters.Initilize(); err != nil {
				return err
			}
		}

		lastPartOfRoute := strs[len(strs)-1]
		if lastPartOfRoute == "" {
			lastPartOfRoute = strs[len(strs)-2]
		}

		p.log.Debug().Msgf("last part of route \"%s\" is \"%s\"", k, lastPartOfRoute)

		if err := previousLevelRoutes[lastPartOfRoute].Initilize(ctx, parentRoute+"/"+k, parameters, externalStorage, pub); err != nil {
			return err
		}

		if rc[configKey] == nil {
			return nil
		}

		if err := p.fillExcludedRoutes(ctx, rc[configKey], parentRoute+"/"+k, parameters); err != nil {
			return nil
		}

		if err := p.fillRoutes(
			ctx,
			externalStorage,
			pub,
			rc[configKey].Routes,
			previousLevelRoutes[lastPartOfRoute].Routes,
			parameters,
			parentRoute+"/"+k,
		); err != nil {
			return err
		}
	}

	return nil
}

func (p *HTTPProxy) fillExcludedRoutes(
	ctx *ctxint.Context,
	rc *RouteConfig,
	parentRoute string,
	parentParameters *RouteParameters,
) error {
	for _, route := range rc.Excluded {
		k := strings.Trim(parentRoute+"/"+route, "/")
		p.log.Debug().Msgf("parse excluded route \"%s\"", k)
		// Check for duplicate
		if p.FindExcludedRouteByPath(k) != nil {
			p.log.Warn().Msgf("duplicated excluded route: %s", k)
			return nil
		}

		var previousLevelRoutes map[string]*Route
		routes := p.excluded

		p.log.Debug().Msgf("parse excluded route \"%s\"", k)
		strs := strings.Split(k, "/")
		for _, s := range strs {
			if s == "" {
				continue
			}
			p.log.Debug().Msgf("parse path item \"%s\"", s)
			if route, ok := routes[s]; !ok {
				p.log.Debug().Msgf("create excluded route node \"%s\"", s)
				routes[s] = &Route{
					Routes: make(map[string]*Route),
				}
				previousLevelRoutes = routes
				routes = routes[s].Routes
			} else {
				routes = route.Routes
			}
		}

		lastPartOfRoute := strs[len(strs)-1]
		if lastPartOfRoute == "" {
			lastPartOfRoute = strs[len(strs)-2]
		}

		p.log.Debug().Msgf("last part of route \"%s\" is \"%s\"", k, lastPartOfRoute)
		previousLevelRoutes[lastPartOfRoute].InitilizeExcluded(ctx, k, parentParameters)
	}
	return nil
}

func (p *HTTPProxy) findRoute(path string, routes map[string]*Route) *Route {
	path = strings.Trim(path, "/")
	strs := strings.Split(path, "/")
	var (
		route *Route
		ok    bool
	)

	for _, s := range strs {
		if s == "" {
			continue
		}
		p.log.Debug().Msgf("search path item \"%s\"", s)
		if route, ok = routes[s]; !ok {
			return route
		}
		routes = route.Routes
	}

	return route
}

func (p *HTTPProxy) FindRouteByPath(path string) *Route {
	return p.findRoute(path, p.routes)
}

func (p *HTTPProxy) FindRouteByHTTPRequest(r *http.Request) *Route {
	return p.FindRouteByPath(r.URL.Path)
}

func (p *HTTPProxy) FindExcludedRouteByPath(path string) *Route {
	return p.findRoute(path, p.excluded)
}

func (p *HTTPProxy) FindExcludedRouteByHTTPRequest(r *http.Request) *Route {
	return p.FindExcludedRouteByPath(r.URL.Path)
}

func (p *HTTPProxy) refresh(rrdata *accpmodels.RRData, hk string, route *Route) {
	// Check that we have refresh limit by request count
	if rrdata.Refresh.MaxCount == 0 {
		return
	}

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
}

func (p *HTTPProxy) responseHandle(data interface{}, w http.ResponseWriter, r *http.Request, hk string, route *Route) {
	rrdata, ok := data.(*accpmodels.RRData)
	if !ok {
		p.log.Error().Msg("failed to convert data from cache to RRData")
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

	go p.refresh(rrdata, hk, route)
}

func (p *HTTPProxy) waitAnswer(w http.ResponseWriter, r *http.Request, hk string, ch chan struct{}, route *Route) {
	<-ch

	if data, err := route.Cache.Select(hk); err == nil {
		p.responseHandle(data, w, r, hk, route)
		return
	}

	http.Error(w, "failed to get data from cache", http.StatusServiceUnavailable)
}

func errResponse(error string, code int) *http.Response {
	resp := &http.Response{
		StatusCode: code,
		Body:       ioutil.NopCloser(bytes.NewBufferString(error)),
	}

	resp.Header.Set("Content-Type", "text/plain; charset=utf-8")
	resp.Header.Set("X-Content-Type-Options", "nosniff")

	return resp
}

func (p *HTTPProxy) CachedHandler(route *Route, w http.ResponseWriter, r *http.Request) {
	hk, err := p.hashKey(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Finding a response to a request in the memory cache
	if data, err := route.Cache.Select(hk); err == nil {
		p.responseHandle(data, w, r, hk, route)
		return
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

			var resp *http.Response
			r.URL, err = url.Parse(route.Parameters.DSN + r.URL.String())
			if err != nil {
				resp = errResponse(err.Error(), http.StatusServiceUnavailable)
			} else {
				// nolint
				resp, err = client.Do(r)
				if err != nil {
					resp = errResponse(err.Error(), http.StatusServiceUnavailable)
				}
			}
			defer resp.Body.Close()

			rrData := accpmodels.NewRRData()
			if err := rrData.Request.Read(r); err != nil {
				p.log.Err(err).Msg("failed to read data from request")
			}

			rrData.Refresh.MaxCount = route.Parameters.Refresh.Count

			if err := rrData.Response.Read(resp); err != nil {
				p.log.Err(err).Msg("failed to read data from response")
			}

			if err := rrData.Response.Write(w); err != nil {
				p.log.Err(err).Msg("failed to write data to client from response")
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

// NonCachedHandler is handler for proxy requests to excluded routes and routes whitch not need to cache
func (p *HTTPProxy) NonCachedHandler(route *Route, w http.ResponseWriter, r *http.Request) {
	// Proxy request to backend
	client := route.Pool.GetFromPool()

	var err error
	r.URL, err = url.Parse(route.Parameters.DSN + r.URL.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	p.log.Debug().Msg(r.URL.String())

	resp, err := client.Do(r)
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
	// Handle excluded routes
	route := p.FindExcludedRouteByHTTPRequest(r)
	if route != nil {
		p.NonCachedHandler(route, w, r)
		return
	}

	route = p.FindRouteByHTTPRequest(r)

	// Adding to request header the requestID
	p.HydrationID(r)

	// The check an authorization token
	err := p.HydrationIntrospect(route, r)
	if _, ok := err.(*introspection.ErrTokenInactive); ok {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Check limits
	if res, err := route.CheckLimits(r); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	} else if !*res {
		http.Error(w, "limit reached", http.StatusTooManyRequests)
		return
	}

	if route.Parameters.Methods.Has(r.Method) {
		p.CachedHandler(route, w, r)
		return
	}
	p.NonCachedHandler(route, w, r)
}

func (p *HTTPProxy) hashKey(r *http.Request) (string, error) {
	buf := bytes.Buffer{}
	if r.Body != nil {
		if _, err := io.Copy(&buf, r.Body); err != nil {
			return "", err
		}
	}

	p.log.Debug().Msgf("request: %s; body: %s", r.URL.RequestURI(), buf.String())
	sum := sha256.New().Sum([]byte(r.URL.RequestURI() + buf.String()))
	return base64.URLEncoding.EncodeToString(sum), nil
}

func (p *HTTPProxy) HydrationID(r *http.Request) {
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
		r.Header.Del("x-request-id")
		r.Header.Add("x-request-id", newUUID.String())
	}
}

func (p *HTTPProxy) HydrationIntrospect(route *Route, r *http.Request) error {
	if p.introspector == nil || !route.Introspect {
		p.log.Debug().Msgf("no introspector or disabled introspection: %s", route.Route)
		return nil
	}

	content, err := p.introspector.IntrospectRequest(r)
	if err != nil {
		return err
	}

	var str string
	switch p.cfg.Hydration.Introspect {
	case "nothing":
		return nil
	case "plaintext":
		str = strings.ReplaceAll(strings.ReplaceAll(string(content), "\"", "\\\""), "\n", "")
	case "base64":
		str = base64.StdEncoding.EncodeToString(content)
	}

	r.Header.Add("accp-introspect-body", str)
	p.log.Debug().Msgf("accp-introspect-body header: %s", r.Header.Get("accp-introspect-body"))

	return nil
}

func (p *HTTPProxy) Start() {
	p.log.Debug().Msg("start proxy")
	p.log.Fatal().Err(p.srv.ListenAndServe()).Msg("failed to start proxy")
}

func (p *HTTPProxy) Shutdown() error {
	return p.srv.Shutdown(context.Background())
}

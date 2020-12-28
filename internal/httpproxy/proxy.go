package httpproxy

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache/external"
	ctxint "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/publisher"
	"github.com/soldatov-s/accp/internal/routes"
)

type empty struct{}

type HTTPProxy struct {
	cfg          *Config
	ctx          *ctxint.Context
	log          zerolog.Logger
	srv          *http.Server
	routes       routes.MapRoutes
	excluded     routes.MapRoutes
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
	p.routes = make(map[string]*routes.Route)
	p.excluded = make(map[string]*routes.Route)

	if err := p.fillRoutes(ctx, externalStorage, pub, p.introspector, p.cfg.Routes, p.routes, nil, ""); err != nil {
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
	introspector introspection.Introspector,
	rc map[string]*routes.Config,
	r map[string]*routes.Route,
	parentParameters *routes.Parameters,
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
		if err := p.routes.FindDuplicated(parentRoute + "/" + k); err != nil {
			p.log.Warn().Err(err).Msg("duplicated route")
			return err
		}

		var previousLevelRoutes map[string]*routes.Route
		routes1 := r
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
			if route, ok := routes1[s]; !ok {
				p.log.Debug().Msgf("create route node \"%s\"", s)
				routes1[s] = &routes.Route{
					Routes: make(map[string]*routes.Route),
				}
				previousLevelRoutes = routes1
				routes1 = routes1[s].Routes
			} else {
				routes1 = route.Routes
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

		if err := previousLevelRoutes[lastPartOfRoute].Initilize(ctx, parentRoute+"/"+k, parameters, externalStorage, pub, introspector); err != nil {
			return err
		}

		if rc[configKey] == nil {
			return nil
		}

		if err := p.fillExcludedRoutes(ctx, rc[configKey], parentRoute+"/"+k, parameters); err != nil {
			return err
		}

		if err := p.fillRoutes(
			ctx,
			externalStorage,
			pub,
			introspector,
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
	rc *routes.Config,
	parentRoute string,
	parentParameters *routes.Parameters,
) error {
	for _, route := range rc.Excluded {
		k := strings.Trim(parentRoute+"/"+route, "/")
		p.log.Debug().Msgf("parse excluded route \"%s\"", k)
		// Check for duplicate
		if err := p.excluded.FindDuplicated(k); err != nil {
			p.log.Warn().Err(err).Msg("duplicated route")
			return err
		}

		var previousLevelRoutes map[string]*routes.Route
		routes1 := p.excluded

		p.log.Debug().Msgf("parse excluded route \"%s\"", k)
		strs := strings.Split(k, "/")
		for _, s := range strs {
			if s == "" {
				continue
			}
			p.log.Debug().Msgf("parse path item \"%s\"", s)
			if route, ok := routes1[s]; !ok {
				p.log.Debug().Msgf("create excluded route node \"%s\"", s)
				routes1[s] = &routes.Route{
					Routes: make(map[string]*routes.Route),
				}
				previousLevelRoutes = routes1
				routes1 = routes1[s].Routes
			} else {
				routes1 = route.Routes
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

func (p *HTTPProxy) proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Handle excluded routes
	route := p.excluded.FindRouteByHTTPRequest(r)
	if route != nil {
		route.NotCached(w, r)
		return
	}

	route = p.routes.FindRouteByHTTPRequest(r)

	// Adding to request header the requestID
	p.HydrationID(r)

	// The check an authorization token
	err := route.HydrationIntrospect(r)
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

	if route.Parameters.Cache.Disabled {
		route.NotCached(w, r)
		return
	}

	if route.Parameters.Methods.Has(r.Method) {
		route.CachedHandler(w, r)
		return
	}
	route.NotCached(w, r)
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

func (p *HTTPProxy) Start() {
	p.log.Debug().Msg("start proxy")
	p.log.Fatal().Err(p.srv.ListenAndServe()).Msg("failed to start proxy")
}

func (p *HTTPProxy) Shutdown() error {
	return p.srv.Shutdown(context.Background())
}

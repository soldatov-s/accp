package httpproxy

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/httpsrv"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/publisher"
	"github.com/soldatov-s/accp/internal/rabbitmq"
	"github.com/soldatov-s/accp/internal/redis"
	"github.com/soldatov-s/accp/internal/routes"
)

type empty struct{}

type HTTPProxy struct {
	cfg          *Config
	ctx          context.Context
	log          zerolog.Logger
	srv          *httpsrv.Server
	routes       routes.MapRoutes
	introspector introspection.Introspector
	storage      external.Storage
	pub          publisher.Publisher
}

func NewHTTPProxy(ctx context.Context, cfg *Config) (*HTTPProxy, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}

	p := &HTTPProxy{
		ctx:          ctx,
		cfg:          cfg,
		log:          logger.GetPackageLogger(ctx, empty{}),
		introspector: introspection.Get(ctx),
		storage:      redis.Get(ctx),
		pub:          rabbitmq.Get(ctx),
		routes:       make(routes.MapRoutes),
	}

	p.srv = httpsrv.NewHTTPServer(cfg.Listen, p.HydrationID(http.HandlerFunc(p.proxyHandler)))

	if err := p.fillRoutes(p.cfg.Routes, p.routes, nil, ""); err != nil {
		return nil, err
	}

	return p, nil
}

// fillRoutes fill routes map
func (p *HTTPProxy) fillRoutes(rc routes.MapConfig, r routes.MapRoutes, parentParameters *routes.Parameters, parentRoute string) error {
	// Sort config map
	keys := rc.SortKeys()

	for _, configKey := range keys {
		if rc[configKey] == nil {
			return nil
		}

		params := parentParameters
		if params != nil {
			params = params.Merge(rc[configKey].Parameters)
		} else {
			params = rc[configKey].Parameters
			if err := params.Initilize(); err != nil {
				return err
			}
		}

		k := strings.Trim(configKey, "/")
		p.log.Debug().Msgf("parse route \"%s\"", k)
		route, err := r.AddRouteByPath(p.ctx, k, parentRoute+"/"+k, params)
		if err != nil {
			p.log.Warn().Err(err).Msgf("failed add route to map %s", parentRoute+"/"+k)
			return err
		}

		if err := p.fillExcludedRoutes(rc[configKey], parentRoute+"/"+k, params); err != nil {
			return err
		}

		if err := p.fillRoutes(rc[configKey].Routes, route.Routes, params, parentRoute+"/"+k); err != nil {
			return err
		}

		route.Initilize()
	}

	return nil
}

func (p *HTTPProxy) fillExcludedRoutes(rc *routes.Config, parentRoute string, parentParameters *routes.Parameters) error {
	for _, route := range rc.Excluded {
		k := strings.Trim(parentRoute+"/"+route, "/")
		p.log.Debug().Msgf("parse excluded route \"%s\"", k)

		route, err := p.routes.AddExludedRouteByPath(p.ctx, k, k, parentParameters)
		if err != nil {
			p.log.Warn().Err(err).Msgf("failed to add exluded route to map %s", parentRoute+"/"+k)
			return err
		}

		route.Initilize()
	}
	return nil
}

func (p *HTTPProxy) proxyHandler(w http.ResponseWriter, r *http.Request) {
	route := p.routes.FindRouteByHTTPRequest(r)

	// Handle excluded routes
	if route.IsExcluded() {
		route.NotCached(w, r)
		return
	}

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
	} else if *res {
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

func (p *HTTPProxy) HydrationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		next.ServeHTTP(w, r)
	})
}

func (p *HTTPProxy) Start() {
	p.log.Debug().Msg("start proxy")
	p.log.Fatal().Err(p.srv.ListenAndServe()).Msg("failed to start proxy")
}

func (p *HTTPProxy) Shutdown() error {
	return p.srv.Shutdown()
}

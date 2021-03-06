package httpproxy

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/httpsrv"
	"github.com/soldatov-s/accp/internal/httputils"
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

	cfg.SetDefault()

	p := &HTTPProxy{
		ctx:          ctx,
		cfg:          cfg,
		log:          logger.GetPackageLogger(ctx, empty{}),
		introspector: introspection.Get(ctx),
		storage:      redis.Get(ctx),
		pub:          rabbitmq.Get(ctx),
		routes:       make(routes.MapRoutes),
	}

	p.srv = httpsrv.NewHTTPServer(cfg.Listen, p.hydrationID(http.HandlerFunc(p.proxyHandler)))

	if err := p.fillRoutes(p.cfg.Routes, p.routes, nil, ""); err != nil {
		return nil, err
	}

	p.log.Info().Msg("proxy created")

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
			params.SetDefault()
		}

		k := strings.Trim(configKey, "/")
		p.log.Debug().Msgf("parse route \"%s\"", k)
		route, err := r.AddRouteByPath(p.ctx, k, parentRoute+"/"+k, params)
		if err != nil {
			p.log.Warn().Err(err).Msgf("failed add route to map %s", parentRoute+"/"+k)
			return err
		}

		if err := p.fillRoutes(rc[configKey].Routes, route.Routes, params, parentRoute+"/"+k); err != nil {
			return err
		}
	}

	return nil
}

func (p *HTTPProxy) proxyHandler(w http.ResponseWriter, r *http.Request) {
	route := p.routes.FindRouteByHTTPRequest(r)
	if route != nil {
		route.ProxyHandler(w, r)
		return
	}
	p.log.Error().Msgf("route %s not found", r.URL.String())
	http.Error(w, "route "+r.URL.String()+" not found", http.StatusNotFound)
}

func (p *HTTPProxy) hydrationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !p.cfg.RequestID {
			return
		}

		requestID := httputils.GetRequestID(r)
		if requestID == "" {
			newUUID, err := uuid.NewRandom()
			if err != nil {
				p.log.Err(err).Msg("failed to generate requesID")
				return
			}
			r.Header.Del(httputils.RequestIDHeader)
			r.Header.Add(httputils.RequestIDHeader, newUUID.String())
		}

		next.ServeHTTP(w, r)
	})
}

func (p *HTTPProxy) Start() error {
	p.log.Debug().Msg("start proxy")
	return p.srv.ListenAndServe()
}

func (p *HTTPProxy) Shutdown() error {
	return p.srv.Shutdown()
}

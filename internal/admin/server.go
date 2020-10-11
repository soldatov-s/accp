package admin

import (
	"context"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	ctxint "github.com/soldatov-s/accp/internal/ctx"
)

type Config struct {
	Listen string
	Pprof  bool
}

type Server struct {
	cfg *Config
	ctx *ctxint.Context
	log zerolog.Logger
	srv *http.Server
}

func NewAdmin(
	ctx *ctxint.Context,
	cfg *Config,
) (*Server, error) {
	a := &Server{
		ctx: ctx,
		cfg: cfg,
	}

	a.srv = &http.Server{
		Addr:           cfg.Listen,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	r := http.NewServeMux()

	// Alive
	r.HandleFunc("/health/alive", a.aliveHandler)

	// Added metrics
	r.Handle("/metrics", promhttp.Handler())

	// Register pprof
	if a.cfg.Pprof {
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	a.srv.Handler = r

	return a, nil
}

func (a *Server) aliveHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	_, err := w.Write([]byte("{\"result\":\"ok\"}"))
	if err != nil {
		a.log.Err(err).Msg("failed write body")
	}
}

func (a *Server) Start() {
	a.log.Debug().Msg("start admin server")
	a.log.Fatal().Err(a.srv.ListenAndServe()).Msg("failed to start admin server")
}

func (a *Server) Shutdown() error {
	return a.srv.Shutdown(context.Background())
}

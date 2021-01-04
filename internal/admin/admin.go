package admin

import (
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	ctxint "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpsrv"
)

type Admin struct {
	cfg *Config
	ctx *ctxint.Context
	log zerolog.Logger
	srv *httpsrv.Server
}

func NewAdmin(ctx *ctxint.Context, cfg *Config) (*Admin, error) {
	a := &Admin{
		ctx: ctx,
		cfg: cfg,
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

	a.srv = httpsrv.NewHTTPServer(cfg.Listen, r)

	return a, nil
}

func (a *Admin) aliveHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	_, err := w.Write([]byte("{\"result\":\"ok\"}"))
	if err != nil {
		a.log.Err(err).Msg("failed write body")
	}
}

func (a *Admin) Start() {
	a.log.Debug().Msg("start admin server")
	a.log.Fatal().Err(a.srv.ListenAndServe()).Msg("failed to start admin server")
}

func (a *Admin) Shutdown() error {
	return a.srv.Shutdown()
}

package admin

import (
	"context"
	"net/http"
	"net/http/pprof"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/httpsrv"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/metrics"
)

const (
	ReadyEndpoint   = "/health/ready"
	AliveEndpoint   = "/health/alive"
	MetricsEndpoint = "/metrics"
)

type Admin struct {
	cfg *Config
	ctx context.Context
	log zerolog.Logger
	srv *httpsrv.Server
	mux *http.ServeMux

	metricsMutex sync.RWMutex
	metrics      metrics.MapMetricsOptions

	aliveCheckMutex sync.RWMutex
	aliveHandlers   metrics.MapCheckFunc

	readyCheckMutex sync.RWMutex
	readyHandlers   metrics.MapCheckFunc
}

type empty struct{}

func NewAdmin(ctx context.Context, cfg *Config) (*Admin, error) {
	a := &Admin{
		ctx: ctx,
		cfg: cfg,
		log: logger.GetPackageLogger(ctx, empty{}),
		mux: http.NewServeMux(),
	}

	// Alive
	a.mux.HandleFunc(ReadyEndpoint, a.readyCheckHandler)
	a.mux.HandleFunc(AliveEndpoint, a.aliveHandler)

	// Added metrics
	prometheus.MustRegister(prometheus.NewBuildInfoCollector())
	a.mux.Handle("/metrics", a.prometheusMiddleware(promhttp.Handler()))

	// Register pprof
	if a.cfg.Pprof {
		a.mux.HandleFunc("/debug/pprof/", pprof.Index)
		a.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		a.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		a.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		a.mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	a.srv = httpsrv.NewHTTPServer(cfg.Listen, a.mux)

	return a, nil
}

func (a *Admin) errAnswer(w http.ResponseWriter, msg, code string) {
	answ := ErrorAnswer{
		Body: ErrorAnswerBody{
			Code: code,
			BaseAnswer: BaseAnswer{
				StatusCode: http.StatusServiceUnavailable,
				Details:    msg,
			},
		},
	}

	w.WriteHeader(http.StatusFailedDependency)
	err := answ.WriteJSON(w)
	if err != nil {
		a.log.Err(err)
	}
}

func (a *Admin) aliveHandler(w http.ResponseWriter, r *http.Request) {
	a.aliveCheckMutex.RLock()
	for key, f := range a.aliveHandlers {
		result, msg := f()
		if !result {
			a.errAnswer(w, msg, key)
			return
		}
	}
	a.aliveCheckMutex.Unlock()

	answ := ResultAnswer{Body: "ok"}
	err := answ.WriteJSON(w)
	if err != nil {
		a.log.Err(err)
	}
}

func (a *Admin) readyCheckHandler(w http.ResponseWriter, r *http.Request) {
	a.readyCheckMutex.RLock()
	for key, f := range a.readyHandlers {
		result, msg := f()
		if !result {
			a.errAnswer(w, msg, key)
			return
		}
	}
	a.aliveCheckMutex.Unlock()

	answ := ResultAnswer{Body: "ok"}
	err := answ.WriteJSON(w)
	if err != nil {
		a.log.Err(err)
	}
}

func (a *Admin) prometheusMiddleware(handler http.Handler) http.Handler {
	a.metricsMutex.RLock()
	for name, v := range a.metrics {
		v.MetricFunc(v.Metric)
		a.log.Debug().Msg("run metric " + name)
	}
	a.metricsMutex.Unlock()

	return handler
}

// RegisterMetric should register a metric of defined type. Passed
// metricName should be used only as internal identifier. Provider
// should provide instructions for using metricOptions as well as
// cast to appropriate type.
func (a *Admin) RegisterMetric(metricName string, options interface{}) error {
	if metricName == "" {
		a.log.Error().Msg("metric name is empty")
		return ErrEmptyMetricName
	}

	metricOptions, ok := options.(*metrics.MetricOptions)
	if !ok {
		return ErrInvalidMetricOptions(metrics.MetricOptions{})
	}

	a.metricsMutex.Lock()
	a.metrics[metricName] = metricOptions
	prometheus.MustRegister(metricOptions.Metric.(prometheus.Collector))
	a.metricsMutex.Unlock()

	return nil
}

// RegisterReadyCheck should register a function for /health/ready
// endpoint.
func (a *Admin) RegisterReadyCheck(dependencyName string, checkFunc metrics.CheckFunc) error {
	if dependencyName == "" {
		a.log.Error().Msg("dependency name is empty")
		return ErrEmptyDependencyName
	}

	if checkFunc == nil {
		a.log.Error().Msg("checkFunc is null")
		return ErrCheckFuncIsNil
	}

	a.readyCheckMutex.Lock()
	a.readyHandlers[dependencyName] = checkFunc
	a.readyCheckMutex.Unlock()

	return nil
}

// RegisterAliveCheck should register a function for /health/alive
// endpoint.
func (a *Admin) RegisterAliveCheck(dependencyName string, checkFunc metrics.CheckFunc) error {
	if dependencyName == "" {
		a.log.Error().Msg("dependency name is empty")
		return ErrEmptyDependencyName
	}

	if checkFunc == nil {
		a.log.Error().Msg("checkFunc is null")
		return ErrCheckFuncIsNil
	}

	a.aliveCheckMutex.Lock()
	a.aliveHandlers[dependencyName] = checkFunc
	a.aliveCheckMutex.Unlock()

	return nil
}

func (a *Admin) Start(m metrics.MapMetricsOptions, aliveHandlers, readyHandlers metrics.MapCheckFunc) error {
	a.metricsMutex.Lock()
	a.metrics.Fill(m)
	a.metricsMutex.Unlock()

	a.aliveCheckMutex.Lock()
	a.aliveHandlers.Fill(aliveHandlers)
	a.aliveCheckMutex.Unlock()

	a.readyCheckMutex.Lock()
	a.readyHandlers.Fill(readyHandlers)
	a.readyCheckMutex.Unlock()

	a.log.Debug().Msg("start admin server")

	return a.srv.ListenAndServe()
}

func (a *Admin) Shutdown() error {
	return a.srv.Shutdown()
}

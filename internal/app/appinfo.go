package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	accp "github.com/soldatov-s/accp/internal"
	"github.com/soldatov-s/accp/internal/admin"
	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/meta"
	"github.com/soldatov-s/accp/internal/metrics"
	"github.com/soldatov-s/accp/internal/rabbitmq"
	"github.com/soldatov-s/accp/internal/redis"
	"github.com/soldatov-s/accp/internal/utils"
)

// Loop is application loop, exit on SIGTERM
func Loop(ctx context.Context) {
	var closeSignal chan os.Signal
	m := meta.Get(ctx)
	log := logger.Get(ctx).GetLogger(m.Name, nil)

	exit := make(chan struct{})
	closeSignal = make(chan os.Signal, 1)
	signal.Notify(closeSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-closeSignal
		_ = Shutdown(ctx)
		log.Info().Msg("Exit program")
		close(exit)
	}()

	// Exit app if chan is closed
	<-exit
}

// getAllMetrics return all metrics from databases and caches
func getAllMetrics(ctx context.Context) (metrics.MapMetricsOptions, error) {
	m := make(metrics.MapMetricsOptions)
	p := accp.Get(ctx)
	var err error
	p.Range(func(k, v interface{}) bool {
		if v, ok := v.(metrics.IMetrics); ok {
			if _, err = v.GetAllMetrics(m); err != nil {
				return false
			}
		}

		return true
	})

	return m, err
}

// getAllAliveHandlers return all aliveHandlers from databases and caches
// nolint : duplicate
func getAllAliveHandlers(ctx context.Context) (metrics.MapCheckFunc, error) {
	handlers := make(metrics.MapCheckFunc)
	p := accp.Get(ctx)
	var err error
	p.Range(func(k, v interface{}) bool {
		if m, ok := v.(metrics.IMetrics); ok {
			if _, err = m.GetAllAliveHandlers(handlers); err != nil {
				return false
			}
		}

		return true
	})
	return handlers, err
}

// getAllReadyHandlers return all readyHandlers from databases and caches
// nolint : duplicate
func getAllReadyHandlers(ctx context.Context) (metrics.MapCheckFunc, error) {
	handlers := make(metrics.MapCheckFunc)
	p := accp.Get(ctx)
	var err error
	p.Range(func(k, v interface{}) bool {
		if m, ok := v.(metrics.IMetrics); ok {
			if _, err = m.GetAllReadyHandlers(handlers); err != nil {
				return false
			}
		}

		return true
	})
	return handlers, err
}

func StartStatistics(ctx context.Context) error {
	a := admin.Get(ctx)
	if a == nil {
		return nil
	}

	// Collecting all metrics from context
	m, err := getAllMetrics(ctx)
	if err != nil {
		return err
	}

	// Collecting all aliveHandlers from context
	aliveHandlers, err := getAllAliveHandlers(ctx)
	if err != nil {
		return err
	}

	// Collecting all readyHandlers from context
	readyHandlers, err := getAllReadyHandlers(ctx)
	if err != nil {
		return err
	}

	return a.Start(m, aliveHandlers, readyHandlers)
}

func providersOrder() []string {
	return []string{redis.ProviderName, rabbitmq.ProviderName, httpproxy.ProviderName, admin.ProviderName}
}

// Start all providers
func Start(ctx context.Context) error {
	provs := accp.Get(ctx)
	for _, v := range providersOrder() {
		if p, ok := provs.Load(v); ok {
			if err := p.(accp.IProvider).Start(); err != nil {
				return err
			}
		}
	}

	return StartStatistics(ctx)
}

// Shutdown all providers
func Shutdown(ctx context.Context) error {
	provs := accp.Get(ctx)
	for _, v := range utils.ReverseStringSlice(providersOrder()) {
		if p, ok := provs.Load(v); ok {
			if err := p.(accp.IProvider).Shutdown(); err != nil {
				return err
			}
		}
	}

	return nil
}

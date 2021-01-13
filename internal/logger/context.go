package logger

import (
	"context"

	"github.com/rs/zerolog"
	accp "github.com/soldatov-s/accp/internal"
	"github.com/soldatov-s/accp/internal/meta"
)

const (
	DefaultProviderName = "logger"
)

func RegistrateAndInitilize(ctx context.Context, cfg *Config) context.Context {
	ctx = accp.RegistrateByName(ctx, DefaultProviderName, NewLogger())
	Get(ctx).Initialize(cfg)

	return ctx
}

func Get(ctx context.Context) *Logger {
	v := accp.GetByName(ctx, DefaultProviderName)
	if v != nil {
		return v.(*Logger)
	}
	return nil
}

// GetPackageLogger return logger for package
func GetPackageLogger(ctx context.Context, emptyStruct interface{}) zerolog.Logger {
	log := Get(ctx)
	if log == nil {
		accp.RegistrateByName(ctx, DefaultProviderName, NewLogger())
	}

	a := meta.Get(ctx)
	l := log.GetLogger(a.Name, nil)
	return Initialize(l, emptyStruct)
}

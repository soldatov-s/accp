package logger

import (
	"context"

	"github.com/rs/zerolog"
	accp "github.com/soldatov-s/accp/internal"
	"github.com/soldatov-s/accp/internal/meta"
)

const (
	defaultAppName      = "accp"
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
		log = NewLogger()
		accp.RegistrateByName(ctx, DefaultProviderName, log)
	}

	a := meta.Get(ctx)
	var l *zerolog.Logger
	if a != nil {
		l = log.GetLogger(a.Name, nil)
	} else {
		l = log.GetLogger(defaultAppName, nil)
	}
	return Initialize(l, emptyStruct)
}

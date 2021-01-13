package httpproxy

import (
	"context"

	accp "github.com/soldatov-s/accp/internal"
)

const (
	ProviderName = "httpproxy"
)

func Registrate(ctx context.Context, cfg *Config) (context.Context, error) {
	i, err := NewHTTPProxy(ctx, cfg)
	if err != nil {
		return nil, err
	}

	ctx = accp.RegistrateByName(ctx, ProviderName, i)
	return ctx, nil
}

func Get(ctx context.Context) *HTTPProxy {
	return accp.GetByName(ctx, ProviderName).(*HTTPProxy)
}

package redis

import (
	"context"

	accp "github.com/soldatov-s/accp/internal"
)

const (
	ProviderName = "redisclient"
)

func Registrate(ctx context.Context, cfg *Config) (context.Context, error) {
	if cfg == nil {
		return ctx, nil
	}

	i, err := NewClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	ctx = accp.RegistrateByName(ctx, ProviderName, i)
	return ctx, nil
}

func Get(ctx context.Context) *Client {
	return accp.GetByName(ctx, ProviderName).(*Client)
}

package admin

import (
	"context"

	accp "github.com/soldatov-s/accp/internal"
)

const (
	ProviderName = "admin"
)

func Registrate(ctx context.Context, cfg *Config) (context.Context, error) {
	i, err := NewAdmin(ctx, cfg)
	if err != nil {
		return nil, err
	}

	ctx = accp.RegistrateByName(ctx, ProviderName, i)
	return ctx, nil
}

func Get(ctx context.Context) *Admin {
	return accp.GetByName(ctx, ProviderName).(*Admin)
}

package introspection

import (
	"context"

	accp "github.com/soldatov-s/accp/internal"
)

const (
	DefaultProviderName = "introspector"
)

func Registrate(ctx context.Context, cfg *Config) (context.Context, error) {
	if Get(ctx) != nil {
		return ctx, nil
	}

	i, err := NewIntrospector(ctx, cfg)
	if err != nil {
		return nil, err
	}

	ctx = accp.RegistrateByName(ctx, DefaultProviderName, i)
	return ctx, nil
}

func Get(ctx context.Context) *Introspect {
	if v, ok := accp.GetByName(ctx, DefaultProviderName).(*Introspect); ok {
		return v
	}
	return nil
}

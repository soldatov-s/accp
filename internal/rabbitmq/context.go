package rabbitmq

import (
	"context"

	accp "github.com/soldatov-s/accp/internal"
)

const (
	ProviderName = "rabbitmq"
)

func Registrate(ctx context.Context, cfg *Config) (context.Context, error) {
	i, err := NewPublisher(ctx, cfg)
	if err != nil {
		return nil, err
	}

	ctx = accp.RegistrateByName(ctx, ProviderName, i)
	return ctx, nil
}

func Get(ctx context.Context) *Publish {
	return accp.GetByName(ctx, ProviderName).(*Publish)
}

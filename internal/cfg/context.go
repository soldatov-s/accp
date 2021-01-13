package cfg

import (
	"context"

	accp "github.com/soldatov-s/accp/internal"
	"github.com/spf13/cobra"
)

const (
	DefaultProviderName = "config"
)

func RegistrateAndParse(ctx context.Context, command *cobra.Command) (context.Context, error) {
	c, err := NewConfig(command)
	if err != nil {
		return nil, err
	}

	ctx = accp.RegistrateByName(ctx, DefaultProviderName, c)

	return ctx, nil
}

func Get(ctx context.Context) *Config {
	return accp.GetByName(ctx, DefaultProviderName).(*Config)
}

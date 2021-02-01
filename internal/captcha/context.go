package captcha

import (
	"context"

	accp "github.com/soldatov-s/accp/internal"
)

const (
	DefaultProviderName = "captcha"
)

func Registrate(ctx context.Context, cfg *Config) (context.Context, error) {
	if Get(ctx) != nil {
		return ctx, nil
	}

	i, err := NewGoogleCaptcha(ctx, cfg)
	if err != nil {
		return nil, err
	}

	ctx = accp.RegistrateByName(ctx, DefaultProviderName, i)
	return ctx, nil
}

func Get(ctx context.Context) *GoogleCaptcha {
	if v, ok := accp.GetByName(ctx, DefaultProviderName).(*GoogleCaptcha); ok {
		return v
	}
	return nil
}

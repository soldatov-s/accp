package meta

import (
	"context"

	accp "github.com/soldatov-s/accp/internal"
)

const (
	AppItem accp.Item = "app"
)

func Registrate(ctx context.Context) context.Context {
	v := Get(ctx)
	if v != nil {
		return ctx
	}

	return context.WithValue(ctx, AppItem, NewApplicationInfo())
}

func Get(ctx context.Context) *ApplicationInfo {
	if v, ok := ctx.Value(AppItem).(*ApplicationInfo); ok {
		return v
	}
	return nil
}

func SetAppInfo(ctx context.Context, name, builded, hash, version, description string) context.Context {
	ctx = Registrate(ctx)
	a := Get(ctx)
	a.Name = name
	a.Builded = builded
	a.Hash = hash
	a.Version = version
	a.Description = description

	return ctx
}

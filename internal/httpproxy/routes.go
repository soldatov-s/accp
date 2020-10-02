package httpproxy

import (
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cache/memory"
	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpclient"
)

type Route struct {
	Routes map[string]*Route
	Cache  struct {
		Mem      *memory.Cache
		External *external.Cache
	}
	Pool *httpclient.Pool
}

func (r *Route) Initilize(ctx *context.Context, parameters *RouteParameters, externalStorage external.ExternalStorage) error {
	var err error
	r.Cache.Mem, err = memory.NewCache(ctx, parameters.Cache.Memory)
	if err != nil {
		return err
	}

	r.Cache.External, err = external.NewCache(ctx, parameters.Cache.External, externalStorage)
	if err != nil {
		return err
	}

	r.Pool = httpclient.NewPool(parameters.Pool.Size, parameters.Pool.Timeout)

	return nil
}

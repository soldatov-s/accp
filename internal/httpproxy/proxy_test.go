package httpproxy_test

import (
	"net/http"
	"testing"

	"github.com/soldatov-s/accp/internal/cfg"
	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/stretchr/testify/require"
)

func initProxy(t *testing.T) *httpproxy.HTTPProxy {
	c, err := cfg.LoadTestConfig()
	require.Nil(t, err)

	ctx := context.NewContext()
	ctx.InitilizeLogger(c.Logger)
	i, err := introspection.NewIntrospector(ctx, c.Introspector)
	require.Nil(t, err)

	p, err := httpproxy.NewHTTPProxy(ctx, c.Proxy, i, nil, nil)
	require.Nil(t, err)

	return p
}

func TestNewHTTPProxy(t *testing.T) {
	initProxy(t)
}

func TestFindRoute(t *testing.T) {
	p := initProxy(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	route := p.FindRoute(r)
	require.NotNil(t, route)

	t.Logf("route value %+v", route)
}

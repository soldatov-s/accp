package httpproxy_test

import (
	"testing"

	"github.com/soldatov-s/accp/internal/cfg"
	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPProxy(t *testing.T) {
	c, err := cfg.LoadTestConfig()
	require.Nil(t, err)

	ctx := context.NewContext()
	ctx.InitilizeLogger(c.Logger)
	i, err := introspection.NewIntrospector(ctx, c.Introspector)
	require.Nil(t, err)

	_, err = httpproxy.NewHTTPProxy(ctx, c.Proxy, i, nil, nil)
	require.Nil(t, err)
}

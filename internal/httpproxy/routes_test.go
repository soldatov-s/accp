package httpproxy_test

import (
	"net/http"
	"testing"
	"time"

	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/introspection"
	testhelpers "github.com/soldatov-s/accp/x/test_helpers"
	testProxyHelpers "github.com/soldatov-s/accp/x/test_helpers/proxy"
	"github.com/stretchr/testify/require"
)

func initRoute(t *testing.T) *httpproxy.Route {
	err := testhelpers.LoadTestYAML()
	require.Nil(t, err)

	lc, err := testhelpers.LoadTestConfigLogger()
	require.Nil(t, err)

	ctx := context.NewContext()
	ctx.InitilizeLogger(lc)

	ic, err := testhelpers.LoadTestConfigIntrospector()
	require.Nil(t, err)

	i, err := introspection.NewIntrospector(ctx, ic)
	require.Nil(t, err)

	pc, err := testhelpers.LoadTestConfigProxy()
	require.Nil(t, err)

	p, err := httpproxy.NewHTTPProxy(ctx, pc, i, nil, nil)
	require.Nil(t, err)

	return p.FindRouteByPath("/api/v1/users")
}
func TestRoute_GetLimitsFromRequest(t *testing.T) {
	route := initRoute(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	r.Header.Add("Authorization", "bearer "+testProxyHelpers.TestToken)

	limits := route.GetLimitsFromRequest(r)

	require.Equal(t, limits["token"], testProxyHelpers.TestToken)
}

func TestRoute_CheckLimits(t *testing.T) {
	route := initRoute(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	r.Header.Add("Authorization", "bearer "+testProxyHelpers.TestToken)

	res, err := route.CheckLimits(r)
	require.Nil(t, err)
	require.Equal(t, *res, true)

	res, err = route.CheckLimits(r)
	require.Nil(t, err)
	require.Equal(t, *res, false)

	time.Sleep(3 * time.Second)

	res, err = route.CheckLimits(r)
	require.Nil(t, err)
	require.Equal(t, *res, true)
}

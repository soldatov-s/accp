package routes

import (
	"net/http"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/cache"
	"github.com/soldatov-s/accp/internal/cache/memory"
	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/routes/refresh"
	testctxhelper "github.com/soldatov-s/accp/x/test_helpers/ctx"
	testProxyHelpers "github.com/soldatov-s/accp/x/test_helpers/proxy"
	"github.com/stretchr/testify/require"
)

func initRouteParameters(t *testing.T) *Parameters {
	var err error
	parameters := &Parameters{
		DSN: "http://localhost:9090",
		Cache: &cache.Config{
			Memory: &memory.CacheConfig{
				TTL:    2 * time.Second,
				TTLErr: 1 * time.Second,
			},
		},
		Pool: &httpclient.PoolConfig{
			Size:    20,
			Timeout: 10 * time.Second,
		},
		Refresh: &refresh.Config{
			Count: 3,
			Time:  10 * time.Second,
		},
		Introspect: true,
	}

	err = parameters.Initilize()
	require.Nil(t, err)

	parameters.Limits["token"].Counter = 1
	parameters.Limits["token"].PT = 3 * time.Second

	return parameters
}

func initRoute(t *testing.T) *Route {
	var err error
	r := &Route{
		Routes: make(map[string]*Route),
	}

	ctx := testctxhelper.InitTestCtx(t)

	parameters := initRouteParameters(t)

	ic := &introspection.Config{
		DSN:            "http://localhost:8001",
		Endpoint:       "/oauth2/introspect",
		ContentType:    "application/x-www-form-urlencoded",
		Method:         "POST",
		ValidMarker:    `"active":true`,
		BodyTemplate:   `token_type_hint=access_token&token={{.Token}}`,
		CookieName:     []string{"access-token"},
		QueryParamName: []string{"access_token"},
		PoolSize:       50,
		PoolTimeout:    10 * time.Second,
	}

	i, err := introspection.NewIntrospector(ctx, ic)
	require.Nil(t, err)

	err = r.Initilize(ctx, "/api/v1/users", parameters, nil, nil, i)
	require.Nil(t, err)

	return r
}
func TestRoute_GetLimitsFromRequest(t *testing.T) {
	route := initRoute(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	r.Header.Add("Authorization", "bearer "+testProxyHelpers.TestToken)

	l := route.getLimitsFromRequest(r)

	require.Equal(t, l["token"], testProxyHelpers.TestToken)
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

func TestIntrospection(t *testing.T) {
	server := testProxyHelpers.FakeIntrospectorService(t, "localhost:8001")
	server.Start()
	defer server.Close()

	route := initRoute(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	r.Header.Add("Authorization", "bearer "+testProxyHelpers.TestToken)

	err = route.HydrationIntrospect(r)
	require.Nil(t, err)

	r.Header.Set("Authorization", "bearer "+testProxyHelpers.BadToken)
	err = route.HydrationIntrospect(r)
	require.NotNil(t, err)
}

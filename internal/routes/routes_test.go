package routes

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/cache"
	"github.com/soldatov-s/accp/internal/cache/memory"
	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/limits"
	"github.com/soldatov-s/accp/internal/routes/refresh"
	testProxyHelpers "github.com/soldatov-s/accp/x/test_helpers/proxy"
	"github.com/stretchr/testify/require"
)

func initRouteParameters(t *testing.T) *Parameters {
	var err error
	parameters := &Parameters{
		DSN: "http://localhost:9090",
		Cache: &cache.Config{
			Memory: &memory.Config{
				TTL:    2 * time.Second,
				TTLErr: 1 * time.Second,
			},
		},
		Pool: &httpclient.Config{
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

// nolint : unused
func initRoute(t *testing.T) *Route {
	ctx := context.Background()
	params := initRouteParameters(t)
	r := NewRoute(ctx, "/api/v1/users", params)

	// ic := &introspection.Config{
	// 	DSN:            "http://localhost:8001",
	// 	Endpoint:       "/oauth2/introspect",
	// 	ContentType:    "application/x-www-form-urlencoded",
	// 	Method:         "POST",
	// 	ValidMarker:    `"active":true`,
	// 	BodyTemplate:   `token_type_hint=access_token&token={{.Token}}`,
	// 	CookieName:     []string{"access-token"},
	// 	QueryParamName: []string{"access_token"},
	// 	Pool: &httpclient.PoolConfig{
	// 		Size:    50,
	// 		Timeout: 10 * time.Second,
	// 	},
	// }

	// i, err := introspection.NewIntrospector(ctx, ic)
	// require.Nil(t, err)

	r.Initilize()

	return r
}
func TestRoute_GetLimitsFromRequest(t *testing.T) {
	route := initRoute(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	r.Header.Add("Authorization", "bearer "+testProxyHelpers.TestToken)

	l := limits.NewLimitedParamsOfRequest(route.Parameters.Limits, r)

	require.Equal(t, l["token"], testProxyHelpers.TestToken)
}

func TestRoute_CheckLimits(t *testing.T) {
	route := initRoute(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	r.Header.Add("Authorization", "bearer "+testProxyHelpers.TestToken)

	res, err := route.CheckLimits(r)
	require.Nil(t, err)
	require.Equal(t, *res, false)

	res, err = route.CheckLimits(r)
	require.Nil(t, err)
	require.Equal(t, *res, true)

	time.Sleep(3 * time.Second)

	res, err = route.CheckLimits(r)
	require.Nil(t, err)
	require.Equal(t, *res, false)
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

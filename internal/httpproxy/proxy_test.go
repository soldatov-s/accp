package httpproxy_test

import (
	"net/http"
	"testing"

	"github.com/soldatov-s/accp/internal/cfg"
	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/stretchr/testify/assert"
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

func TestFindRouteByPath(t *testing.T) {
	p := initProxy(t)

	route := p.FindRouteByPath("/api/v1/users")
	require.NotNil(t, route)

	t.Logf("route value %+v", route)
}

func TestFindRouteByHTTPRequest(t *testing.T) {
	p := initProxy(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	route := p.FindRouteByHTTPRequest(r)
	require.NotNil(t, route)

	t.Logf("route value %+v", route)
}

func TestFindExcluededRouteByHTTPRequest(t *testing.T) {
	p := initProxy(t)

	r, err := http.NewRequest("GET", "/api/v2/", nil)
	require.Nil(t, err)

	route := p.FindExcludedRouteByHTTPRequest(r)
	require.NotNil(t, route)

	t.Logf("route value %+v", route)
}

func TestHTTPProxy_HydrationID(t *testing.T) {
	p := initProxy(t)

	tests := []struct {
		name                string
		testHeaderValue     string
		expectedHeaderValue string
	}{
		{
			name:                "x-request-id exist",
			testHeaderValue:     "abc123",
			expectedHeaderValue: "abc123",
		},
		{
			name:            "x-request-id not exist",
			testHeaderValue: "",
		},
	}
	for _, tt := range tests {
		var headerValue string
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest("GET", "/api/v1/users", nil)
			require.Nil(t, err)
			r.Header.Add("x-request-id", tt.testHeaderValue)

			if tt.testHeaderValue != "" {
				p.HydrationID(r)
				headerValue = r.Header.Get("x-request-id")
				assert.Equal(t, headerValue, tt.expectedHeaderValue)
			} else {
				p.HydrationID(r)
				headerValue = r.Header.Get("x-request-id")
				assert.NotEqual(t, headerValue, "")
			}
			t.Logf("x-request-id is %s", headerValue)
		})
	}
}

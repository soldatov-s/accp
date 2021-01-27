package routes

import (
	"context"
	"net/http"
	"testing"

	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/meta"
	"github.com/stretchr/testify/require"
)

func initApp(ctx context.Context) context.Context {
	return meta.SetAppInfo(ctx, "accp", "", "", "", "test")
}

func initLogger(ctx context.Context) context.Context {
	// Registrate logger
	logCfg := &logger.Config{
		Level:           logger.LoggerLevelDebug,
		NoColoredOutput: true,
		WithTrace:       false,
	}
	ctx = logger.RegistrateAndInitilize(ctx, logCfg)

	return ctx
}

func TestAddRouteByPath(t *testing.T) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)

	rm := make(MapRoutes)
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "adding the route /api/v1/users",
			testFunc: func() {
				_, err := rm.AddRouteByPath(ctx, "/api/v1/users", "/api/v1/users", &Parameters{})
				require.Nil(t, err)

				require.Contains(t, rm, "api")
				require.Contains(t, rm["api"].Routes, "v1")
				require.Contains(t, rm["api"].Routes["v1"].Routes, "users")
			},
		},
		{
			name: "adding the route /api/v1/testers",
			testFunc: func() {
				_, err := rm.AddRouteByPath(ctx, "/api/v1/testers", "/api/v1/testers", &Parameters{})
				require.Nil(t, err)

				require.Contains(t, rm, "api")
				require.Contains(t, rm["api"].Routes, "v1")
				require.Contains(t, rm["api"].Routes["v1"].Routes, "testers")
			},
		},
		{
			name: "adding the duplicated route /api/v1/testers",
			testFunc: func() {
				_, err := rm.AddRouteByPath(ctx, "/api/v1/testers", "/api/v1/testers", &Parameters{})
				require.NotNil(t, err)

				require.Contains(t, rm, "api")
				require.Contains(t, rm["api"].Routes, "v1")
				require.Contains(t, rm["api"].Routes["v1"].Routes, "testers")
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc()
		})
	}
}

func TestFindRouteByPath(t *testing.T) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)

	rm := make(MapRoutes)

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "added and find route /api/v1",
			testFunc: func() {
				_, err := rm.AddRouteByPath(ctx, "/api/v1", "/api/v1", &Parameters{})
				require.Nil(t, err)

				r := rm.FindRouteByPath("/api/v1")
				require.NotNil(t, r)

				t.Logf("route: %+v", r.route)
			},
		},
		{
			name: "added and find route /api/v1/users",
			testFunc: func() {
				_, err := rm.AddRouteByPath(ctx, "/api/v1/users", "/api/v1/users", &Parameters{})
				require.Nil(t, err)

				r := rm.FindRouteByPath("/api/v1/users")
				require.NotNil(t, r)

				t.Logf("route: %+v", r.route)
			},
		},
		{
			name: "added and find route /api/v1/testers",
			testFunc: func() {
				_, err := rm.AddRouteByPath(ctx, "/api/v1/testers", "/api/v1/testers", &Parameters{})
				require.Nil(t, err)

				r := rm.FindRouteByPath("/api/v1/testers")
				require.NotNil(t, r)

				t.Logf("route: %+v", r.route)
			},
		},
		{
			name: "find route /api/v1/testers",
			testFunc: func() {
				r := rm.FindRouteByPath("/api/v1/admins")
				require.NotNil(t, r)

				t.Logf("route: %+v", r.route)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc()
		})
	}
}

func TestFindRouteByHTTPRequest(t *testing.T) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)

	rm := make(MapRoutes)

	_, err := rm.AddRouteByPath(ctx, "/api/v1/users", "/api/v1/users", &Parameters{})
	require.Nil(t, err)

	req, err := http.NewRequest(http.MethodGet, "http://localhost:10000"+"/api/v1/users", nil)
	require.Nil(t, err)

	r := rm.FindRouteByHTTPRequest(req)
	require.NotNil(t, r)

	t.Logf("route: %+v", r)

	_, err = rm.AddRouteByPath(ctx, "/api/v1/testers", "/api/v1/testers", &Parameters{})
	require.Nil(t, err)

	req, err = http.NewRequest(http.MethodGet, "http://localhost:10000"+"/api/v1/testers", nil)
	require.Nil(t, err)

	r = rm.FindRouteByHTTPRequest(req)
	require.NotNil(t, r)

	t.Logf("route: %+v", r)
}

package httpproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/cache"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cache/memory"
	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/httputils"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/meta"
	"github.com/soldatov-s/accp/internal/rabbitmq"
	"github.com/soldatov-s/accp/internal/redis"
	"github.com/soldatov-s/accp/internal/routes"
	"github.com/soldatov-s/accp/internal/routes/refresh"
	"github.com/soldatov-s/accp/x/dockertest"
	testproxyhelpers "github.com/soldatov-s/accp/x/test_helpers/proxy"
	"github.com/soldatov-s/accp/x/test_helpers/resilience"
	"github.com/stretchr/testify/require"
)

const (
	testExchangeName = "accp.test.events"
	testPoolSize     = 10
	testPoolTimeout  = 5 * time.Second
	testLimitCounter = 1
	testLimitPT      = 3 * time.Second
	testRouteKey     = "ACCP_TEST"
	testRequestID    = "123456789"
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

func initPublisherConfig() *rabbitmq.Config {
	return &rabbitmq.Config{
		ExchangeName: testExchangeName,
	}
}

// nolint : gocritic
func initPublish(ctx context.Context, t *testing.T) (context.Context, string) {
	dsn, err := dockertest.RunRabbitMQ()
	require.Nil(t, err)

	cfg := initPublisherConfig()
	cfg.DSN = dsn

	t.Logf("connecting to rabbitmq: %s", dsn)
	err = resilience.Retry(
		t,
		time.Second*5,
		time.Minute*5,
		func() (err error) {
			ctx, err = rabbitmq.Registrate(ctx, cfg)
			require.Nil(t, err)
			c := rabbitmq.Get(ctx)
			return c.Start()
		},
	)
	require.Nil(t, err)

	t.Logf("connected to rabbitmq: %s", dsn)

	return ctx, dsn
}

func initExternalCacheConfig() *redis.Config {
	return &redis.Config{
		MinIdleConnections:    10,
		MaxOpenedConnections:  30,
		MaxConnectionLifetime: 30 * time.Second,
	}
}

func initExternalCache(ctx context.Context, t *testing.T) context.Context {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)

	cfg := initExternalCacheConfig()
	cfg.DSN = dsn

	t.Logf("connecting to redis: %s", dsn)
	err = resilience.Retry(
		t,
		time.Second*5,
		time.Minute*5,
		func() (err error) {
			ctx, err = redis.Registrate(ctx, cfg)
			require.Nil(t, err)
			c := redis.Get(ctx)
			return c.Start()
		},
	)
	require.Nil(t, err)

	t.Logf("connected to redis: %s", dsn)

	return ctx
}

func initPool() *httpclient.Config {
	return &httpclient.Config{
		Size:    testPoolSize,
		Timeout: testPoolTimeout,
	}
}

func initIntrospectorConfig() *introspection.Config {
	return &introspection.Config{
		DSN:            testproxyhelpers.DefaultFakeIntrospectorURL,
		Endpoint:       testproxyhelpers.DefaultFakeIntrospectorEndpoint,
		ContentType:    testproxyhelpers.DefaultFakeIntrospectorContentType,
		Method:         testproxyhelpers.DefaultFakeIntrospectorMethod,
		ValidMarker:    testproxyhelpers.DefaultFakeIntrospectorValidMarker,
		BodyTemplate:   testproxyhelpers.DefaultFakeIntrospectorBodyTemplate,
		CookieName:     testproxyhelpers.DefaultFakeIntrospectorCookiesName(),
		QueryParamName: testproxyhelpers.DefaultFakeIntrospectorQueryParamsName(),
		Pool:           initPool(),
	}
}

func initIntrospector(ctx context.Context, t *testing.T) context.Context {
	cfg := initIntrospectorConfig()
	ctx, err := introspection.Registrate(ctx, cfg)
	require.Nil(t, err)
	return ctx
}

func initParameters() *routes.Parameters {
	parameters := &routes.Parameters{
		DSN: testproxyhelpers.DefaultFakeServiceURL,
		Cache: &cache.Config{
			Memory: &memory.Config{
				TTL:    5 * time.Second,
				TTLErr: 3 * time.Second,
			},
			External: &external.Config{
				TTL:       10 * time.Second,
				TTLErr:    5 * time.Second,
				KeyPrefix: "accp_test_",
			},
		},
		Pool: &httpclient.Config{
			Size:    20,
			Timeout: 10 * time.Second,
		},
		Refresh: &refresh.Config{
			MaxCount: 15,
			Time:     3 * time.Second,
		},
		NotIntrospect: false,
		RouteKey:      testRouteKey,
	}

	parameters.SetDefault()
	parameters.Limits.SetDefault()

	// Set limit config for token
	parameters.Limits["token"].MaxCounter = testLimitCounter
	parameters.Limits["token"].TTL = testLimitPT

	return parameters
}

// nolint : deadcode
func initRoute(ctx context.Context, t *testing.T) *routes.Route {
	params := initParameters()
	r := routes.NewRoute(ctx, "/api/v1/users", params)

	return r
}

func initConfig() *Config {
	params := initParameters()
	c := &Config{
		Listen:   defaultListen,
		Routes:   make(routes.MapConfig),
		Excluded: make(routes.MapConfig),
	}
	c.Routes["/api/v1/users"] = &routes.Config{
		Parameters: params,
	}

	c.RequestID = true

	return c
}

func initHTTPProxy(t *testing.T) *HTTPProxy {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	ctx, _ = initPublish(ctx, t)
	ctx = initExternalCache(ctx, t)
	ctx = initIntrospector(ctx, t)

	cfg := initConfig()
	p, err := NewHTTPProxy(ctx, cfg)
	require.Nil(t, err)

	return p
}

func TestNewHTTPProxy(t *testing.T) {
	defer dockertest.KillAllDockers()
	_ = initHTTPProxy(t)
}

func TestHydrationID(t *testing.T) {
	defer dockertest.KillAllDockers()
	p := initHTTPProxy(t)

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test empty requestID",
			testFunc: func() {
				nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestID := httputils.GetRequestID(r)
					require.NotEmpty(t, requestID)

					t.Logf("request id: %s", requestID)
				})

				handlerToTest := p.hydrationID(nextHandler)
				req := httptest.NewRequest("GET", testproxyhelpers.DefaultFakeServiceURL, nil)
				handlerToTest.ServeHTTP(httptest.NewRecorder(), req)
			},
		},
		{
			name: "test not empty requestID",
			testFunc: func() {
				nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestID := httputils.GetRequestID(r)
					require.NotEmpty(t, requestID)
					require.Equal(t, testRequestID, requestID)

					t.Logf("request id: %s", requestID)
				})

				handlerToTest := p.hydrationID(nextHandler)
				req := httptest.NewRequest("GET", testproxyhelpers.DefaultFakeServiceURL, nil)
				req.Header.Add(httputils.RequestIDHeader, testRequestID)
				handlerToTest.ServeHTTP(httptest.NewRecorder(), req)
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

func TestFillRoutes(t *testing.T) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)

	p := &HTTPProxy{
		ctx: ctx,
		log: logger.GetPackageLogger(ctx, empty{}),
	}

	routesCfg := make(routes.MapConfig)
	params := initParameters()
	routesCfg["/api/v1/users"] = &routes.Config{
		Parameters: params,
	}

	mapRoutes := make(routes.MapRoutes)

	err := p.fillRoutes(routesCfg, mapRoutes, nil, "")
	require.Nil(t, err)
	require.Contains(t, mapRoutes, "api")
	require.Contains(t, mapRoutes["api"].Routes, "v1")
	require.Contains(t, mapRoutes["api"].Routes["v1"].Routes, "users")
}

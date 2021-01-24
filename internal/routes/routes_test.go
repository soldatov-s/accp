package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/cache"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cache/memory"
	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/httputils"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/rabbitmq"
	"github.com/soldatov-s/accp/internal/redis"
	rrdata "github.com/soldatov-s/accp/internal/request_response_data"
	"github.com/soldatov-s/accp/internal/routes/refresh"
	"github.com/soldatov-s/accp/x/dockertest"
	testproxyhelpers "github.com/soldatov-s/accp/x/test_helpers/proxy"
	rabbitMQConsumer "github.com/soldatov-s/accp/x/test_helpers/rabbitmq"
	"github.com/soldatov-s/accp/x/test_helpers/resilience"
	"github.com/stretchr/testify/require"
)

const (
	testExchangeName = "accp.test.events"
	testPoolSize     = 10
	testPoolTimeout  = 5 * time.Second
	testLimitCounter = 1
	testLimitPT      = 3 * time.Second
	testQueue        = "test.queue"
	testMessage      = "test message"
	testConsumer     = "test_consumer"
	testRouteKey     = "ACCP_TEST"
)

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

func initParameters() *Parameters {
	parameters := &Parameters{
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
		Introspect: false,
		RouteKey:   testRouteKey,
	}

	parameters.SetDefault()

	// Set limit config for token
	parameters.Limits["token"].Counter = testLimitCounter
	parameters.Limits["token"].PT = testLimitPT

	return parameters
}

// nolint : deadcode
func initRoute(t *testing.T) *Route {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	ctx, _ = initPublish(ctx, t)
	ctx = initExternalCache(ctx, t)
	ctx = initIntrospector(ctx, t)

	params := initParameters()
	r := NewRoute(ctx, "/api/v1/users", params)

	r.Initilize()

	return r
}

func TestNewRoute(t *testing.T) {
	defer dockertest.KillAllDockers()
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test without intropspector, external cache, publisher",
			testFunc: func() {
				params := initParameters()
				r := NewRoute(ctx, "/api/v1/users", params)
				require.NotNil(t, r)
			},
		},
		{
			name: "test with all external services",
			testFunc: func() {
				ctx, _ = initPublish(ctx, t)
				ctx = initExternalCache(ctx, t)
				ctx = initIntrospector(ctx, t)

				params := initParameters()
				r := NewRoute(ctx, "/api/v1/users", params)
				require.NotNil(t, r)
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

func TestInitilize(t *testing.T) {
	defer dockertest.KillAllDockers()
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test without intropspector, external cache, publisher",
			testFunc: func() {
				params := initParameters()
				r := NewRoute(ctx, "/api/v1/users", params)
				require.NotNil(t, r)

				r.Initilize()
			},
		},
		{
			name: "test with all external services",
			testFunc: func() {
				ctx, _ = initPublish(ctx, t)
				ctx = initExternalCache(ctx, t)
				ctx = initIntrospector(ctx, t)

				params := initParameters()
				r := NewRoute(ctx, "/api/v1/users", params)
				require.NotNil(t, r)

				r.Initilize()
				require.NotNil(t, r.Cache)
				require.NotNil(t, r.Limits)
				require.NotNil(t, r.RefreshTimer)
			},
		},
		{
			name: "test exluded route with all external services",
			testFunc: func() {
				params := initParameters()
				r := NewRoute(ctx, "/api/v1/users", params)
				require.NotNil(t, r)

				r.excluded = true

				r.Initilize()
				require.Nil(t, r.Cache)
				require.Nil(t, r.Limits)
				require.Nil(t, r.RefreshTimer)
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

func TestIsExcluded(t *testing.T) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)

	params := initParameters()
	r := NewRoute(ctx, "/api/v1/users", params)
	require.NotNil(t, r)

	r.excluded = true
	result := r.IsExcluded()
	require.True(t, result)
}

// nolint : dupl
func TestCheckLimitsWithoutExternalCache(t *testing.T) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)

	params := initParameters()
	r := NewRoute(ctx, "/api/v1/users", params)
	require.NotNil(t, r)
	r.Initilize()

	req, err := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
	require.Nil(t, err)
	req.Header.Add("Authorization", "bearer "+testproxyhelpers.TestToken)

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "first request",
			testFunc: func() {
				res, err := r.CheckLimits(req)
				require.Nil(t, err)
				require.False(t, *res)
			},
		},
		{
			name: "second request, overflow",
			testFunc: func() {
				res, err := r.CheckLimits(req)
				require.Nil(t, err)
				require.True(t, *res)
			},
		},
		{
			name: "request after timeout, limit removed",
			testFunc: func() {
				time.Sleep(testLimitPT + 1*time.Second)

				res, err := r.CheckLimits(req)
				require.Nil(t, err)
				require.False(t, *res)
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

// nolint : dupl
func TestCheckLimitsWithExternalCache(t *testing.T) {
	defer dockertest.KillAllDockers()
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	ctx = initExternalCache(ctx, t)

	params := initParameters()
	r := NewRoute(ctx, "/api/v1/users", params)
	require.NotNil(t, r)
	r.Initilize()

	req, err := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
	require.Nil(t, err)
	req.Header.Add("Authorization", "bearer "+testproxyhelpers.TestToken)

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "first request",
			testFunc: func() {
				res, err := r.CheckLimits(req)
				require.Nil(t, err)
				require.False(t, *res)
			},
		},
		{
			name: "second request, overflow",
			testFunc: func() {
				res, err := r.CheckLimits(req)
				require.Nil(t, err)
				require.True(t, *res)
			},
		},
		{
			name: "request after timeout, limit removed",
			testFunc: func() {
				time.Sleep(testLimitPT + 1*time.Second)

				res, err := r.CheckLimits(req)
				require.Nil(t, err)
				require.False(t, *res)
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

func TestHydrationIntrospect(t *testing.T) {
	server := testproxyhelpers.FakeIntrospectorService(t, testproxyhelpers.DefaultFakeIntrospectorHost)
	server.Start()
	defer server.Close()

	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	ctx = initIntrospector(ctx, t)

	params := initParameters()
	params.Introspect = true
	r := NewRoute(ctx, "/api/v1/users", params)
	require.NotNil(t, r)
	r.Initilize()

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test good token",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
				require.Nil(t, err)

				req.Header.Add("Authorization", "bearer "+testproxyhelpers.TestToken)

				err = r.HydrationIntrospect(req)
				require.Nil(t, err)

				header := req.Header.Get(hydrationIntrospectHeader)
				require.Empty(t, header)
			},
		},
		{
			name: "test good token with hydration plaintext",
			testFunc: func() {
				r.Parameters.IntrospectHydration = hydrationIntrospectPlainText
				req, err := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
				require.Nil(t, err)

				req.Header.Add("Authorization", "bearer "+testproxyhelpers.TestToken)

				err = r.HydrationIntrospect(req)
				require.Nil(t, err)

				header := req.Header.Get(hydrationIntrospectHeader)
				require.Equal(t, `{\"active\":true, \"subject\":\"1\", \"token_type\":\"access_token\"}`, header)
			},
		},
		{
			name: "test good token with hydration base64",
			testFunc: func() {
				r.Parameters.IntrospectHydration = hydrationIntrospectBase64
				req, err := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
				require.Nil(t, err)

				req.Header.Add("Authorization", "bearer "+testproxyhelpers.TestToken)

				err = r.HydrationIntrospect(req)
				require.Nil(t, err)

				header := req.Header.Get(hydrationIntrospectHeader)
				require.Equal(t, `eyJhY3RpdmUiOnRydWUsICJzdWJqZWN0IjoiMSIsICJ0b2tlbl90eXBlIjoiYWNjZXNzX3Rva2VuIn0=`, header)
			},
		},
		{
			name: "test bad token",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
				require.Nil(t, err)

				req.Header.Set("Authorization", "bearer "+testproxyhelpers.BadToken)

				err = r.HydrationIntrospect(req)
				require.NotNil(t, err)
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

func TestPublish(t *testing.T) {
	defer dockertest.KillAllDockers()
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	ctx, dsn := initPublish(ctx, t)

	params := initParameters()
	params.Introspect = true
	r := NewRoute(ctx, "/api/v1/users", params)
	require.NotNil(t, r)
	r.Initilize()

	consum, err := rabbitMQConsumer.CreateConsumer(dsn)
	require.Nil(t, err)

	shutdownConsumer := make(chan bool)

	recivedMessage := ""
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		msgs, err1 := consum.StartConsume(testExchangeName, testQueue, r.Parameters.RouteKey, testConsumer)
		require.Nil(t, err1)
		wg.Done()

		t.Log("start consumer")
		for {
			select {
			case <-shutdownConsumer:
				return
			default:
				for d := range msgs {
					t.Logf("received a message: %s", d.Body)
					err1 := json.Unmarshal(d.Body, &recivedMessage)
					require.Nil(t, err1)
					_ = d.Ack(true)
				}
			}
		}
	}()

	wg.Wait()

	err = r.Publish(testMessage)
	require.Nil(t, err)
	close(shutdownConsumer)

	// timeout for consumer
	time.Sleep(3 * time.Second)

	c := rabbitmq.Get(ctx)
	err = c.Shutdown()
	require.Nil(t, err)
	err = consum.Shutdown()
	require.Nil(t, err)

	require.Equal(t, testMessage, recivedMessage)
}

func TestNotCached(t *testing.T) {
	server := testproxyhelpers.FakeBackendService(t, testproxyhelpers.DefaultFakeIntrospectorHost)
	server.Start()
	defer server.Close()

	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)

	params := initParameters()

	r := NewRoute(ctx, testproxyhelpers.GetEndpoint, params)
	require.NotNil(t, r)
	r.Initilize()

	var timestamp time.Time

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "first request",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, nil)
				require.Nil(t, err)

				w := httptest.NewRecorder()
				r.NotCached(w, req)

				resp := w.Result()
				body, err := ioutil.ReadAll(resp.Body)
				require.Nil(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode)

				var respData testproxyhelpers.HTTPBody
				err = json.Unmarshal(body, &respData)
				require.Nil(t, err)
				require.Equal(t, testproxyhelpers.DefaultGetAnswer, respData.Result.Message)

				timestamp = respData.Result.TimeStamp
			},
		},
		{
			name: "second request, timestamp not equal to the first request",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, nil)
				require.Nil(t, err)

				w := httptest.NewRecorder()
				r.NotCached(w, req)

				resp := w.Result()
				body, err := ioutil.ReadAll(resp.Body)
				require.Nil(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode)

				var respData testproxyhelpers.HTTPBody
				err = json.Unmarshal(body, &respData)
				require.Nil(t, err)
				require.Equal(t, testproxyhelpers.DefaultGetAnswer, respData.Result.Message)

				require.NotEqual(t, timestamp, respData.Result.TimeStamp)
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

// nolint : funlen
func TestRequestToBack(t *testing.T) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)

	params := initParameters()

	r := NewRoute(ctx, testproxyhelpers.GetEndpoint, params)
	require.NotNil(t, r)
	r.Initilize()

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test normal request",
			testFunc: func() {
				server := testproxyhelpers.FakeBackendService(t, testproxyhelpers.DefaultFakeIntrospectorHost)
				server.Start()
				defer server.Close()

				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, bytes.NewBufferString(testMessage))
				require.Nil(t, err)

				hk, err := httputils.HashRequest(req)
				require.Nil(t, err)

				w := httptest.NewRecorder()
				rrData := r.requestToBack(hk, w, req)
				require.NotNil(t, rrData)

				// Check rrData.Request
				require.Equal(t, http.MethodGet, rrData.Request.Method)
				require.Equal(t, testproxyhelpers.DefaultFakeServiceURL+testproxyhelpers.GetEndpoint, rrData.Request.URL)
				require.Equal(t, req.Header, rrData.Request.Header)
				require.Equal(t, testMessage, rrData.Request.Body)

				// Check rrData.Response
				var respData testproxyhelpers.HTTPBody
				err = json.Unmarshal([]byte(rrData.Response.Body), &respData)
				require.Nil(t, err)
				require.Equal(t, testproxyhelpers.DefaultGetAnswer, respData.Result.Message)
				require.Equal(t, http.StatusOK, rrData.Response.StatusCode)
				require.NotEmpty(t, rrData.Response.TimeStamp)
				require.NotEmpty(t, rrData.Response.UUID)

				// Check response writer
				respData = testproxyhelpers.HTTPBody{}
				resp := w.Result()
				body, err := ioutil.ReadAll(resp.Body)
				require.Nil(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode)

				err = json.Unmarshal(body, &respData)
				require.Nil(t, err)
				require.Equal(t, testproxyhelpers.DefaultGetAnswer, respData.Result.Message)
			},
		},
		{
			name: "test failed request",
			testFunc: func() {
				errMessage := `Get "http://localhost:10000/api/v1/check_get": dial tcp 127.0.0.1:10000: connect: connection refused`
				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, bytes.NewBufferString(testMessage))
				require.Nil(t, err)

				hk, err := httputils.HashRequest(req)
				require.Nil(t, err)

				w := httptest.NewRecorder()
				rrData := r.requestToBack(hk, w, req)
				require.NotNil(t, rrData)

				// Check rrData.Request
				require.Equal(t, http.MethodGet, rrData.Request.Method)
				require.Equal(t, testproxyhelpers.DefaultFakeServiceURL+testproxyhelpers.GetEndpoint, rrData.Request.URL)
				require.Equal(t, req.Header, rrData.Request.Header)
				require.Equal(t, testMessage, rrData.Request.Body)

				// Check rrData.Response
				require.Nil(t, err)
				require.Equal(t, errMessage, rrData.Response.Body)
				require.Equal(t, http.StatusServiceUnavailable, rrData.Response.StatusCode)

				// Check response writer
				resp := w.Result()
				body, err := ioutil.ReadAll(resp.Body)
				require.Nil(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
				require.Equal(t, errMessage, string(body))
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

// nolint : funlen
func TestCachedHandler(t *testing.T) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)

	params := initParameters()

	r := NewRoute(ctx, testproxyhelpers.GetEndpoint, params)
	require.NotNil(t, r)
	r.Initilize()

	workers := 10

	barrier := make(chan struct{})
	errorsCh := make(chan error, workers)

	server := testproxyhelpers.FakeBackendService(t, testproxyhelpers.DefaultFakeIntrospectorHost)
	server.Start()
	defer server.Close()

	backendRequestCnt := 0
	cacheRequestCnt := 0

	var wg sync.WaitGroup
	go func() {
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// nolint
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
				// all workers will block here until the for loop above has launched all the worker go-routines
				// this is to ensure we fire all the workers off at the same
				<-barrier

				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, bytes.NewBufferString(testMessage))
				require.Nil(t, err)

				w := httptest.NewRecorder()
				r.CachedHandler(w, req)

				var respData testproxyhelpers.HTTPBody
				resp := w.Result()
				body, err := ioutil.ReadAll(resp.Body)
				require.Nil(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode)

				err = json.Unmarshal(body, &respData)
				require.Nil(t, err)
				require.Equal(t, testproxyhelpers.DefaultGetAnswer, respData.Result.Message)

				var err1 error
				t.Log(resp.Header)
				if resp.Header.Get(rrdata.ResponseSourceHeader) == rrdata.ResponseBack.String() {
					if backendRequestCnt > 0 {
						err1 = errors.New("concurent request to backend")
					}
					backendRequestCnt++
				}

				if resp.Header.Get(rrdata.ResponseSourceHeader) == rrdata.ResponseCache.String() {
					cacheRequestCnt++
				}

				errorsCh <- err1
			}()
		}

		// wait until all workers have completed their work
		wg.Wait()
		close(errorsCh)
	}()

	// let the race begin!
	// all worker go-routines will now attempt to hit the "CachedHandler" method
	close(barrier)

	var successCount int
	for err := range errorsCh {
		if err != nil {
			t.Errorf("failed: %s", err)
		} else {
			successCount++
		}
	}

	t.Logf("requests to back %d, requests to cache %d", backendRequestCnt, cacheRequestCnt)
	require.Equal(t, workers, successCount)
}

// nolint : funlen
func TestRefresh(t *testing.T) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)

	params := initParameters()

	r := NewRoute(ctx, testproxyhelpers.GetEndpoint, params)
	require.NotNil(t, r)
	r.Initilize()

	server := testproxyhelpers.FakeBackendService(t, testproxyhelpers.DefaultFakeIntrospectorHost)
	server.Start()
	defer server.Close()

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "referesh by counter",
			testFunc: func() {
				// first request, filling cache
				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, bytes.NewBufferString(testMessage))
				require.Nil(t, err)

				w := httptest.NewRecorder()
				r.CachedHandler(w, req)

				var respData testproxyhelpers.HTTPBody
				resp := w.Result()
				body, err := ioutil.ReadAll(resp.Body)
				require.Nil(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode)

				err = json.Unmarshal(body, &respData)
				require.Nil(t, err)
				require.Equal(t, testproxyhelpers.DefaultGetAnswer, respData.Result.Message)

				randomValue := respData.Result.UUID

				// +1 for refreshing the cache
				for i := 0; i < r.Parameters.Refresh.MaxCount+1; i++ {
					req, err = http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, bytes.NewBufferString(testMessage))
					require.Nil(t, err)

					w = httptest.NewRecorder()
					r.CachedHandler(w, req)
					resp = w.Result()
					defer resp.Body.Close()
				}

				// Sleep for waitig then end of refereshing
				time.Sleep(1 * time.Second)

				// request after refreshing cache
				req, err = http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, bytes.NewBufferString(testMessage))
				require.Nil(t, err)

				w = httptest.NewRecorder()
				r.CachedHandler(w, req)

				resp = w.Result()
				body, err = ioutil.ReadAll(resp.Body)
				require.Nil(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode)
				require.Equal(t, rrdata.ResponseCache.String(), resp.Header.Get(rrdata.ResponseSourceHeader))

				err = json.Unmarshal(body, &respData)
				require.Nil(t, err)

				t.Logf("first random value %s, random value after refresh %s", randomValue, respData.Result.UUID)
				require.NotEqual(t, randomValue, respData.Result.UUID)
			},
		},
		{
			name: "referesh by time",
			testFunc: func() {
				// first request, filling cache
				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, bytes.NewBufferString(testMessage))
				require.Nil(t, err)

				w := httptest.NewRecorder()
				r.CachedHandler(w, req)

				var respData testproxyhelpers.HTTPBody
				resp := w.Result()
				body, err := ioutil.ReadAll(resp.Body)
				require.Nil(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode)

				err = json.Unmarshal(body, &respData)
				require.Nil(t, err)
				require.Equal(t, testproxyhelpers.DefaultGetAnswer, respData.Result.Message)

				randomValue := respData.Result.UUID

				// Sleep for waitig then end of refereshing
				time.Sleep(r.Parameters.Refresh.Time + 1*time.Second)

				// request after refreshing cache
				req, err = http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, bytes.NewBufferString(testMessage))
				require.Nil(t, err)

				w = httptest.NewRecorder()
				r.CachedHandler(w, req)

				resp = w.Result()
				body, err = ioutil.ReadAll(resp.Body)
				require.Nil(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode)
				require.Equal(t, rrdata.ResponseCache.String(), resp.Header.Get(rrdata.ResponseSourceHeader))

				err = json.Unmarshal(body, &respData)
				require.Nil(t, err)

				t.Logf("first random value %s, random value after refresh %s", randomValue, respData.Result.UUID)
				require.NotEqual(t, randomValue, respData.Result.UUID)
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

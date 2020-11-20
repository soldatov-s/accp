package httpproxy_test

import (
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/cache/external"
	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/internal/httpproxy"
	"github.com/soldatov-s/accp/internal/introspection"
	"github.com/soldatov-s/accp/internal/publisher"
	"github.com/soldatov-s/accp/internal/rabbitmq"
	externalcache "github.com/soldatov-s/accp/internal/redis"
	"github.com/soldatov-s/accp/x/dockertest"
	testhelpers "github.com/soldatov-s/accp/x/test_helpers"
	testProxyHelpers "github.com/soldatov-s/accp/x/test_helpers/proxy"
	rabbitMQConsumer "github.com/soldatov-s/accp/x/test_helpers/rabbitmq"
	resilience "github.com/soldatov-s/accp/x/test_helpers/resilence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initExternalCache(t *testing.T) *externalcache.RedisClient {
	lc, err := testhelpers.LoadTestConfigLogger()
	require.Nil(t, err)

	ctx := context.NewContext()
	ctx.InitilizeLogger(lc)

	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)

	t.Logf("Connecting to redis: %s", dsn)

	ec := &externalcache.RedisConfig{
		DSN:                   dsn,
		MinIdleConnections:    10,
		MaxOpenedConnections:  30,
		MaxConnectionLifetime: 30 * time.Second,
	}

	var externalStorage *externalcache.RedisClient
	err = resilience.Retry(
		t,
		time.Second*5,
		time.Minute*5,
		func() (err error) {
			externalStorage, err = externalcache.NewRedisClient(ctx, ec)
			return err
		},
	)
	require.Nil(t, err)
	require.NotNil(t, externalStorage)
	t.Logf("Connected to redis: %s", dsn)

	return externalStorage
}

func initPublisher(t *testing.T) (string, publisher.Publisher) {
	lc, err := testhelpers.LoadTestConfigLogger()
	require.Nil(t, err)

	ctx := context.NewContext()
	ctx.InitilizeLogger(lc)

	dsn, err := dockertest.RunRabbitMQ()
	require.Nil(t, err)

	t.Logf("Connecting to rabbitmq: %s", dsn)

	ec := &rabbitmq.PublisherConfig{
		DSN:           dsn,
		BackoffPolicy: []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second, 15 * time.Second, 20 * time.Second, 25 * time.Second},
		ExchangeName:  "testout.events.dev",
	}

	var pub publisher.Publisher
	err = resilience.Retry(
		t,
		time.Second*5,
		time.Minute*5,
		func() (err error) {
			pub, err = rabbitmq.NewPublisher(ctx, ec)
			return err
		},
	)
	require.Nil(t, err)
	require.NotNil(t, pub)
	t.Logf("Connected to rabbitmq: %s", dsn)

	return dsn, pub
}

func initProxy(t *testing.T) *httpproxy.HTTPProxy {
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

	return p
}

func TestNewHTTPProxy(t *testing.T) {
	initProxy(t)
}

func TestHTTPProxy_FindRouteByPath(t *testing.T) {
	p := initProxy(t)

	route := p.FindRouteByPath("/api/v1/users")
	require.NotNil(t, route)

	t.Logf("route value %+v", route)
}

func TestHTTPProxy_FindRouteByHTTPRequest(t *testing.T) {
	p := initProxy(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	route := p.FindRouteByHTTPRequest(r)
	require.NotNil(t, route)

	t.Logf("route value %+v", route)
}

func TestHTTPProxy_FindExcluededRouteByHTTPRequest(t *testing.T) {
	p := initProxy(t)

	r, err := http.NewRequest("POST", "/api/v1/users/search", nil)
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

func TestHTTPProxy_DefaultHandler(t *testing.T) {
	server := testProxyHelpers.FakeBackendService(t, "localhost:9090")
	server.Start()
	defer server.Close()

	p := initProxy(t)

	r, err := http.NewRequest("POST", "/api/v1/users/search", nil)
	require.Nil(t, err)

	route := p.FindExcludedRouteByHTTPRequest(r)
	require.NotNil(t, route)

	t.Logf("route value %+v", route)

	w := httptest.NewRecorder()
	p.NonCachedHandler(route, w, r)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	t.Log(resp.StatusCode)
	t.Log(resp.Header.Get("Content-Type"))
	t.Log(string(body))
}

func TestIntrospection(t *testing.T) {
	server := testProxyHelpers.FakeIntrospectorService(t, testhelpers.IntrospectorHost)
	server.Start()
	defer server.Close()

	p := initProxy(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	route := p.FindRouteByHTTPRequest(r)
	require.NotNil(t, route)

	r.Header.Add("Authorization", "bearer "+testProxyHelpers.TestToken)

	err = p.HydrationIntrospect(route, r)
	require.Nil(t, err)

	r.Header.Set("Authorization", "bearer "+testProxyHelpers.BadToken)
	err = p.HydrationIntrospect(route, r)
	require.NotNil(t, err)
}

func TestHTTPProxy_GetHandler(t *testing.T) {
	server := testProxyHelpers.FakeBackendService(t, "localhost:9090")
	server.Start()
	defer server.Close()

	p := initProxy(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	route := p.FindRouteByHTTPRequest(r)
	require.NotNil(t, route)

	t.Logf("route value %+v", route)

	w := httptest.NewRecorder()
	p.CachedHandler(route, w, r)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	t.Log(resp.StatusCode)
	t.Log(resp.Header.Get("Content-Type"))
	t.Log(string(body))

	t.Log("take answer from cache")
	w = httptest.NewRecorder()
	p.CachedHandler(route, w, r)

	resp = w.Result()
	body, _ = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	t.Log(resp.StatusCode)
	t.Log(resp.Header.Get("Content-Type"))
	t.Log(string(body))
}

func TestHTTPProxy_GetHandlerExternalCache(t *testing.T) {
	server := testProxyHelpers.FakeBackendService(t, "localhost:9090")
	server.Start()
	defer server.Close()

	p := initProxy(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	route := p.FindRouteByHTTPRequest(r)
	require.NotNil(t, route)

	lc, err := testhelpers.LoadTestConfigLogger()
	require.Nil(t, err)

	ctx := context.NewContext()
	ctx.InitilizeLogger(lc)

	route.Cache.External, err = external.NewCache(ctx, route.Parameters.Cache.External, initExternalCache(t))
	require.Nil(t, err)

	w := httptest.NewRecorder()
	p.CachedHandler(route, w, r)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	t.Log(resp.StatusCode)
	t.Log(resp.Header.Get("Content-Type"))
	t.Log(string(body))

	// Sleep, inmemory cache invalidates
	t.Log("sleep, inmemory cache invalidates")
	time.Sleep(5 * time.Second)

	r, err = http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	t.Log("take answer from cache")
	w = httptest.NewRecorder()
	p.CachedHandler(route, w, r)

	resp = w.Result()
	body, _ = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	t.Log(resp.StatusCode)
	t.Log(resp.Header.Get("Content-Type"))
	t.Log(string(body))

	dockertest.KillAllDockers()
}

func TestMultipleClienRequests(t *testing.T) {
	workers := 10

	barrier := make(chan struct{})
	errorsCh := make(chan error, workers)

	server := testProxyHelpers.FakeBackendService(t, "localhost:9090")
	server.Start()
	defer server.Close()

	p := initProxy(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	route := p.FindRouteByHTTPRequest(r)
	require.NotNil(t, route)

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

				t.Log("take answer from cache")
				w := httptest.NewRecorder()
				p.CachedHandler(route, w, r)

				resp := w.Result()
				body, _ := ioutil.ReadAll(resp.Body)
				defer resp.Body.Close()

				t.Log(resp.StatusCode)
				t.Log(resp.Header.Get("Content-Type"))
				t.Log(string(body))

				errorsCh <- err
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
}

func TestHTTPProxy_GetHandlerSendMessageToQueue(t *testing.T) {
	server := testProxyHelpers.FakeBackendService(t, "localhost:9090")
	server.Start()
	defer server.Close()

	p := initProxy(t)

	r, err := http.NewRequest("GET", "/api/v1/users", nil)
	require.Nil(t, err)

	route := p.FindRouteByHTTPRequest(r)
	require.NotNil(t, route)

	lc, err := testhelpers.LoadTestConfigLogger()
	require.Nil(t, err)

	ctx := context.NewContext()
	ctx.InitilizeLogger(lc)

	var dsn string
	dsn, route.Publisher = initPublisher(t)

	consum, err := rabbitMQConsumer.CreateConsumer(dsn)
	require.Nil(t, err)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		msgs, err1 := consum.StartConsume("testout.events.dev", "test.queue", route.Parameters.RouteKey, "tests")
		require.Nil(t, err1)
		wg.Done()

		for d := range msgs {
			t.Logf("Received a message: %s", d.Body)
			require.Equal(t, []byte(`{"URL":"http://localhost:9090/api/v1/users","Method":"GET","Body":"","Header":{}}`), d.Body)
			_ = d.Ack(true)
		}
	}()

	wg.Wait()

	w := httptest.NewRecorder()
	p.CachedHandler(route, w, r)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	t.Log(resp.StatusCode)
	t.Log(resp.Header.Get("Content-Type"))
	t.Log(string(body))

	t.Log("take answer from cache")
	w = httptest.NewRecorder()
	p.CachedHandler(route, w, r)

	resp = w.Result()
	body, _ = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	t.Log(resp.StatusCode)
	t.Log(resp.Header.Get("Content-Type"))
	t.Log(string(body))

	err = route.Publisher.(*rabbitmq.Publish).Shutdown()
	require.Nil(t, err)

	dockertest.KillAllDockers()
}

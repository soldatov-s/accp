package limits

import (
	"context"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/meta"
	"github.com/soldatov-s/accp/internal/redis"
	"github.com/soldatov-s/accp/x/dockertest"
	"github.com/soldatov-s/accp/x/test_helpers/resilience"
	"github.com/stretchr/testify/require"
)

const (
	testRoute          = "test"
	testToken          = "testToken"
	testCounter        = 5
	testPT             = 1 * time.Second
	testAuthCookieName = "accp-authorization"
)

func testHeader() []string {
	return []string{authorizationHeader}
}

func testCookie() []string {
	return []string{testAuthCookieName}
}

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

func initExternal(ctx context.Context, t *testing.T, dsn string) *redis.Client {
	if dsn == "" {
		return nil
	}

	cfg := &redis.Config{
		MinIdleConnections:    10,
		MaxOpenedConnections:  30,
		MaxConnectionLifetime: 30 * time.Second,
	}

	cfg.DSN = dsn

	t.Logf("connecting to redis: %s", dsn)

	client, err := redis.NewClient(ctx, cfg)
	require.Nil(t, err)
	require.NotNil(t, client)

	err = resilience.Retry(
		t,
		time.Second*5,
		time.Minute*5,
		func() (err error) {
			err = client.Start()
			return err
		},
	)

	require.Nil(t, err)
	require.NotNil(t, client)

	t.Logf("connected to redis: %s", dsn)

	return client
}

func initCacheConfig() *external.Config {
	return &external.Config{
		TTL:       5 * time.Second,
		TTLErr:    3 * time.Second,
		KeyPrefix: "accp_",
	}
}

func initCache(t *testing.T, dsn string) *external.Cache {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initCacheConfig()

	client := initExternal(ctx, t, dsn)
	c := external.NewCache(ctx, cfg, client)

	return c
}

func initConfig() *Config {
	c := &Config{
		Header:  testHeader(),
		Cookie:  testCookie(),
		Counter: testCounter,
		PT:      testPT,
	}

	return c
}

func TestNewLimit(t *testing.T) {
	l := NewLimit()
	require.NotNil(t, l)
	require.Equal(t, int64(0), l.Counter)
	require.NotEmpty(t, l.LastAccess)
}

func initLimitTable(t *testing.T, dsn string) *LimitTable {
	c := initCache(t, dsn)
	cfg := initConfig()

	lt := NewLimitTable(testRoute, cfg, c)

	return lt
}

func TestNewLimitTable(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	lt := initLimitTable(t, dsn)
	require.NotNil(t, lt)
	require.NotNil(t, lt.clearTimer)
	require.Equal(t, testCounter, lt.maxCount)
	require.Equal(t, testPT, lt.pt)
	require.Equal(t, testRoute, lt.route)
	require.NotNil(t, lt.cache)
}

// nolint : dupl
func TestInc(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	lt := initLimitTable(t, dsn)
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "first increment",
			testFunc: func() {
				err = lt.Inc(testToken)
				require.Nil(t, err)

				v, ok := lt.list.Load(testToken)
				require.True(t, ok)
				require.NotNil(t, v)
				limit, ok := v.(*Limit)
				require.True(t, ok)
				require.Equal(t, int64(1), limit.Counter)
				require.NotEmpty(t, limit.LastAccess)

				var limitInCache int
				err = lt.cache.GetLimit(lt.route+"_"+testToken, &limitInCache)
				require.Nil(t, err)
				require.Equal(t, 1, limitInCache)
			},
		},
		{
			name: "second increment",
			testFunc: func() {
				err = lt.Inc(testToken)
				require.Nil(t, err)

				v, ok := lt.list.Load(testToken)
				require.True(t, ok)
				require.NotNil(t, v)
				limit, ok := v.(*Limit)
				require.True(t, ok)
				require.Equal(t, int64(2), limit.Counter)
				require.NotEmpty(t, limit.LastAccess)

				var limitInCache int
				err = lt.cache.GetLimit(lt.route+"_"+testToken, &limitInCache)
				require.Nil(t, err)
				require.Equal(t, 2, limitInCache)
			},
		},
		{
			name: "after expire",
			testFunc: func() {
				time.Sleep(testPT)
				var limitInCache int
				err = lt.cache.GetLimit(lt.route+"_"+testToken, &limitInCache)
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

func TestClearTable(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	lt := initLimitTable(t, dsn)
	require.NotNil(t, lt)

	v := NewLimit()
	lt.list.Store(testToken, v)

	// Timer starts after initilizing limit
	// NewLimit creates time stamp much later.
	// After double pt defaultClearLimitPeriod really deletes limit
	time.Sleep(2 * defaultClearLimitPeriod)

	_, ok := lt.list.Load(testToken)
	require.False(t, ok)
}

func TestCheck(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	lt := initLimitTable(t, dsn)
	require.NotNil(t, lt)

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "check after init",
			testFunc: func() {
				var result bool
				err = lt.Check(testToken, &result)
				require.Nil(t, err)

				require.False(t, result)
			},
		},
		{
			name: "check after first increment",
			testFunc: func() {
				err = lt.Inc(testToken)
				require.Nil(t, err)

				var result bool
				err = lt.Check(testToken, &result)
				require.Nil(t, err)

				require.False(t, result)
			},
		},
		{
			name: "check after overflow",
			testFunc: func() {
				for i := 0; i < testCounter; i++ {
					err = lt.Inc(testToken)
					require.Nil(t, err)
				}

				var result bool
				err = lt.Check(testToken, &result)
				require.Nil(t, err)

				require.True(t, result)
			},
		},
		{
			name: "check after expire",
			testFunc: func() {
				time.Sleep(defaultClearLimitPeriod)

				var result bool
				err = lt.Check(testToken, &result)
				require.Nil(t, err)

				require.False(t, result)
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

func TestNewLimits(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	mc := NewMapConfig()
	c := initCache(t, dsn)

	l := NewLimits(testRoute, mc, c)
	require.NotNil(t, l)
	require.Equal(t, len(l), len(mc))
	for k := range mc {
		v, ok := l[k]
		require.True(t, ok)
		require.Equal(t, defaultCounter, v.maxCount)
		require.Equal(t, defaultPT, v.pt)
		require.Equal(t, v.route, testRoute)
		require.NotNil(t, v.cache)
	}
}

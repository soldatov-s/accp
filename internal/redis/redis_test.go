package redis

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/meta"
	"github.com/soldatov-s/accp/x/dockertest"
	"github.com/soldatov-s/accp/x/test_helpers/resilience"
	"github.com/stretchr/testify/require"
)

const (
	defaultKey = "test"
)

type testCacheData struct {
	Name       string
	StatusCode int
}

func (d *testCacheData) MarshalBinary() ([]byte, error) {
	return json.Marshal(d)
}

func (d *testCacheData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, d)
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

func initConfig() *Config {
	return &Config{
		MinIdleConnections:    10,
		MaxOpenedConnections:  30,
		MaxConnectionLifetime: 30 * time.Second,
	}
}

func initClient(t *testing.T, dsn string) *Client {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()

	cfg.DSN = dsn

	t.Logf("connecting to redis: %s", dsn)

	client, err := NewClient(ctx, cfg)
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

func TestNewRedisClient(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	_ = initClient(t, dsn)
}

func TestAdd(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)

	tesdData := &testCacheData{Name: "test data", StatusCode: 200}
	testTTL := 5 * time.Second
	err = c.Add(defaultKey, tesdData, testTTL)
	require.Nil(t, err)

	d, err := c.Conn.TTL(c.Conn.Context(), defaultKey).Result()
	require.Nil(t, err)
	require.Equal(t, testTTL, d)
}

func TestSelect(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)
	testData := &testCacheData{Name: "test data", StatusCode: 200}
	testTTL := 5 * time.Second
	err = c.Add(defaultKey, testData, testTTL)
	require.Nil(t, err)

	var result testCacheData
	err = c.Select(defaultKey, &result)
	require.Nil(t, err)
	require.Equal(t, testData, &result)
	t.Logf("data from cache: %+v", result)
}

func TestExpire(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)
	testData := &testCacheData{Name: "test data", StatusCode: 200}
	testTTL := time.Duration(-1)
	c.Conn.Set(c.Conn.Context(), defaultKey, testData, 0)

	d, err := c.Conn.TTL(c.Conn.Context(), defaultKey).Result()
	require.Nil(t, err)
	require.Equal(t, testTTL, d)

	testTTL = 5 * time.Second
	err = c.Expire(defaultKey, testTTL)
	require.Nil(t, err)

	d, err = c.Conn.TTL(c.Conn.Context(), defaultKey).Result()
	require.Nil(t, err)
	require.Equal(t, testTTL, d)
}

func TestUpdate(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)
	testData := &testCacheData{Name: "test data", StatusCode: 200}
	testTTL := 5 * time.Second
	err = c.Add(defaultKey, testData, testTTL)
	require.Nil(t, err)

	testData = &testCacheData{Name: "test data", StatusCode: 300}
	err = c.Update(defaultKey, testData, testTTL)
	require.Nil(t, err)

	d, err := c.Conn.TTL(c.Conn.Context(), defaultKey).Result()
	require.Nil(t, err)
	require.Equal(t, testTTL, d)

	var result testCacheData
	err = c.Select(defaultKey, &result)
	require.Nil(t, err)
	require.Equal(t, testData, &result)
	t.Logf("data from cache: %+v", result)
}

func TestJSONGet(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)
	testData := &testCacheData{Name: "test data", StatusCode: 200}
	testTTL := 5 * time.Second
	err = c.Add(defaultKey, testData, testTTL)
	require.Nil(t, err)

	var result int
	err = c.JSONGet(defaultKey, "StatusCode", &result)
	require.Nil(t, err)
	require.Equal(t, testData.StatusCode, result)
}

func TestJSONSet(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)
	testData := &testCacheData{Name: "test data", StatusCode: 200}
	testTTL := 5 * time.Second
	err = c.Add(defaultKey, testData, testTTL)
	require.Nil(t, err)

	err = c.JSONSet(defaultKey, "StatusCode", "300")
	require.Nil(t, err)

	var result int
	err = c.JSONGet(defaultKey, "StatusCode", &result)
	require.Nil(t, err)
	require.Equal(t, 300, result)
}

func TestJSONSetNX(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)
	testData := &testCacheData{Name: "test data", StatusCode: 200}
	testTTL := 5 * time.Second
	err = c.Add(defaultKey, testData, testTTL)
	require.Nil(t, err)

	jData, err := testData.MarshalBinary()
	require.Nil(t, err)

	err = c.JSONSetNX(defaultKey, ".", string(jData))
	require.NotNil(t, err)

	err = c.JSONDelete(defaultKey, ".")
	require.Nil(t, err)

	err = c.JSONSetNX(defaultKey, ".", string(jData))
	require.Nil(t, err)
}

func TestJSONDelete(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)
	testData := &testCacheData{Name: "test data", StatusCode: 200}
	testTTL := 5 * time.Second
	err = c.Add(defaultKey, testData, testTTL)
	require.Nil(t, err)

	err = c.JSONDelete(defaultKey, ".")
	require.Nil(t, err)

	var result testCacheData
	err = c.Select(defaultKey, &result)
	require.NotNil(t, err)
	require.NotEqual(t, testData, &result)

	err = c.Add(defaultKey, testData, testTTL)
	require.Nil(t, err)

	err = c.JSONDelete(defaultKey, ".StatusCode")
	require.Nil(t, err)

	err = c.Select(defaultKey, &result)
	require.Nil(t, err)
	require.NotEqual(t, testData, &result)
	require.Equal(t, 0, result.StatusCode)
}

func TestFormatSec(t *testing.T) {
	res := formatSec(time.Millisecond)
	require.Equal(t, int64(1), res)

	res = formatSec(10 * time.Second)
	require.Equal(t, int64(10), res)
}

func TestLimitTTL(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)

	testTTL := 5 * time.Second

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "first increment",
			testFunc: func() {
				err = c.LimitTTL(defaultKey, testTTL)
				require.Nil(t, err)

				var d time.Duration
				d, err = c.Conn.TTL(c.Conn.Context(), defaultKey).Result()
				require.Nil(t, err)
				require.Equal(t, testTTL, d)

				var result int64
				cmdString := c.Conn.Get(c.Conn.Context(), defaultKey)
				_, err = cmdString.Result()
				require.Nil(t, err)

				err = cmdString.Scan(&result)
				require.Nil(t, err)
				require.Equal(t, int64(1), result)
			},
		},
		{
			name: "second increment",
			testFunc: func() {
				err = c.LimitTTL(defaultKey, testTTL)
				require.Nil(t, err)

				var result int64
				cmdString := c.Conn.Get(c.Conn.Context(), defaultKey)
				_, err = cmdString.Result()
				require.Nil(t, err)

				err = cmdString.Scan(&result)
				require.Nil(t, err)
				require.Equal(t, int64(2), result)
			}},
		{
			name: "after expire",
			testFunc: func() {
				time.Sleep(testTTL)
				cmdString := c.Conn.Get(c.Conn.Context(), defaultKey)
				_, err = cmdString.Result()
				require.NotNil(t, err)
			},
		},
		{
			name: "increment after expire",
			testFunc: func() {
				err = c.LimitTTL(defaultKey, testTTL)
				require.Nil(t, err)

				var result int64
				cmdString := c.Conn.Get(c.Conn.Context(), defaultKey)
				_, err = cmdString.Result()
				require.Nil(t, err)

				err = cmdString.Scan(&result)
				require.Nil(t, err)
				require.Equal(t, int64(1), result)
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

func TestLimitCount(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)

	testCount := 3

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "first increment",
			testFunc: func() {
				err = c.LimitCount(defaultKey, testCount)
				require.Nil(t, err)

				var result int64
				cmdString := c.Conn.Get(c.Conn.Context(), defaultKey)
				_, err = cmdString.Result()
				require.Nil(t, err)

				err = cmdString.Scan(&result)
				require.Nil(t, err)
				require.Equal(t, int64(1), result)
			},
		},
		{
			name: "second increment",
			testFunc: func() {
				err = c.LimitCount(defaultKey, testCount)
				require.Nil(t, err)

				var result int64
				cmdString := c.Conn.Get(c.Conn.Context(), defaultKey)
				_, err = cmdString.Result()
				require.Nil(t, err)

				err = cmdString.Scan(&result)
				require.Nil(t, err)
				require.Equal(t, int64(2), result)
			}},
		{
			name: "increment after overflow",
			testFunc: func() {
				err = c.LimitCount(defaultKey, testCount)
				require.Nil(t, err)

				var result int64
				cmdString := c.Conn.Get(c.Conn.Context(), defaultKey)
				_, err = cmdString.Result()
				require.Nil(t, err)

				err = cmdString.Scan(&result)
				require.Nil(t, err)
				require.Equal(t, int64(0), result)
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

func TestGetLimit(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test get count limit",
			testFunc: func() {
				testCount := 3
				err = c.LimitCount(defaultKey, testCount)
				require.Nil(t, err)

				var cnt int64
				err = c.GetLimit(defaultKey, &cnt)
				require.Nil(t, err)
				require.Equal(t, int64(1), cnt)
			},
		},
		{
			name: "test get ttl limit",
			testFunc: func() {
				testTTL := 5 * time.Second
				err = c.LimitTTL(defaultKey, testTTL)
				require.Nil(t, err)

				var cnt int64
				err = c.GetLimit(defaultKey, &cnt)
				require.Nil(t, err)
				require.Equal(t, int64(1), cnt)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc()
			c.Conn.Client.Del(c.Conn.Context(), defaultKey)
		})
	}
}

func TestGetMetrics(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)
	m := c.GetMetrics()
	require.NotNil(t, m)

	_, ok := m[ProviderName+"_status"]
	require.True(t, ok)
}

func TestGetReadyHandlers(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initClient(t, dsn)
	h := c.GetReadyHandlers()
	require.NotNil(t, h)

	_, ok := h[strings.ToUpper(ProviderName+"_notfailed")]
	require.True(t, ok)
}

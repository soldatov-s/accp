package external

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soldatov-s/accp/internal/cache/errors"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/meta"
	"github.com/soldatov-s/accp/internal/redis"
	"github.com/soldatov-s/accp/x/dockertest"
	"github.com/soldatov-s/accp/x/test_helpers/resilience"
	"github.com/stretchr/testify/require"
)

const (
	defaultKey = "test"
)

type testCacheData struct {
	Name       string `json:"name"`
	StatusCode int    `json:"status_code"`
}

type testCacheDataWithUUID struct {
	Data *testCacheData `json:"data"`
	UUID uuid.UUID      `json:"uuid"`
}

func (d *testCacheDataWithUUID) MarshalBinary() ([]byte, error) {
	return json.Marshal(d)
}

func (d *testCacheDataWithUUID) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, d)
}

func (d *testCacheDataWithUUID) GetStatusCode() int {
	return d.Data.StatusCode
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

func initConfig() *Config {
	return &Config{
		TTL:       5 * time.Second,
		TTLErr:    3 * time.Second,
		KeyPrefix: "accp_",
	}
}

func initCache(t *testing.T, dsn string) (*Cache, *redis.Client) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()

	external := initExternal(ctx, t, dsn)
	c := NewCache(ctx, cfg, external)

	return c, external
}
func TestNewCache(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "external cache is not nil",
			testFunc: func() {
				dsn, err := dockertest.RunRedis()
				require.Nil(t, err)
				defer dockertest.KillAllDockers()

				c, _ := initCache(t, dsn)
				require.NotNil(t, c)
			},
		},
		{
			name: "external cache is nil",
			testFunc: func() {
				c, _ := initCache(t, "")
				require.Nil(t, c)
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

func TestAdd(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c, client := initCache(t, dsn)
	cfg := initConfig()

	tesdData := &testCacheDataWithUUID{
		Data: &testCacheData{Name: "test data", StatusCode: 200},
		UUID: uuid.New(),
	}
	err = c.Add(defaultKey, tesdData)
	require.Nil(t, err)

	var result testCacheDataWithUUID
	err = c.ExternalStorage.Select(cfg.KeyPrefix+defaultKey, &result)
	require.Nil(t, err)
	require.Equal(t, tesdData, &result)

	d, err := client.Conn.TTL(client.Conn.Context(), cfg.KeyPrefix+defaultKey).Result()
	require.Nil(t, err)
	require.Equal(t, cfg.TTL, d)

	// Test add StatusCode 400
	tesdData.Data.StatusCode = 400
	err = c.Add(defaultKey, tesdData)
	require.Nil(t, err)
	d, err = client.Conn.TTL(client.Conn.Context(), cfg.KeyPrefix+defaultKey).Result()
	require.Nil(t, err)
	require.Equal(t, cfg.TTLErr, d)
}

func TestSelect(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c, _ := initCache(t, dsn)

	tesdData := &testCacheDataWithUUID{
		Data: &testCacheData{Name: "test data", StatusCode: 200},
		UUID: uuid.New(),
	}
	err = c.Add(defaultKey, tesdData)
	require.Nil(t, err)

	var result testCacheDataWithUUID
	err = c.Select(defaultKey, &result)
	require.Nil(t, err)
	require.Equal(t, tesdData, &result)

	// Test not found
	err = c.Select(defaultKey+"1", &result)
	require.NotNil(t, err)
	require.Equal(t, err, errors.ErrNotFound)
}

func TestExpire(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	cfg := initConfig()
	c, client := initCache(t, dsn)

	tesdData := &testCacheDataWithUUID{
		Data: &testCacheData{Name: "test data", StatusCode: 200},
		UUID: uuid.New(),
	}
	err = c.Add(defaultKey, tesdData)
	require.Nil(t, err)

	time.Sleep(2 * time.Second)

	err = c.Expire(defaultKey)
	require.Nil(t, err)

	d, err := client.Conn.TTL(client.Conn.Context(), cfg.KeyPrefix+defaultKey).Result()
	require.Nil(t, err)
	require.Equal(t, cfg.TTL, d)
}

func TestUpdate(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	cfg := initConfig()
	c, client := initCache(t, dsn)

	tesdData := &testCacheDataWithUUID{
		Data: &testCacheData{Name: "test data", StatusCode: 200},
		UUID: uuid.New(),
	}
	err = c.Add(defaultKey, tesdData)
	require.Nil(t, err)

	var result testCacheDataWithUUID
	err = c.Select(defaultKey, &result)
	require.Nil(t, err)
	require.Equal(t, tesdData, &result)

	d, err := client.Conn.TTL(client.Conn.Context(), cfg.KeyPrefix+defaultKey).Result()
	require.Nil(t, err)
	require.Equal(t, cfg.TTL, d)

	tesdData.Data.Name = "test data 1"
	tesdData.Data.StatusCode = 400

	err = c.Update(defaultKey, tesdData)
	require.Nil(t, err)

	err = c.Select(defaultKey, &result)
	require.Nil(t, err)
	require.Equal(t, tesdData, &result)

	d, err = client.Conn.TTL(client.Conn.Context(), cfg.KeyPrefix+defaultKey).Result()
	require.Nil(t, err)
	require.Equal(t, cfg.TTLErr, d)
}

func TestJSONGet(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c, _ := initCache(t, dsn)

	testData := &testCacheDataWithUUID{
		Data: &testCacheData{Name: "test data", StatusCode: 200},
		UUID: uuid.New(),
	}
	err = c.Add(defaultKey, testData)
	require.Nil(t, err)

	var result int
	err = c.JSONGet(defaultKey, "data.status_code", &result)
	require.Nil(t, err)
	require.Equal(t, testData.Data.StatusCode, result)
}

func TestJSONSet(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c, _ := initCache(t, dsn)

	testData := &testCacheDataWithUUID{
		Data: &testCacheData{Name: "test data", StatusCode: 200},
		UUID: uuid.New(),
	}
	err = c.Add(defaultKey, testData)
	require.Nil(t, err)

	err = c.JSONSet(defaultKey, "data.status_code", "300")
	require.Nil(t, err)

	var result int
	err = c.JSONGet(defaultKey, "data.status_code", &result)
	require.Nil(t, err)
	require.Equal(t, 300, result)
}

func TestJSONSetNX(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c, _ := initCache(t, dsn)

	testData := &testCacheDataWithUUID{
		Data: &testCacheData{Name: "test data", StatusCode: 200},
		UUID: uuid.New(),
	}
	err = c.Add(defaultKey, testData)
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

	c, _ := initCache(t, dsn)

	testData := &testCacheDataWithUUID{
		Data: &testCacheData{Name: "test data", StatusCode: 200},
		UUID: uuid.New(),
	}
	err = c.Add(defaultKey, testData)
	require.Nil(t, err)

	err = c.JSONDelete(defaultKey, ".")
	require.Nil(t, err)

	var result testCacheDataWithUUID
	err = c.Select(defaultKey, &result)
	require.NotNil(t, err)
	require.NotEqual(t, testData, &result)

	err = c.Add(defaultKey, testData)
	require.Nil(t, err)

	err = c.JSONDelete(defaultKey, "data.status_code")
	require.Nil(t, err)

	err = c.Select(defaultKey, &result)
	require.Nil(t, err)
	require.NotEqual(t, testData, &result)
	require.Equal(t, 0, result.Data.StatusCode)
}

func TestGetUUID(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c, _ := initCache(t, dsn)

	testData := &testCacheDataWithUUID{
		Data: &testCacheData{Name: "test data", StatusCode: 200},
		UUID: uuid.New(),
	}
	err = c.Add(defaultKey, testData)
	require.Nil(t, err)

	var UUID string
	err = c.GetUUID(defaultKey, &UUID)
	require.Nil(t, err)
	require.Equal(t, testData.UUID.String(), UUID)
}

func TestLimitTTL(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	cfg := initConfig()
	c, client := initCache(t, dsn)

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
				d, err = client.Conn.TTL(client.Conn.Context(), cfg.KeyPrefix+defaultKey).Result()
				require.Nil(t, err)
				require.Equal(t, testTTL, d)

				var result int64
				cmdString := client.Conn.Get(client.Conn.Context(), cfg.KeyPrefix+defaultKey)
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
				cmdString := client.Conn.Get(client.Conn.Context(), cfg.KeyPrefix+defaultKey)
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
				cmdString := client.Conn.Get(client.Conn.Context(), cfg.KeyPrefix+defaultKey)
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
				cmdString := client.Conn.Get(client.Conn.Context(), cfg.KeyPrefix+defaultKey)
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

	cfg := initConfig()
	c, client := initCache(t, dsn)

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
				cmdString := client.Conn.Get(client.Conn.Context(), cfg.KeyPrefix+defaultKey)
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
				cmdString := client.Conn.Get(client.Conn.Context(), cfg.KeyPrefix+defaultKey)
				_, err = cmdString.Result()
				require.Nil(t, err)

				err = cmdString.Scan(&result)
				require.Nil(t, err)
				require.Equal(t, int64(2), result)
			},
		},
		{
			name: "third increment",
			testFunc: func() {
				err = c.LimitCount(defaultKey, testCount)
				require.Nil(t, err)

				var result int64
				cmdString := client.Conn.Get(client.Conn.Context(), cfg.KeyPrefix+defaultKey)
				_, err = cmdString.Result()
				require.Nil(t, err)

				err = cmdString.Scan(&result)
				require.Nil(t, err)
				require.Equal(t, int64(3), result)
			},
		},
		{
			name: "increment after overflow",
			testFunc: func() {
				err = c.LimitCount(defaultKey, testCount)
				require.Nil(t, err)

				var result int64
				cmdString := client.Conn.Get(client.Conn.Context(), cfg.KeyPrefix+defaultKey)
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

	cfg := initConfig()
	c, client := initCache(t, dsn)

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
			client.Conn.Client.Del(client.Conn.Context(), cfg.KeyPrefix+defaultKey)
		})
	}
}

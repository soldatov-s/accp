package rrdata

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
	defaultKey = "test"
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

func initConfig() *external.Config {
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
	cfg := initConfig()

	client := initExternal(ctx, t, dsn)
	c := external.NewCache(ctx, cfg, client)

	return c
}
func initNewRefreshData(cache *external.Cache) *RefreshData {
	return NewRefreshData(defaultKey, 3, cache)
}
func TestNewRefreshData(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initCache(t, dsn)
	require.NotNil(t, c)

	d := initNewRefreshData(c)
	require.NotNil(t, d)
}

func TestInc(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initCache(t, dsn)
	require.NotNil(t, c)

	d := initNewRefreshData(c)
	require.NotNil(t, d)

	// Test new counter
	err = d.Inc()
	require.Nil(t, err)

	var cnt int64
	err = c.GetLimit(defaultRefreshPrefix+defaultKey, &cnt)
	require.Nil(t, err)
	require.Equal(t, int64(1), cnt)

	// Test after create
	err = d.Inc()
	require.Nil(t, err)
}

func TestCurrent(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initCache(t, dsn)
	require.NotNil(t, c)

	d := initNewRefreshData(c)
	require.NotNil(t, d)

	err = d.Inc()
	require.Nil(t, err)

	cnt := d.Current()
	require.Equal(t, int(1), cnt)
}

func TestCheck(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initCache(t, dsn)
	require.NotNil(t, c)

	d := initNewRefreshData(c)
	require.NotNil(t, d)

	err = d.Inc()
	require.Nil(t, err)
	t.Log("current", d.counter, "max", d.maxCount)

	res := d.Check()
	require.True(t, res)

	err = d.Inc()
	require.Nil(t, err)
	t.Log("current", d.counter, "max", d.maxCount)

	res = d.Check()
	require.True(t, res)

	err = d.Inc()
	require.Nil(t, err)
	t.Log("current", d.counter, "max", d.maxCount)

	res = d.Check()
	require.False(t, res)

	err = d.Inc()
	require.Nil(t, err)
	t.Log("current", d.counter, "max", d.maxCount)

	// conter equal max
	res = d.Check()
	require.False(t, res)

	err = d.Inc()
	require.Nil(t, err)
	t.Log("current", d.counter, "max", d.maxCount)

	res = d.Check()
	require.True(t, res)
}

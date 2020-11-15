package externalcache_test

import (
	"testing"
	"time"

	context "github.com/soldatov-s/accp/internal/ctx"
	externalcache "github.com/soldatov-s/accp/internal/redis"
	"github.com/soldatov-s/accp/x/dockertest"
	testhelpers "github.com/soldatov-s/accp/x/test_helpers"
	resilience "github.com/soldatov-s/accp/x/test_helpers/resilence"
	"github.com/stretchr/testify/require"
)

func TestNewRedisClient(t *testing.T) {
	err := testhelpers.LoadTestYAML()
	require.Nil(t, err)

	lc, err := testhelpers.LoadTestConfigLogger()
	require.Nil(t, err)

	ctx := context.NewContext()
	ctx.InitilizeLogger(lc)

	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

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
}

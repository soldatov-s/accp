package redis

// import (
// 	"context"
// 	"testing"
// 	"time"

// 	"github.com/soldatov-s/accp/x/dockertest"
// 	testhelpers "github.com/soldatov-s/accp/x/test_helpers"
// 	resilience "github.com/soldatov-s/accp/x/test_helpers/resilence"
// 	"github.com/stretchr/testify/require"
// )

// func TestNewRedisClient(t *testing.T) {
// 	err := testhelpers.LoadTestYAML()
// 	require.Nil(t, err)

// 	ctx := context.Background()

// 	dsn, err := dockertest.RunRedis()
// 	require.Nil(t, err)
// 	defer dockertest.KillAllDockers()

// 	t.Logf("Connecting to redis: %s", dsn)

// 	ec := &Config{
// 		DSN:                   dsn,
// 		MinIdleConnections:    10,
// 		MaxOpenedConnections:  30,
// 		MaxConnectionLifetime: 30 * time.Second,
// 	}

// 	var externalStorage *RedisClient
// 	err = resilience.Retry(
// 		t,
// 		time.Second*5,
// 		time.Minute*5,
// 		func() (err error) {
// 			externalStorage, err = NewRedisClient(ctx, ec)
// 			return err
// 		},
// 	)
// 	require.Nil(t, err)
// 	require.NotNil(t, externalStorage)
// 	t.Logf("Connected to redis: %s", dsn)
// }

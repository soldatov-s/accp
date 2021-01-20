package redis

import (
	"context"
	"testing"

	"github.com/soldatov-s/accp/x/dockertest"
	"github.com/stretchr/testify/require"
)

func TestRegistrate(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()

	cfg.DSN = dsn

	ctx, err = Registrate(ctx, cfg)
	require.Nil(t, err)
	require.NotNil(t, ctx)
}

func TestGet(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()

	cfg.DSN = dsn

	ctx, err = Registrate(ctx, cfg)
	require.Nil(t, err)
	require.NotNil(t, ctx)

	c := Get(ctx)
	require.NotNil(t, c)
}

package introspection

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistrate(t *testing.T) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()

	ctx, err := Registrate(ctx, cfg)
	require.Nil(t, err)
	require.NotNil(t, ctx)
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()

	ctx, err := Registrate(ctx, cfg)
	require.Nil(t, err)
	require.NotNil(t, ctx)

	c := Get(ctx)
	require.NotNil(t, c)
}

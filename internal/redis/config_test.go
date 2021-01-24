package redis

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testDSN = "redis://redis:6379"
)

func TestSetDefault(t *testing.T) {
	c := &Config{}
	c.SetDefault()
	require.Equal(t, defaultMaxConnLifetime, c.MaxConnectionLifetime)
	require.Equal(t, defaultMaxOpenedConnections, c.MaxOpenedConnections)
	require.Equal(t, defaultMinIdleConnections, c.MinIdleConnections)
}

func TestOptions(t *testing.T) {
	c := &Config{}
	c.SetDefault()
	opt, err := c.Options()
	require.Nil(t, opt)
	require.NotNil(t, err)

	c.DSN = testDSN
	opt, err = c.Options()
	require.NotNil(t, opt)
	require.Nil(t, err)
}

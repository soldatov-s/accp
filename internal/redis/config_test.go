package redis

import (
	"testing"

	"github.com/stretchr/testify/require"
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
	require.NotNil(t, opt)
	require.Nil(t, err)

	c.DSN = ""
	opt, err = c.Options()
	require.Nil(t, opt)
	require.NotNil(t, err)
}

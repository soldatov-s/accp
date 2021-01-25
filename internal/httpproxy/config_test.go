package httpproxy

import (
	"testing"

	"github.com/soldatov-s/accp/internal/errors"
	"github.com/stretchr/testify/require"
)

func TestSetDefault(t *testing.T) {
	c := Config{}
	c.SetDefault()
	require.Equal(t, defaultListen, c.Listen)
}

func TestValidate(t *testing.T) {
	c := Config{}
	err := c.Validate()
	require.NotNil(t, err)
	require.Equal(t, errors.EmptyConfigParameter("routes"), err)
}

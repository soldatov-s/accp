package cfg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	cfg := &Configuration{}

	err := LoadTestYAML()
	require.Nil(t, err)

	err = cfg.parse()
	require.Nil(t, err)

	require.Nil(t, cfg.Admin)
	require.Nil(t, cfg.Rabbitmq)
	require.Nil(t, cfg.Redis)

	t.Logf("Test config: %+v", cfg)
}

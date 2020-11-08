package cfg_test

import (
	"testing"

	"github.com/soldatov-s/accp/internal/cfg"
	testHelpers "github.com/soldatov-s/accp/x/test_helpers"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	cfg := &cfg.Configuration{}

	err := testHelpers.LoadTestYAML()
	require.Nil(t, err)

	err = cfg.Parse()
	require.Nil(t, err)

	require.Nil(t, cfg.Admin)
	require.Nil(t, cfg.Rabbitmq)
	require.Nil(t, cfg.Redis)

	t.Logf("Test config: %+v", cfg)
}

package cfg_test

import (
	"testing"

	"github.com/soldatov-s/accp/internal/cfg"
	testHelpers "github.com/soldatov-s/accp/x/test_helpers"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	config := &cfg.Configuration{}

	err := testHelpers.LoadTestYAML()
	require.Nil(t, err)

	err = config.Parse()
	require.Nil(t, err)

	require.Nil(t, config.Admin)
	require.Nil(t, config.Rabbitmq)
	require.Nil(t, config.Redis)

	t.Logf("Test config: %+v", config)
}

package routes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortKeys(t *testing.T) {
	mc := make(MapConfig)
	mc["a_key"] = &Config{}
	mc["az_key"] = &Config{}
	mc["z_key"] = &Config{}
	mc["c_key"] = &Config{}
	keys := mc.SortKeys()
	require.Equal(t, []string{"a_key", "az_key", "c_key", "z_key"}, keys)
}

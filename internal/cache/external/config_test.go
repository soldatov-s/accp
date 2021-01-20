package external

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSetDefault(t *testing.T) {
	c := &Config{}
	c.SetDefault()
	require.Equal(t, defaultTTL, c.TTL)
	require.Equal(t, defaultTTLErr, c.TTLErr)
	require.Equal(t, defaultKeyPrefix, c.KeyPrefix)
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name           string
		srcConfig      *Config
		targetConfig   *Config
		expectedConfig *Config
	}{
		{
			name:           "src is nil",
			srcConfig:      nil,
			targetConfig:   &Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second, KeyPrefix: "accp_"},
			expectedConfig: &Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second, KeyPrefix: "accp_"},
		},
		{
			name:           "target is nil",
			srcConfig:      &Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second, KeyPrefix: "accp_"},
			targetConfig:   nil,
			expectedConfig: &Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second, KeyPrefix: "accp_"},
		},
		{
			name:           "target is not nil",
			srcConfig:      &Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second, KeyPrefix: "accp_"},
			targetConfig:   &Config{TTL: 5 * time.Second, TTLErr: 10 * time.Second, KeyPrefix: "accp2_"},
			expectedConfig: &Config{TTL: 5 * time.Second, TTLErr: 10 * time.Second, KeyPrefix: "accp2_"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cc := tt.srcConfig.Merge(tt.targetConfig)
			require.Equal(t, tt.expectedConfig, cc)
		})
	}
}

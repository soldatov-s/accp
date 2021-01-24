package httpclient

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSetDefault(t *testing.T) {
	c := &Config{}
	c.SetDefault()
	require.Equal(t, defaultSize, c.Size)
	require.Equal(t, defaultTimeout, c.Timeout)
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
			targetConfig:   &Config{Size: 1, Timeout: 2 * time.Second},
			expectedConfig: &Config{Size: 1, Timeout: 2 * time.Second},
		},
		{
			name:           "target is nil",
			srcConfig:      &Config{Size: 1, Timeout: 2 * time.Second},
			targetConfig:   nil,
			expectedConfig: &Config{Size: 1, Timeout: 2 * time.Second},
		},
		{
			name:           "target is not nil",
			srcConfig:      &Config{Size: 1, Timeout: 2 * time.Second},
			targetConfig:   &Config{Size: 5, Timeout: 5 * time.Second},
			expectedConfig: &Config{Size: 5, Timeout: 5 * time.Second},
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

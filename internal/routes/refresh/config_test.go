package refresh

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSetDefault(t *testing.T) {
	c := &Config{}
	c.SetDefault()
	require.Equal(t, defaultMaxCount, c.MaxCount)
	require.Equal(t, defaultTime, c.Time)
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
			targetConfig:   &Config{MaxCount: 1, Time: 2 * time.Second},
			expectedConfig: &Config{MaxCount: 1, Time: 2 * time.Second},
		},
		{
			name:           "target is nil",
			srcConfig:      &Config{MaxCount: 1, Time: 2 * time.Second},
			targetConfig:   nil,
			expectedConfig: &Config{MaxCount: 1, Time: 2 * time.Second},
		},
		{
			name:           "target is not nil",
			srcConfig:      &Config{MaxCount: 1, Time: 2 * time.Second},
			targetConfig:   &Config{MaxCount: 5, Time: 10 * time.Second},
			expectedConfig: &Config{MaxCount: 5, Time: 10 * time.Second},
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

package limits

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSetDefault(t *testing.T) {
	c := &Config{}
	c.SetDefault()
	require.Equal(t, defaultCounter, c.MaxCounter)
	require.Equal(t, defaultTTL, c.TTL)
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
			targetConfig:   &Config{TTL: 1 * time.Second, MaxCounter: 1, Cookie: []string{"test1", "test2"}, Header: []string{"test1", "test2"}},
			expectedConfig: &Config{TTL: 1 * time.Second, MaxCounter: 1, Cookie: []string{"test1", "test2"}, Header: []string{"test1", "test2"}},
		},
		{
			name:           "target is nil",
			srcConfig:      &Config{TTL: 1 * time.Second, MaxCounter: 1, Cookie: []string{"test1", "test2"}, Header: []string{"test1", "test2"}},
			targetConfig:   nil,
			expectedConfig: &Config{TTL: 1 * time.Second, MaxCounter: 1, Cookie: []string{"test1", "test2"}, Header: []string{"test1", "test2"}},
		},
		{
			name:         "target is not nil",
			srcConfig:    &Config{TTL: 1 * time.Second, MaxCounter: 1, Cookie: []string{"test1", "test2"}, Header: []string{"test1", "test2"}},
			targetConfig: &Config{TTL: 2 * time.Second, MaxCounter: 2, Cookie: []string{"test3"}, Header: []string{"test3"}},
			expectedConfig: &Config{
				TTL:        2 * time.Second,
				MaxCounter: 2,
				Cookie:     []string{"test1", "test2", "test3"},
				Header:     []string{"test1", "test2", "test3"},
			},
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

func TestNewMapConfig(t *testing.T) {
	c := NewMapConfig()
	require.NotNil(t, c)
}

func TestMapConfig_SetDefault(t *testing.T) {
	c := NewMapConfig()
	c.SetDefault()
	require.NotNil(t, c)
	_, ok := c["token"]
	require.True(t, ok)
	_, ok = c["ip"]
	require.True(t, ok)
}

func TestMapConfig_Merge(t *testing.T) {
	tests := []struct {
		name           string
		srcConfig      MapConfig
		targetConfig   MapConfig
		expectedConfig MapConfig
	}{
		{
			name:           "src is nil",
			srcConfig:      nil,
			targetConfig:   MapConfig{"test1": &Config{Header: []string{"TEST1"}}, "test2": &Config{Header: []string{"TEST2"}}},
			expectedConfig: MapConfig{"test1": &Config{Header: []string{"TEST1"}}, "test2": &Config{Header: []string{"TEST2"}}},
		},
		{
			name:           "target is nil",
			srcConfig:      MapConfig{"test1": &Config{Header: []string{"TEST1"}}, "test2": &Config{Header: []string{"TEST2"}}},
			targetConfig:   nil,
			expectedConfig: MapConfig{"test1": &Config{Header: []string{"TEST1"}}, "test2": &Config{Header: []string{"TEST2"}}},
		},
		{
			name:         "target is not nil",
			srcConfig:    MapConfig{"test1": &Config{Header: []string{"TEST1"}}, "test2": &Config{Header: []string{"TEST2"}}},
			targetConfig: MapConfig{"test1": &Config{Header: []string{"TEST2"}}, "test3": &Config{Header: []string{"TEST3"}}},
			expectedConfig: MapConfig{
				"test1": &Config{Header: []string{"TEST1", "TEST2"}},
				"test2": &Config{Header: []string{"TEST2"}},
				"test3": &Config{Header: []string{"TEST3"}},
			},
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

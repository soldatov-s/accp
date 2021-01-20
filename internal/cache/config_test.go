package cache

import (
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cache/memory"
	"github.com/stretchr/testify/require"
)

func TestSetDefault(t *testing.T) {
	tests := []struct {
		name           string
		memConfig      *memory.Config
		extConfig      *external.Config
		expectedConfig *Config
	}{
		{
			name:      "all config exists",
			memConfig: &memory.Config{},
			extConfig: &external.Config{},
			expectedConfig: func() *Config {
				cfg := &Config{}
				cfg.Memory = &memory.Config{}
				cfg.Memory.SetDefault()
				cfg.External = &external.Config{}
				cfg.External.SetDefault()
				return cfg
			}(),
		},
		{
			name:      "external config is nil",
			memConfig: &memory.Config{},
			extConfig: nil,
			expectedConfig: func() *Config {
				cfg := &Config{}
				cfg.Memory = &memory.Config{}
				cfg.Memory.SetDefault()
				return cfg
			}(),
		},
		{
			name:      "memory and external config is nil",
			memConfig: nil,
			extConfig: nil,
			expectedConfig: func() *Config {
				cfg := &Config{}
				cfg.Memory = &memory.Config{}
				cfg.Memory.SetDefault()
				return cfg
			}(),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Memory:   tt.memConfig,
				External: tt.extConfig,
			}

			cfg.SetDefault()
			require.Equal(t, tt.expectedConfig, cfg)
		})
	}
}

// nolint : dupl
func TestMerge(t *testing.T) {
	tests := []struct {
		name           string
		srcConfig      *Config
		targetConfig   *Config
		expectedConfig *Config
	}{
		{
			name: "target is nil",
			srcConfig: &Config{
				Disabled: false,
				Memory:   &memory.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second},
				External: &external.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second, KeyPrefix: "accp_"},
			},
			targetConfig: nil,
			expectedConfig: &Config{
				Disabled: false,
				Memory:   &memory.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second},
				External: &external.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second, KeyPrefix: "accp_"},
			},
		},
		{
			name: "target is not nil, external config in target is nil",
			srcConfig: &Config{
				Disabled: false,
				Memory:   &memory.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second},
				External: &external.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second, KeyPrefix: "accp_"},
			},
			targetConfig: &Config{
				Disabled: false,
				Memory:   &memory.Config{TTL: 5 * time.Second, TTLErr: 10 * time.Second},
				External: nil,
			},
			expectedConfig: &Config{
				Disabled: false,
				Memory:   &memory.Config{TTL: 5 * time.Second, TTLErr: 10 * time.Second},
				External: &external.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second, KeyPrefix: "accp_"},
			},
		},
		{
			name: "target is not nil, but disabled",
			srcConfig: &Config{
				Disabled: false,
				Memory:   &memory.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second},
				External: &external.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second, KeyPrefix: "accp_"},
			},
			targetConfig: &Config{
				Disabled: true,
				Memory:   &memory.Config{TTL: 5 * time.Second, TTLErr: 10 * time.Second},
				External: &external.Config{TTL: 5 * time.Second, TTLErr: 10 * time.Second, KeyPrefix: "accp_"},
			},
			expectedConfig: &Config{
				Disabled: true,
				Memory:   &memory.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second},
				External: &external.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second, KeyPrefix: "accp_"},
			},
		},
		{
			name: "target is not nil",
			srcConfig: &Config{
				Disabled: false,
				Memory:   &memory.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second},
				External: &external.Config{TTL: 1 * time.Second, TTLErr: 2 * time.Second, KeyPrefix: "accp_"},
			},
			targetConfig: &Config{
				Disabled: false,
				Memory:   &memory.Config{TTL: 5 * time.Second, TTLErr: 10 * time.Second},
				External: &external.Config{TTL: 5 * time.Second, TTLErr: 10 * time.Second, KeyPrefix: "accp_"},
			},
			expectedConfig: &Config{
				Disabled: false,
				Memory:   &memory.Config{TTL: 5 * time.Second, TTLErr: 10 * time.Second},
				External: &external.Config{TTL: 5 * time.Second, TTLErr: 10 * time.Second, KeyPrefix: "accp_"},
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

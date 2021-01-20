package rabbitmq

import (
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/errors"
	"github.com/stretchr/testify/require"
)

func TestSetDefault(t *testing.T) {
	c := &Config{}
	c.SetDefault()
	require.Equal(t, defaultBackoffPolicy(), c.BackoffPolicy)
	require.Equal(t, defaultExchangeName, c.ExchangeName)
}

func TestValidate(t *testing.T) {
	c := &Config{}
	err := c.Validate()
	require.Equal(t, errors.EmptyConfigParameter("dsn"), err)

	c.DSN = "amqp://guest:guest@rabbitmq:5672"
	err = c.Validate()
	require.Nil(t, err)
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
			targetConfig:   &Config{DSN: "test", ExchangeName: "test", BackoffPolicy: []time.Duration{2 * time.Second}},
			expectedConfig: &Config{DSN: "test", ExchangeName: "test", BackoffPolicy: []time.Duration{2 * time.Second}},
		},
		{
			name:           "target is nil",
			srcConfig:      &Config{DSN: "test", ExchangeName: "test", BackoffPolicy: []time.Duration{2 * time.Second}},
			targetConfig:   nil,
			expectedConfig: &Config{DSN: "test", ExchangeName: "test", BackoffPolicy: []time.Duration{2 * time.Second}},
		},
		{
			name:           "target is not nil",
			srcConfig:      &Config{DSN: "test", ExchangeName: "test", BackoffPolicy: []time.Duration{2 * time.Second}},
			targetConfig:   &Config{DSN: "test2", ExchangeName: "test2", BackoffPolicy: []time.Duration{5 * time.Second}},
			expectedConfig: &Config{DSN: "test2", ExchangeName: "test2", BackoffPolicy: []time.Duration{5 * time.Second}},
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

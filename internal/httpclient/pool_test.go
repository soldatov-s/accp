package httpclient

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testSize    = 5
	testTimeout = 3 * time.Second
)

func initConfig() *Config {
	return &Config{
		Size:    testSize,
		Timeout: testTimeout,
	}
}

func TestNewPool(t *testing.T) {
	cfg := initConfig()
	p := NewPool(cfg)
	require.NotNil(t, p)
}

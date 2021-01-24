package httpclient

import (
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func initTransport(c *Config) *http.Transport {
	dialer := &net.Dialer{
		Timeout: c.Timeout,
	}

	return &http.Transport{
		MaxIdleConns:          1024,
		MaxIdleConnsPerHost:   1024,
		Dial:                  dialer.Dial,
		TLSHandshakeTimeout:   c.Timeout,
		ExpectContinueTimeout: c.Timeout,
		IdleConnTimeout:       c.Timeout,
		ResponseHeaderTimeout: c.Timeout,
	}
}
func TestNewPoolClient(t *testing.T) {
	cfg := initConfig()
	transport := initTransport(cfg)
	c := NewPoolClient(cfg.Timeout, transport)
	require.NotNil(t, c)
}

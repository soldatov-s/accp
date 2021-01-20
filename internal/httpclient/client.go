package httpclient

import (
	"net/http"
	"time"
)

const (
	defaultClientTimeout = 60 * time.Second
)

func NewPoolClient(timeout time.Duration, netTransport http.RoundTripper) *http.Client {
	clientTimeout := defaultClientTimeout
	if timeout > 0 {
		clientTimeout = timeout
	}

	return &http.Client{
		Transport: netTransport,
		Timeout:   clientTimeout,
	}
}

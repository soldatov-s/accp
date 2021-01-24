package httpclient

import (
	"net/http"
	"time"
)

func NewPoolClient(timeout time.Duration, netTransport http.RoundTripper) *http.Client {
	clientTimeout := defaultTimeout
	if timeout > 0 {
		clientTimeout = timeout
	}

	return &http.Client{
		Transport: netTransport,
		Timeout:   clientTimeout,
	}
}

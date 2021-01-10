package httpclient

import (
	"net/http"
	"time"
)

type Client struct {
	*http.Client
}

const (
	defaultClientTimeout = 60 * time.Second
)

func NewPoolClient(timeout time.Duration, netTransport http.RoundTripper) *Client {
	clientTimeout := defaultClientTimeout
	if timeout > 0 {
		clientTimeout = timeout
	}

	return &Client{
		Client: &http.Client{
			Transport: netTransport,
			Timeout:   clientTimeout,
		},
	}
}

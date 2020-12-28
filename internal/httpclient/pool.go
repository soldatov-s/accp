package httpclient

import (
	"net"
	"net/http"
	"time"
)

type Pool struct {
	ch           chan *Client
	netTransport *http.Transport
}

func NewPool(size int, timeout time.Duration) *Pool {
	p := &Pool{}
	p.ch = make(chan *Client, size)

	clientTimeout := defaultClientTimeout
	if timeout > 0 {
		clientTimeout = timeout
	}

	dialer := &net.Dialer{
		Timeout: clientTimeout,
	}

	p.netTransport = &http.Transport{
		MaxIdleConns:          1024,
		MaxIdleConnsPerHost:   1024,
		Dial:                  dialer.Dial,
		TLSHandshakeTimeout:   clientTimeout,
		ExpectContinueTimeout: clientTimeout,
		IdleConnTimeout:       clientTimeout,
		ResponseHeaderTimeout: clientTimeout,
	}

	for i := 0; i < size; i++ {
		p.ch <- NewPoolClient(timeout, p.netTransport)
	}
	return p
}

func (p *Pool) GetFromPool() *Client {
	return <-p.ch
}

func (p *Pool) PutToPool(client *Client) {
	go func() {
		p.ch <- client
	}()
}

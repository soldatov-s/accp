package httpclient

import (
	"net"
	"net/http"
)

type Pool struct {
	ch           chan *http.Client
	netTransport *http.Transport
}

func NewPool(poolCfg *Config) *Pool {
	p := &Pool{}
	p.ch = make(chan *http.Client, poolCfg.Size)

	clientTimeout := defaultClientTimeout
	if poolCfg.Timeout > 0 {
		clientTimeout = poolCfg.Timeout
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

	for i := 0; i < poolCfg.Size; i++ {
		p.ch <- NewPoolClient(poolCfg.Timeout, p.netTransport)
	}
	return p
}

func (p *Pool) GetFromPool() *http.Client {
	return <-p.ch
}

func (p *Pool) PutToPool(client *http.Client) {
	go func() {
		p.ch <- client
	}()
}

package httpclient

import (
	"net"
	"net/http"
)

type Pool struct {
	ch           chan *http.Client
	netTransport *http.Transport
}

func NewPool(cfg *Config) *Pool {
	p := &Pool{}
	p.ch = make(chan *http.Client, cfg.Size)

	cfg.SetDefault()

	dialer := &net.Dialer{
		Timeout: cfg.Timeout,
	}

	p.netTransport = &http.Transport{
		MaxIdleConns:          1024,
		MaxIdleConnsPerHost:   1024,
		Dial:                  dialer.Dial,
		TLSHandshakeTimeout:   cfg.Timeout,
		ExpectContinueTimeout: cfg.Timeout,
		IdleConnTimeout:       cfg.Timeout,
		ResponseHeaderTimeout: cfg.Timeout,
	}

	for i := 0; i < cfg.Size; i++ {
		p.ch <- NewPoolClient(cfg.Timeout, p.netTransport)
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

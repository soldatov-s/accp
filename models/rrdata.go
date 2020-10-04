package models

import (
	"sync"

	"github.com/soldatov-s/accp/internal/httpclient"
)

// RRData contains request and response for it
type RRData struct {
	Response *Response
	Request  *Request
	Refresh  struct {
		Mu       sync.Mutex
		MaxCount int
		Counter  int
	}
}

func (r *RRData) MarshalBinary() (data []byte, err error) {
	return r.Response.MarshalBinary()
}

func (r *RRData) UnmarshalBinary(data []byte) error {
	return r.Response.UnmarshalBinary(data)
}

func (r *RRData) Update(client *httpclient.Client) error {
	req, err := r.Request.BuildRequest()
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	return r.Response.Read(resp)
}

package models

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/httpclient"
)

// RRData contains request and response for it
type RRData struct {
	Response *Response
	Request  *Request
	UUID     string
	Refresh  struct {
		mu       sync.Mutex
		MaxCount int
		Counter  int
	}
}

func NewRRData() *RRData {
	return &RRData{
		Response: &Response{},
		Request:  &Request{},
		UUID:     uuid.New().String(),
	}
}

func (r *RRData) MarshalBinary() (data []byte, err error) {
	return json.Marshal(r)
}

func (r *RRData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}

func (r *RRData) GetStatusCode() int {
	return r.Response.StatusCode
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

	err = r.Response.Read(resp)
	if err != nil {
		return err
	}

	r.UUID = uuid.New().String()

	return nil
}

func (r *RRData) LoadRefreshCounter(hk string, externalStorage *external.Cache) error {
	if externalStorage != nil {
		if err := externalStorage.JSONGet(hk, "RefreshCounter", &r.Refresh.Counter); err != nil {
			return err
		}
	}

	return nil
}

func (r *RRData) UpdateRefreshCounter(hk string, externalStorage *external.Cache) error {
	if externalStorage != nil {
		data, err := json.Marshal(&r.Refresh.Counter)
		if err != nil {
			return err
		}

		if err := externalStorage.JSONSet(hk, "RefreshCounter", string(data)); err != nil {
			return err
		}
	}

	return nil
}

func (r *RRData) MuLock() {
	r.Refresh.mu.Lock()
}

func (r *RRData) MuUnlock() {
	r.Refresh.mu.Unlock()
}

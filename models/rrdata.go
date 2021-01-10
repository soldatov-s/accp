package models

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/httpclient"
)

type Refresh struct {
	mu       sync.Mutex
	MaxCount int
	Counter  int
}

// RData contains response for it
type RData struct {
	Response *Response
	UUID     uuid.UUID
	Refresh  *Refresh
}

func (r *RData) MarshalBinary() (data []byte, err error) {
	return json.Marshal(r)
}

func (r *RData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}

// RRData contains request and response for it
type RRData struct {
	RData
	Request *Request
}

func NewRRData() *RRData {
	return &RRData{
		RData: RData{
			Response: &Response{},
			UUID:     uuid.New(),
		},
		Request: &Request{},
	}
}

func (r *RRData) MarshalBinary() (data []byte, err error) {
	return json.Marshal(&r.RData)
}

func (r *RRData) UnmarshalBinary(data []byte) error {
	rData := RData{}
	if err := json.Unmarshal(data, &rData); err != nil {
		return err
	}
	r.RData.Response = rData.Response
	r.RData.UUID = rData.UUID
	r.RData.Refresh = rData.Refresh
	return nil
}

func (r *RRData) GetStatusCode() int {
	return r.Response.StatusCode
}

func (r *RRData) Update(client *httpclient.Client) error {
	req, err := r.Request.BuildRequest()
	if err != nil {
		return err
	}

	return r.UpdateByRequest(client, req)
}

func (r *RRData) UpdateByRequest(client *httpclient.Client, req *http.Request) error {
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	err = r.Response.Read(resp)
	if err != nil {
		return err
	}

	r.UUID = uuid.New()

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

func (r *RRData) ReadAll(req *http.Request, resp *http.Response) error {
	if err := r.Request.Read(req); err != nil {
		return errors.Wrap(err, "failed to read data from request")
	}

	if err := r.Response.Read(resp); err != nil {
		return errors.Wrap(err, "failed to read data from response")
	}

	return nil
}

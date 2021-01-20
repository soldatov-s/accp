package rrdata

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/pkg/errors"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/httputils"
)

// RequestResponseData contains request and response for it
type RequestResponseData struct {
	Response *ResponseData
	Request  *RequestData
	Mu       sync.RWMutex
}

func NewRequestResponseData(hk string, maxCount int, cache *external.Cache) *RequestResponseData {
	return &RequestResponseData{
		Response: NewResponseData(hk, maxCount, cache),
		Request:  &RequestData{},
	}
}

func (r *RequestResponseData) MarshalBinary() (data []byte, err error) {
	return json.Marshal(&r.Response)
}

func (r *RequestResponseData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &r.Response)
}

func (r *RequestResponseData) GetStatusCode() int {
	return r.Response.StatusCode
}

func (r *RequestResponseData) Update(client *http.Client) error {
	req, err := r.Request.BuildRequest()
	if err != nil {
		return err
	}

	return r.UpdateByRequest(client, req)
}

func (r *RequestResponseData) UpdateByRequest(client *http.Client, req *http.Request) error {
	// nolint
	resp, err := client.Do(req)
	if err != nil {
		resp = httputils.ErrResponse(err.Error(), http.StatusServiceUnavailable)
	}
	defer resp.Body.Close()

	err = r.Response.Read(resp)
	if err != nil {
		return err
	}

	return nil
}

func (r *RequestResponseData) ReadAll(req *http.Request, resp *http.Response) error {
	if err := r.Request.Read(req); err != nil {
		return errors.Wrap(err, "failed to read data from request")
	}

	if err := r.Response.Read(resp); err != nil {
		return errors.Wrap(err, "failed to read data from response")
	}

	return nil
}

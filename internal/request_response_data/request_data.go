package rrdata

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/soldatov-s/accp/internal/httputils"
	"github.com/valyala/bytebufferpool"
)

// RequestData describes structure for holding information about request
type RequestData struct {
	URL    string
	Method string
	Body   string
	Header http.Header
	mu     sync.RWMutex
}

func NewRequestData(req *http.Request) (*RequestData, error) {
	r := &RequestData{}
	if err := r.Read(req); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *RequestData) MarshalBinary() (data []byte, err error) {
	return json.Marshal(r)
}

func (r *RequestData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}

func (r *RequestData) Read(req *http.Request) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	if req.Body != nil {
		_, err := io.Copy(buf, req.Body)
		if err != nil {
			return err
		}
	}

	req.Body = ioutil.NopCloser(bytes.NewReader(buf.Bytes()))
	r.Body = buf.String()
	r.Header = make(http.Header)
	httputils.CopyHeader(r.Header, req.Header)
	r.Method = req.Method
	r.URL = req.URL.String()

	return nil
}

func (r *RequestData) BuildRequest() (*http.Request, error) {
	if r == nil {
		return nil, ErrEmptyRequest
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	req, err := http.NewRequest(r.Method, r.URL, bytes.NewBufferString(r.Body))
	if err != nil {
		return nil, err
	}

	httputils.CopyHeader(req.Header, r.Header)
	return req, nil
}

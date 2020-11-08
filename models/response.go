package models

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/soldatov-s/accp/internal/httputils"
)

type Response struct {
	readMu     sync.RWMutex
	Body       string
	Header     http.Header
	StatusCode int
}

func (r *Response) MarshalBinary() (data []byte, err error) {
	return json.Marshal(r)
}

func (r *Response) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}

func (r *Response) Write(w http.ResponseWriter) error {
	r.readMu.RLock()
	defer r.readMu.RUnlock()

	httputils.CopyHeader(w.Header(), r.Header)
	w.WriteHeader(r.StatusCode)
	_, err := w.Write([]byte(r.Body))
	return err
}

func (r *Response) Read(resp *http.Response) error {
	r.readMu.Lock()
	defer r.readMu.Unlock()

	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, resp.Body)
	if err != nil {
		return err
	}

	r.Body = buf.String()
	r.Header = make(http.Header)
	httputils.CopyHeader(r.Header, resp.Header)
	r.StatusCode = resp.StatusCode

	return nil
}

package models

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/soldatov-s/accp/internal/httputils"
)

type Request struct {
	URL    string
	Method string
	Body   string
	Header http.Header
	mu     sync.Mutex
}

func (r *Request) Read(req *http.Request) error {
	if r == nil {
		r = &Request{}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, req.Body)
	if err != nil {
		return err
	}

	r.Body = buf.String()
	r.Header = make(http.Header)
	httputils.CopyHeader(r.Header, req.Header)
	r.Method = req.Method
	r.URL = req.URL.String()

	return nil
}

func (r *Request) BuildRequest() (*http.Request, error) {
	if r == nil {
		return nil, errors.New("empty request")
	}
	return http.NewRequest(r.Method, r.URL, bytes.NewBufferString(r.Body))
}

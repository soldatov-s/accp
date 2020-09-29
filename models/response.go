package models

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/soldatov-s/accp/internal/httputils"
)

type ResponseData struct {
	Body       string
	Header     http.Header
	StatusCode int
}

func (r *ResponseData) MarshalBinary() (data []byte, err error) {
	return json.Marshal(r)
}

func (r *ResponseData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}

func (r *ResponseData) Write(w http.ResponseWriter) error {
	httputils.CopyHeader(w.Header(), r.Header)
	w.WriteHeader(r.StatusCode)
	_, err := w.Write([]byte(r.Body))
	return err
}

func (r *ResponseData) Read(resp *http.Response) error {
	if r == nil {
		r = &ResponseData{}
	}

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

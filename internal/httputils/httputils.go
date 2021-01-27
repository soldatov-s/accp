package httputils

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/valyala/bytebufferpool"
)

const (
	RequestIDHeader = "x-request-id"
)

func CopyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func CopyHTTPResponse(w http.ResponseWriter, resp *http.Response) error {
	CopyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	_, err := io.Copy(w, resp.Body)
	return err
}

func HashRequest(r *http.Request) (string, error) {
	hashedString := r.URL.RequestURI() + r.Method

	var buf *bytebufferpool.ByteBuffer
	if r.Body != nil {
		buf = bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

		_, err := io.Copy(buf, r.Body)
		if err != nil {
			return "", err
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(buf.Bytes()))
		hashedString += buf.String()
	}
	sum := sha256.New().Sum([]byte(hashedString))
	return base64.URLEncoding.EncodeToString(sum), nil
}

func ErrResponse(errormsg string, code int) *http.Response {
	resp := &http.Response{
		StatusCode: code,
		Body:       ioutil.NopCloser(bytes.NewBufferString(errormsg)),
	}

	resp.Header = make(http.Header)
	resp.Header.Set("Content-Type", "text/plain; charset=utf-8")
	resp.Header.Set("X-Content-Type-Options", "nosniff")

	return resp
}

func GetRequestID(r *http.Request) string {
	return r.Header.Get(RequestIDHeader)
}

func CopyRequestWithDSN(req *http.Request, dsn string) (*http.Request, error) {
	var (
		proxyReq *http.Request
		err      error
	)
	if req.Body != nil {
		var body []byte
		body, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
		proxyReq, err = http.NewRequest(req.Method, dsn+req.URL.String(), bytes.NewReader(body))
	} else {
		proxyReq, err = http.NewRequest(req.Method, dsn+req.URL.String(), nil)
	}
	if err != nil {
		return nil, err
	}

	CopyHeader(proxyReq.Header, req.Header)

	return proxyReq, nil
}

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
	var sum []byte
	introspectBody := r.Header.Get("accp-introspect-body")

	var buf *bytebufferpool.ByteBuffer
	if r.Body != nil {
		buf = bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

		_, err := io.Copy(buf, r.Body)
		if err != nil {
			return "", err
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(buf.Bytes()))
		sum = sha256.New().Sum([]byte(r.URL.RequestURI() + buf.String() + introspectBody))
	} else {
		sum = sha256.New().Sum([]byte(r.URL.RequestURI() + introspectBody))
	}

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

package httputils

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
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
	buf := bytebufferpool.Get()
	if r.Body != nil {
		if _, err := io.Copy(buf, r.Body); err != nil {
			return "", err
		}
	}

	introspectBody := r.Header.Get("accp-introspect-body")

	sum := sha256.New().Sum([]byte(r.URL.RequestURI() + buf.String() + introspectBody))
	return base64.URLEncoding.EncodeToString(sum), nil
}

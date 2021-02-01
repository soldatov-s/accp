package testproxyhelpers

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	ContentTypeFormUrlencoded = "application/x-www-form-urlencoded"
)

func FakeService(t *testing.T, host string, handler http.Handler) *httptest.Server {
	listener, err := net.Listen("tcp", host)
	if err != nil {
		t.Fatal(fmt.Sprintf("httptest: failed to listen on a port: %v", err))
	}

	return &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: handler},
	}
}

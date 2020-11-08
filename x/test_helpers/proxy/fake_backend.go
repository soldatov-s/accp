package testproxyhelpers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	DefaultFakeSericeHost = "localhost:10000"
	DefaultGetAnswer      = `{"result":{ "answer" : "it's a get request"}}`
	DefaultPostAnswer     = `{"result":{ "answer" : "it's a post request"}}`
	DefaultPutAnswer      = `{"result":{ "answer" : "it's a put request"}}`
)

func getRequest(_ *http.Request) (res []byte, err error) {
	return []byte(DefaultGetAnswer), nil
}

func postRequest(_ *http.Request) (res []byte, err error) {
	return []byte(DefaultPostAnswer), nil
}

func putRequest(_ *http.Request) (res []byte, err error) {
	return []byte(DefaultPutAnswer), nil
}

func FakeBackendService(t *testing.T, host string) *httptest.Server {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var (
			err error
			res []byte
		)

		switch r.Method {
		case http.MethodGet:
			switch r.URL.Path {
			// case "/api/v1/users/search":
			// 	fallthrough
			default:
				res, err = getRequest(r)
			}
		case http.MethodPost:
			res, err = postRequest(r)
		case http.MethodPut:
			res, err = putRequest(r)
		}

		if err != nil {
			t.Fatal(err)
		}
		w.Header().Add("Content-Type", "application/json")
		_, err = w.Write(res)
		if err != nil {
			t.Fatal(err)
		}
	}

	return FakeService(t, host, http.HandlerFunc(handler))
}

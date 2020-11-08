package testproxyhelpers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	TestToken = "abcdefg1234567"
	BadToken  = "bad"
)

func FakeIntrospectorService(t *testing.T, host string) *httptest.Server {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var (
			err error
			res []byte
		)

		if r.Method == http.MethodPost &&
			r.URL.Path == "/oauth2/introspect" &&
			r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			err = r.ParseForm()
			if r.PostForm.Get("token") == TestToken {
				res = []byte(`{"active":true, "subject":"1", "token_type":"access_token"}`)
				t.Log("it's an active token")
			} else {
				res = []byte(`{"active":false}`)
				t.Log("it's an inactive token")
			}
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

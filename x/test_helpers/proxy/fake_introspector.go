package testproxyhelpers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	DefaultFakeIntrospectorHost           = "localhost:10001"
	DefaultFakeIntrospectorURL            = "http://" + DefaultFakeIntrospectorHost
	DefaultFakeIntrospectorEndpoint       = "/oauth2/introspect"
	DefaultFakeIntrospectorContentType    = "application/x-www-form-urlencoded"
	DefaultFakeIntrospectorMethod         = "POST"
	DefaultFakeIntrospectorValidMarker    = `"active":true`
	DefaultFakeIntrospectorBodyTemplate   = `token_type_hint=access_token&token={{.Token}}`
	DefaultFakeIntrospectorCookieName     = "access-token"
	DefaultFakeIntrospectorQueryParamName = "access_token"
	DefaultFakeIntrospectorHeaderName     = "authorization"

	TestToken = "goodToken"
	BadToken  = "badToken"
)

func DefaultFakeIntrospectorCookiesName() []string {
	return []string{DefaultFakeIntrospectorCookieName}
}

func DefaultFakeIntrospectorQueryParamsName() []string {
	return []string{DefaultFakeIntrospectorQueryParamName}
}

func DefaultFakeIntrospectorHeadersName() []string {
	return []string{DefaultFakeIntrospectorHeaderName}
}

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
			t.Logf("token %s", r.PostForm.Get("token"))
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

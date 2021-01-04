package testproxyhelpers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/introspection"
	testctxhelper "github.com/soldatov-s/accp/x/test_helpers/ctx"
	"github.com/stretchr/testify/require"
)

const (
	TestToken = "goodToken"
	BadToken  = "badToken"
)

func InitTestIntrospector(t *testing.T) *introspection.Introspect {
	ctx := testctxhelper.InitTestCtx(t)

	ic := &introspection.Config{
		DSN:            "http://localhost:8001",
		Endpoint:       "/oauth2/introspect",
		ContentType:    "application/x-www-form-urlencoded",
		Method:         "POST",
		ValidMarker:    `"active":true`,
		BodyTemplate:   `token_type_hint=access_token&token={{.Token}}`,
		CookieName:     []string{"access-token"},
		QueryParamName: []string{"access_token"},
		Pool: &httpclient.PoolConfig{
			Size:    50,
			Timeout: 10 * time.Second,
		},
	}

	i, err := introspection.NewIntrospector(ctx, ic)
	require.Nil(t, err)

	return i
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

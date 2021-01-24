package limits

import (
	"net/http"
	"testing"

	testProxyHelpers "github.com/soldatov-s/accp/x/test_helpers/proxy"
	"github.com/stretchr/testify/require"
)

const (
	testIP              = "10.1.10.113"
	testBearerToken     = "bearer " + testToken
	testUserCookieName  = "user-cookie"
	testUserCookieValue = "test_value"
	testItemUser        = "useritem"
)

// nolint : funlen
func TestNewLimitedParamsOfRequest(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "get ip and token from header",
			testFunc: func() {
				mc := NewMapConfig()
				req, err := http.NewRequest(http.MethodGet, testProxyHelpers.DefaultFakeServiceURL+testProxyHelpers.GetEndpoint, nil)
				require.Nil(t, err)
				req.Header.Add(authorizationHeader, testToken)
				req.Header.Add(ipHeader, testIP)

				lp := NewLimitedParamsOfRequest(mc, req)
				require.NotNil(t, lp)
				require.Equal(t, 2, len(lp))

				v, ok := lp[defaultItemToken]
				require.True(t, ok)
				require.Equal(t, testToken, v)

				v, ok = lp[defaultItemIP]
				require.True(t, ok)
				require.Equal(t, testIP, v)
			},
		},
		{
			name: "get ip and bearer token from header",
			testFunc: func() {
				mc := NewMapConfig()
				req, err := http.NewRequest(http.MethodGet, testProxyHelpers.DefaultFakeServiceURL+testProxyHelpers.GetEndpoint, nil)
				require.Nil(t, err)
				req.Header.Add(authorizationHeader, testBearerToken)
				req.Header.Add(ipHeader, testIP)

				lp := NewLimitedParamsOfRequest(mc, req)
				require.NotNil(t, lp)
				require.Equal(t, 2, len(lp))

				v, ok := lp[defaultItemToken]
				require.True(t, ok)
				require.Equal(t, testToken, v)

				v, ok = lp[defaultItemIP]
				require.True(t, ok)
				require.Equal(t, testIP, v)
			},
		},
		{
			name: "get ip from x-forwarded-for with proxy and token from header",
			testFunc: func() {
				mc := NewMapConfig()
				req, err := http.NewRequest(http.MethodGet, testProxyHelpers.DefaultFakeServiceURL+testProxyHelpers.GetEndpoint, nil)
				require.Nil(t, err)
				req.Header.Add(authorizationHeader, testToken)
				req.Header.Add(ipHeader, testIP+", 192.168.1.100")

				lp := NewLimitedParamsOfRequest(mc, req)
				require.NotNil(t, lp)
				require.Equal(t, 2, len(lp))

				v, ok := lp[defaultItemToken]
				require.True(t, ok)
				require.Equal(t, testToken, v)

				v, ok = lp[defaultItemIP]
				require.True(t, ok)
				require.Equal(t, testIP, v)
			},
		},
		{
			name: "get ip from header, token from cookie",
			testFunc: func() {
				mc := NewMapConfig()
				mc[defaultItemToken].Cookie = testCookie()
				req, err := http.NewRequest(http.MethodGet, testProxyHelpers.DefaultFakeServiceURL+testProxyHelpers.GetEndpoint, nil)
				require.Nil(t, err)

				cookie := http.Cookie{Name: testAuthCookieName, Value: testToken}
				req.AddCookie(&cookie)

				req.Header.Add(ipHeader, testIP)

				lp := NewLimitedParamsOfRequest(mc, req)
				require.NotNil(t, lp)
				require.Equal(t, 2, len(lp))

				v, ok := lp[defaultItemToken]
				require.True(t, ok)
				require.Equal(t, testToken, v)

				v, ok = lp[defaultItemIP]
				require.True(t, ok)
				require.Equal(t, testIP, v)
			},
		},
		{
			name: "get ip from header, token from cookie, user cookie",
			testFunc: func() {
				mc := NewMapConfig()

				mc[defaultItemToken].Cookie = testCookie()
				mc[testItemUser] = &Config{Cookie: []string{testUserCookieName}}
				req, err := http.NewRequest(http.MethodGet, testProxyHelpers.DefaultFakeServiceURL+testProxyHelpers.GetEndpoint, nil)
				require.Nil(t, err)

				cookie := http.Cookie{Name: testAuthCookieName, Value: testToken}
				req.AddCookie(&cookie)

				cookie = http.Cookie{Name: testUserCookieName, Value: testUserCookieValue}
				req.AddCookie(&cookie)

				t.Log(req.Header.Get("Cookie"))

				req.Header.Add(ipHeader, testIP)

				lp := NewLimitedParamsOfRequest(mc, req)
				require.NotNil(t, lp)
				require.Equal(t, 3, len(lp))

				v, ok := lp[defaultItemToken]
				require.True(t, ok)
				require.Equal(t, testToken, v)

				v, ok = lp[defaultItemIP]
				require.True(t, ok)
				require.Equal(t, testIP, v)

				t.Log(lp)
				v, ok = lp[testItemUser]
				require.True(t, ok)
				require.Equal(t, testUserCookieValue, v)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc()
		})
	}
}

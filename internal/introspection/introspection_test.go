package introspection

import (
	"context"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/meta"
	testproxyhelpers "github.com/soldatov-s/accp/x/test_helpers/proxy"
	"github.com/stretchr/testify/require"
)

const (
	testPoolSize    = 10
	testPoolTimeout = 5 * time.Second
)

func initApp(ctx context.Context) context.Context {
	return meta.SetAppInfo(ctx, "accp", "", "", "", "test")
}

func initLogger(ctx context.Context) context.Context {
	// Registrate logger
	logCfg := &logger.Config{
		Level:           logger.LoggerLevelDebug,
		NoColoredOutput: true,
		WithTrace:       false,
	}
	ctx = logger.RegistrateAndInitilize(ctx, logCfg)

	return ctx
}

func initPool() *httpclient.Config {
	return &httpclient.Config{
		Size:    testPoolSize,
		Timeout: testPoolTimeout,
	}
}

func initConfig() *Config {
	return &Config{
		DSN:            testproxyhelpers.DefaultFakeIntrospectorURL,
		Endpoint:       testproxyhelpers.DefaultFakeIntrospectorEndpoint,
		ContentType:    testproxyhelpers.DefaultFakeIntrospectorContentType,
		Method:         testproxyhelpers.DefaultFakeIntrospectorMethod,
		ValidMarker:    testproxyhelpers.DefaultFakeIntrospectorValidMarker,
		BodyTemplate:   testproxyhelpers.DefaultFakeIntrospectorBodyTemplate,
		CookieName:     testproxyhelpers.DefaultFakeIntrospectorCookiesName(),
		QueryParamName: testproxyhelpers.DefaultFakeIntrospectorQueryParamsName(),
		Pool:           initPool(),
	}
}

func initTestIntrospector(t *testing.T) *Introspect {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()

	i, err := NewIntrospector(ctx, cfg)
	require.Nil(t, err)

	return i
}

func TestNewIntrospector(t *testing.T) {
	_ = initTestIntrospector(t)
}

func TestExtractToken(t *testing.T) {
	i := initTestIntrospector(t)

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test not found token",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, nil)
				require.Nil(t, err)

				token, err := i.extractToken(req)
				require.NotNil(t, err)
				require.Equal(t, ErrBadAuthRequest, err)
				require.Empty(t, token)
			},
		},
		{
			name: "test token in header",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, nil)
				require.Nil(t, err)

				req.Header.Add(authorizationHeader, "token "+testproxyhelpers.TestToken)

				token, err := i.extractToken(req)
				require.Nil(t, err)
				require.Equal(t, testproxyhelpers.TestToken, token)
			},
		},
		{
			name: "test token in cookie",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, nil)
				require.Nil(t, err)

				cookie := http.Cookie{Name: testproxyhelpers.DefaultFakeIntrospectorCookieName, Value: testproxyhelpers.TestToken}
				req.AddCookie(&cookie)

				token, err := i.extractToken(req)
				require.Nil(t, err)
				require.Equal(t, testproxyhelpers.TestToken, token)
			},
		},
		{
			name: "test token in query",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodGet,
					testproxyhelpers.GetEndpoint+"?"+testproxyhelpers.DefaultFakeIntrospectorQueryParamName+"="+testproxyhelpers.TestToken,
					nil)
				require.Nil(t, err)

				token, err := i.extractToken(req)
				require.Nil(t, err)
				require.Equal(t, testproxyhelpers.TestToken, token)
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

func TestBuildRequest(t *testing.T) {
	i := initTestIntrospector(t)

	req, err := i.buildRequest(testproxyhelpers.TestToken)
	require.Nil(t, err)

	require.Equal(t, testproxyhelpers.DefaultFakeIntrospectorMethod, req.Method)
	require.Equal(t, testproxyhelpers.DefaultFakeIntrospectorURL+testproxyhelpers.DefaultFakeIntrospectorEndpoint, req.URL.String())
	contentType := req.Header.Get("Content-Type")
	require.Equal(t, testproxyhelpers.DefaultFakeIntrospectorContentType, contentType)
	body, err := ioutil.ReadAll(req.Body)
	require.Nil(t, err)
	defer req.Body.Close()
	require.Equal(t, "token_type_hint=access_token&token=goodToken", string(body))
}

func TestIsValid(t *testing.T) {
	i := initTestIntrospector(t)
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "token is not valid",
			testFunc: func() {
				result := i.isValid([]byte("not valid"))
				require.False(t, result)
			},
		},
		{
			name: "token is valid",
			testFunc: func() {
				result := i.isValid([]byte(testproxyhelpers.DefaultFakeIntrospectorValidMarker))
				require.True(t, result)
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

func TestTrimFields(t *testing.T) {
	i := initTestIntrospector(t)

	i.cfg.TrimmedFilds = []string{"exp", "iat"}
	i.initRegex()

	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "1-st type of body",
			content: []byte(`{"token":"token","token_type":"bearer","exp":12345,"iat":123456}`),
		},
		{
			name:    "2-nd type of body",
			content: []byte(`{"exp":12345,"iat":123456,"token":"token","token_type":"bearer"}`),
		},
		{
			name:    "3-ed type of body",
			content: []byte(`{"token":"token","exp":12345,"iat":123456,"token_type":"bearer"}`),
		},
		{
			name:    "4-th type of body",
			content: []byte(`{"token":"token","exp":12345,"token_type":"bearer","iat":123456}`),
		},
	}

	expectedContent := []byte(`{"token":"token","token_type":"bearer"}`)
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("introspect response before trim: %s", string(tt.content))
			content := i.trimFields(tt.content)
			t.Logf("introspect response after trim: %s", string(content))
			require.Equal(t, expectedContent, content)
		})
	}
}

func TestIntrospectRequest(t *testing.T) {
	i := initTestIntrospector(t)
	server := testproxyhelpers.FakeIntrospectorService(t, testproxyhelpers.DefaultFakeIntrospectorHost)
	server.Start()
	defer server.Close()

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test not found token",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, nil)
				require.Nil(t, err)

				result, err := i.IntrospectRequest(req)
				require.NotNil(t, err)
				require.Nil(t, result)
			},
		},
		{
			name: "test good token in header",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, nil)
				require.Nil(t, err)

				req.Header.Add(authorizationHeader, "token "+testproxyhelpers.TestToken)

				result, err := i.IntrospectRequest(req)
				require.Nil(t, err)
				require.Equal(t, `{"active":true, "subject":"1", "token_type":"access_token"}`, string(result))
			},
		},
		{
			name: "test bad token in header",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodGet, testproxyhelpers.GetEndpoint, nil)
				require.Nil(t, err)

				req.Header.Add(authorizationHeader, "token "+testproxyhelpers.BadToken)

				result, err := i.IntrospectRequest(req)
				require.NotNil(t, err)
				require.Nil(t, result)
				require.Equal(t, &ErrTokenInactive{token: testproxyhelpers.BadToken}, err)
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

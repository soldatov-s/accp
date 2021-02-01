package captcha

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/httpclient"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/meta"
	"github.com/soldatov-s/accp/x/helper"
	testproxyhelpers "github.com/soldatov-s/accp/x/test_helpers/proxy"
	"github.com/stretchr/testify/require"
)

const (
	testPoolSize      = 10
	testPoolTimeout   = 5 * time.Second
	testCaptchaHeader = "Accp-Captcha-Header"
	testCaptchaCookie = "accp-captcha-cookie"
	testJWTToken      = "test_token"
)

func testCaptchaHeaders() helper.Arguments {
	return helper.Arguments{testCaptchaHeader}
}

func testCaptchaCookies() helper.Arguments {
	return helper.Arguments{testCaptchaCookie}
}

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
		TokenSign: testproxyhelpers.DefaultFakeCaptchaSign,
		VerifyURL: testproxyhelpers.DefaultFakeCaptchaURL + testproxyhelpers.DefaultFakeCaptchaEndpoint,
		Key:       testproxyhelpers.DefaultFakeCaptchaKey,
		Header:    testCaptchaHeaders(),
		Cookie:    testCaptchaCookies(),
		Pool:      initPool(),
	}
}

func initTestGoogleCaptcha(t *testing.T) *GoogleCaptcha {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()

	i, err := NewGoogleCaptcha(ctx, cfg)
	require.Nil(t, err)

	return i
}

func TestNewGoogleCaptcha(t *testing.T) {
	_ = initTestGoogleCaptcha(t)
}

func TestGetCaptchaJWTToken(t *testing.T) {
	g := initTestGoogleCaptcha(t)
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test cookie",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodPost, testproxyhelpers.DefaultFakeCaptchaURL+testproxyhelpers.DefaultFakeCaptchaEndpoint, nil)
				require.Nil(t, err)

				cookie := http.Cookie{Name: testCaptchaCookie, Value: testJWTToken, Expires: time.Now(), HttpOnly: true}
				req.AddCookie(&cookie)

				jwtToken := g.getCaptchaJWTToken(req)
				require.Equal(t, testJWTToken, jwtToken)
			},
		},
		{
			name: "test header",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodPost, testproxyhelpers.DefaultFakeCaptchaURL+testproxyhelpers.DefaultFakeCaptchaEndpoint, nil)
				require.Nil(t, err)

				req.Header.Set(testCaptchaHeader, testJWTToken)

				jwtToken := g.getCaptchaJWTToken(req)
				require.Equal(t, testJWTToken, jwtToken)
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

func TestGenerateCaptchaJWT(t *testing.T) {
	g := initTestGoogleCaptcha(t)
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test normal generation",
			testFunc: func() {
				w := httptest.NewRecorder()
				err := g.GenerateCaptchaJWT(w)
				require.Nil(t, err)

				resp := w.Result()
				defer resp.Body.Close()
				jwtToken := resp.Header.Get(testCaptchaHeader)
				require.NotEmpty(t, jwtToken)

				t.Logf("jwtToken %s", jwtToken)
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

func TestValidateCaptchaJWT(t *testing.T) {
	g := initTestGoogleCaptcha(t)

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test normal token",
			testFunc: func() {
				w := httptest.NewRecorder()
				err := g.GenerateCaptchaJWT(w)
				require.Nil(t, err)

				resp := w.Result()
				defer resp.Body.Close()
				jwtToken := resp.Header.Get(testCaptchaHeader)
				require.NotEmpty(t, jwtToken)

				t.Logf("jwtToken %s", jwtToken)

				req, err := http.NewRequest(http.MethodPost, testproxyhelpers.DefaultFakeCaptchaURL+testproxyhelpers.DefaultFakeCaptchaEndpoint, nil)
				require.Nil(t, err)

				req.Header.Set(testCaptchaHeader, jwtToken)

				err = g.validateCaptchaJWT(req)
				require.Nil(t, err)
			},
		},
		{
			name: "test bad token",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodPost, testproxyhelpers.DefaultFakeCaptchaURL+testproxyhelpers.DefaultFakeCaptchaEndpoint, nil)
				require.Nil(t, err)

				req.Header.Set(testCaptchaHeader, testJWTToken)

				err = g.validateCaptchaJWT(req)
				require.NotNil(t, err)
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

func TestValidateReCAPTCHA(t *testing.T) {
	g := initTestGoogleCaptcha(t)
	server := testproxyhelpers.FakeCaptchaService(t, testproxyhelpers.DefaultFakeCaptchaHost)
	server.Start()
	defer server.Close()

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test normal token",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodPost, testproxyhelpers.DefaultFakeCaptchaURL+testproxyhelpers.DefaultFakeCaptchaEndpoint, nil)
				require.Nil(t, err)

				req.Form = url.Values{}
				req.Form["g-recaptcha-response"] = []string{testproxyhelpers.DefaultFakeGoodCaptcha}

				err = g.validateReCAPTCHA(req)
				require.Nil(t, err)
			},
		},
		{
			name: "test bad token",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodPost, testproxyhelpers.DefaultFakeCaptchaURL+testproxyhelpers.DefaultFakeCaptchaEndpoint, nil)
				require.Nil(t, err)

				req.Form = url.Values{}
				req.Form["g-recaptcha-response"] = []string{testproxyhelpers.DefaultFakeBadCaptcha}

				err = g.validateReCAPTCHA(req)
				require.NotNil(t, err)
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

func TestValidate(t *testing.T) {
	g := initTestGoogleCaptcha(t)
	server := testproxyhelpers.FakeCaptchaService(t, testproxyhelpers.DefaultFakeCaptchaHost)
	server.Start()
	defer server.Close()

	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test normal token from google",
			testFunc: func() {
				w := httptest.NewRecorder()
				err := g.GenerateCaptchaJWT(w)
				require.Nil(t, err)

				req, err := http.NewRequest(http.MethodPost, testproxyhelpers.DefaultFakeCaptchaURL+testproxyhelpers.DefaultFakeCaptchaEndpoint, nil)
				require.Nil(t, err)

				req.Form = url.Values{}
				req.Form["g-recaptcha-response"] = []string{testproxyhelpers.DefaultFakeGoodCaptcha}

				src, err := g.Validate(req)
				require.Nil(t, err)
				require.Equal(t, CaptchaFromGoogle, src)
			},
		},
		{
			name: "test normal token from client",
			testFunc: func() {
				w := httptest.NewRecorder()
				err := g.GenerateCaptchaJWT(w)
				require.Nil(t, err)

				resp := w.Result()
				defer resp.Body.Close()
				jwtToken := resp.Header.Get(testCaptchaHeader)
				require.NotEmpty(t, jwtToken)

				t.Logf("jwtToken %s", jwtToken)

				req, err := http.NewRequest(http.MethodPost, testproxyhelpers.DefaultFakeCaptchaURL+testproxyhelpers.DefaultFakeCaptchaEndpoint, nil)
				require.Nil(t, err)

				req.Header.Set(testCaptchaHeader, jwtToken)

				src, err := g.Validate(req)
				require.Nil(t, err)
				require.Equal(t, CaptchaFromClient, src)
			},
		},
		{
			name: "test bad token",
			testFunc: func() {
				req, err := http.NewRequest(http.MethodPost, testproxyhelpers.DefaultFakeCaptchaURL+testproxyhelpers.DefaultFakeCaptchaEndpoint, nil)
				require.Nil(t, err)

				req.Header.Set(testCaptchaHeader, testJWTToken)

				src, err := g.Validate(req)
				require.NotNil(t, err)
				require.Equal(t, CaptchaFromGoogle, src)
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

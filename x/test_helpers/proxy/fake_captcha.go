package testproxyhelpers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	DefaultFakeCaptchaHost     = "localhost:10002"
	DefaultFakeCaptchaURL      = "http://" + DefaultFakeCaptchaHost
	DefaultFakeCaptchaEndpoint = "/siteverify"
	DefaultFakeCaptchaKey      = "captcha_key"
	DefaultFakeGoodCaptcha     = "good_captcha"
	DefaultFakeBadCaptcha      = "bad_captcha"
	DefaultFakeCaptchaSign     = "test_sign"
)

type GoogleRecaptchaResponse struct {
	Success            bool     `json:"success"`
	ChallengeTimestamp string   `json:"challenge_ts"`
	Hostname           string   `json:"hostname"`
	ErrorCodes         []string `json:"error-codes"`
}

func FakeCaptchaService(t *testing.T, host string) *httptest.Server {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var (
			err error
			res []byte
		)

		if r.Method == http.MethodPost &&
			r.URL.Path == DefaultFakeCaptchaEndpoint &&
			r.Header.Get("Content-Type") == ContentTypeFormUrlencoded {
			err = r.ParseForm()
			t.Logf("post form %s", r.PostForm)
			if r.PostForm.Get("response") == DefaultFakeGoodCaptcha && r.PostForm.Get("secret") == DefaultFakeCaptchaSign {
				res = []byte(`{"success":true, "challenge_ts":"2021-01-23T23:03:11Z", "hostname":"localhost"}`)
				t.Log("it's an active captcha")
			} else {
				res = []byte(`{"success":false, "challenge_ts":"2021-01-23T23:03:11Z", "hostname":"localhost", "error-codes":["bad-request"]}`)
				t.Log("it's a bad captcha")
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

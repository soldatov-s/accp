package testproxyhelpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/httputils"
	"github.com/stretchr/testify/require"
)

const (
	GetEndpoint            = "/api/v1/check_get"
	PostEndpoint           = "/api/v1/check_post"
	PutEndpoint            = "/api/v1/check_put"
	DefaultFakeServiceHost = "localhost:10000"
	DefaultFakeServiceURL  = "http://" + DefaultFakeServiceHost
	DefaultGetAnswer       = "it's a get request"
	DefaultPostAnswer      = "it's a post request"
	DefaultPutAnswer       = "it's a put request"
)

type HTTPBody struct {
	Result struct {
		Message   string    `json:"message"`
		TimeStamp time.Time `json:"timestamp"`
	} `json:"result"`
}

func NewResponse(msg string) *HTTPBody {
	answer := &HTTPBody{}
	answer.Result.Message = msg
	answer.Result.TimeStamp = time.Now()
	return answer
}

func DefaultHeader() http.Header {
	h := make(http.Header)
	h.Add("Content-Type", "application/json")
	return h
}

func getRequest(t *testing.T, w http.ResponseWriter, _ *http.Request) {
	respData := NewResponse("it's a get request")
	data, err := json.Marshal(&respData)
	require.Nil(t, err)

	httputils.CopyHeader(w.Header(), DefaultHeader())
	_, err = w.Write(data)
	require.Nil(t, err)
}

// nolint : unparam
func postRequest(_ *http.Request) (res []byte, err error) {
	return []byte(DefaultPostAnswer), nil
}

// nolint : unparam
func putRequest(_ *http.Request) (res []byte, err error) {
	return []byte(DefaultPutAnswer), nil
}

func FakeBackendService(t *testing.T, host string) *httptest.Server {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			switch r.URL.Path {
			case GetEndpoint:
				getRequest(t, w, r)
			default:
				t.Log("not found", r.URL)
				http.NotFound(w, r)
			}
		}
	}

	return FakeService(t, host, http.HandlerFunc(handler))
}

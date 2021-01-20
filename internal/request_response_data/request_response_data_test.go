package rrdata

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soldatov-s/accp/internal/httputils"
	testProxyHelpers "github.com/soldatov-s/accp/x/test_helpers/proxy"
	"github.com/stretchr/testify/require"
)

const (
	testRequestResponseBody      = "test body"
	testRequestResponseHK        = "test"
	testRequestResponseMax       = 5
	testRequestResponseTimeStamp = int64(1611145887)
	testRequestResponseUUID      = "4498280e-91ba-46d8-9030-6720d3ca6a9b"
	// nolint : lll
	testRequestResponseJSON = `{"Body":"test body","Header":null,"StatusCode":200,"TimeStamp":1611145887,"UUID":"4498280e-91ba-46d8-9030-6720d3ca6a9b"}`
)

func TestNewRequestResponseData(t *testing.T) {
	rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
	require.NotNil(t, rrData)
}

func TestRequestResponseData_MarshalBinary(t *testing.T) {
	rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
	require.NotNil(t, rrData)

	rrData.Response.StatusCode = http.StatusOK
	u, err := uuid.Parse(testRequestResponseUUID)
	require.Nil(t, err)
	rrData.Response.Body = testRequestResponseBody
	rrData.Response.UUID = u
	rrData.Response.TimeStamp = testRequestResponseTimeStamp

	data, err := rrData.MarshalBinary()
	require.Nil(t, err)
	require.Equal(t, testRequestResponseJSON, string(data))
}

func TestRequestResponseData_UnmarshalBinary(t *testing.T) {
	rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
	require.NotNil(t, rrData)

	err := rrData.UnmarshalBinary([]byte(testRequestResponseJSON))
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, rrData.Response.StatusCode)
	u, err := uuid.Parse(testRequestResponseUUID)
	require.Nil(t, err)
	require.Equal(t, u, rrData.Response.UUID)
	require.Equal(t, testRequestResponseTimeStamp, rrData.Response.TimeStamp)
}

func TestGetStatusCode(t *testing.T) {
	rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
	require.NotNil(t, rrData)

	rrData.Response.StatusCode = http.StatusOK
	statusCode := rrData.GetStatusCode()
	require.Equal(t, http.StatusOK, statusCode)
}

// nolint : funlen
func TestUpdate(t *testing.T) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test request with error: connection refused",
			testFunc: func() {
				req := initHTTPRequest(t)

				rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
				require.NotNil(t, rrData)

				err := rrData.Request.Read(req)
				require.Nil(t, err)

				err = rrData.Update(client)
				require.Nil(t, err)

				errResp := httputils.ErrResponse(
					`Get "`+testProxyHelpers.DefaultFakeServiceURL+testProxyHelpers.GetEndpoint+
						`": dial tcp 127.0.0.1:10000: connect: connection refused`,
					http.StatusServiceUnavailable,
				)
				require.Equal(t, errResp.StatusCode, rrData.Response.StatusCode)
				require.Equal(t, errResp.Header, rrData.Response.Header)
				bodyBytes, err := ioutil.ReadAll(errResp.Body)
				require.Nil(t, err)
				require.Equal(t, string(bodyBytes), rrData.Response.Body)
				require.NotEmpty(t, rrData.Response.UUID)
				require.NotEmpty(t, rrData.Response.TimeStamp)
			},
		},
		{
			name: "test request to backend",
			testFunc: func() {
				req := initHTTPRequest(t)

				rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
				require.NotNil(t, rrData)

				server := testProxyHelpers.FakeBackendService(t, testProxyHelpers.DefaultFakeServiceHost)
				server.Start()
				defer server.Close()

				err := rrData.Request.Read(req)
				require.Nil(t, err)

				err = rrData.Update(client)
				require.Nil(t, err)

				var respData testProxyHelpers.HTTPBody
				err = json.Unmarshal([]byte(rrData.Response.Body), &respData)
				require.Nil(t, err)

				require.Equal(t, testProxyHelpers.DefaultGetAnswer, respData.Result.Message)
			},
		},
		{
			name: "test double request to backend",
			testFunc: func() {
				req := initHTTPRequest(t)

				rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
				require.NotNil(t, rrData)

				server := testProxyHelpers.FakeBackendService(t, testProxyHelpers.DefaultFakeServiceHost)
				server.Start()
				defer server.Close()

				// first request
				err := rrData.Request.Read(req)
				require.Nil(t, err)

				err = rrData.Update(client)
				require.Nil(t, err)

				var respData testProxyHelpers.HTTPBody
				err = json.Unmarshal([]byte(rrData.Response.Body), &respData)
				require.Nil(t, err)

				require.Equal(t, testProxyHelpers.DefaultGetAnswer, respData.Result.Message)
				firstRequestTimeStamp := respData.Result.TimeStamp

				// second request
				err = rrData.Request.Read(req)
				require.Nil(t, err)

				err = rrData.Update(client)
				require.Nil(t, err)

				err = json.Unmarshal([]byte(rrData.Response.Body), &respData)
				require.Nil(t, err)

				require.Equal(t, testProxyHelpers.DefaultGetAnswer, respData.Result.Message)

				// compare timestamps
				require.NotEqual(t, firstRequestTimeStamp, respData.Result.TimeStamp)
			},
		},
		{
			name: "test request to backend, backend returns 404",
			testFunc: func() {
				req := initHTTPRequest(t)

				rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
				require.NotNil(t, rrData)

				var err error
				req.URL, err = url.Parse(req.URL.String() + "/" + uuid.New().String())
				require.Nil(t, err)

				server := testProxyHelpers.FakeBackendService(t, testProxyHelpers.DefaultFakeServiceHost)
				server.Start()
				defer server.Close()

				err = rrData.Request.Read(req)
				require.Nil(t, err)

				err = rrData.Update(client)
				require.Nil(t, err)
				require.Equal(t, http.StatusNotFound, rrData.Response.StatusCode)

				t.Log(rrData.Response.Body)
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

// nolint : funlen
func TestUpdateByRequest(t *testing.T) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test request with error: connection refused",
			testFunc: func() {
				req := initHTTPRequest(t)

				rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
				require.NotNil(t, rrData)

				err := rrData.UpdateByRequest(client, req)
				require.Nil(t, err)

				errResp := httputils.ErrResponse(
					`Get "`+testProxyHelpers.DefaultFakeServiceURL+testProxyHelpers.GetEndpoint+
						`": dial tcp 127.0.0.1:10000: connect: connection refused`,
					http.StatusServiceUnavailable,
				)
				require.Equal(t, errResp.StatusCode, rrData.Response.StatusCode)
				require.Equal(t, errResp.Header, rrData.Response.Header)
				bodyBytes, err := ioutil.ReadAll(errResp.Body)
				require.Nil(t, err)
				require.Equal(t, string(bodyBytes), rrData.Response.Body)
				require.NotEmpty(t, rrData.Response.UUID)
				require.NotEmpty(t, rrData.Response.TimeStamp)
			},
		},
		{
			name: "test request to backend",
			testFunc: func() {
				req := initHTTPRequest(t)

				rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
				require.NotNil(t, rrData)

				server := testProxyHelpers.FakeBackendService(t, testProxyHelpers.DefaultFakeServiceHost)
				server.Start()
				defer server.Close()

				err := rrData.UpdateByRequest(client, req)
				require.Nil(t, err)

				var respData testProxyHelpers.HTTPBody
				err = json.Unmarshal([]byte(rrData.Response.Body), &respData)
				require.Nil(t, err)

				require.Equal(t, testProxyHelpers.DefaultGetAnswer, respData.Result.Message)
			},
		},
		{
			name: "test double request to backend",
			testFunc: func() {
				req := initHTTPRequest(t)

				rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
				require.NotNil(t, rrData)

				server := testProxyHelpers.FakeBackendService(t, testProxyHelpers.DefaultFakeServiceHost)
				server.Start()
				defer server.Close()

				// first request
				err := rrData.UpdateByRequest(client, req)
				require.Nil(t, err)

				var respData testProxyHelpers.HTTPBody
				err = json.Unmarshal([]byte(rrData.Response.Body), &respData)
				require.Nil(t, err)

				require.Equal(t, testProxyHelpers.DefaultGetAnswer, respData.Result.Message)
				firstRequestTimeStamp := respData.Result.TimeStamp

				// second request
				err = rrData.UpdateByRequest(client, req)
				require.Nil(t, err)

				err = json.Unmarshal([]byte(rrData.Response.Body), &respData)
				require.Nil(t, err)

				require.Equal(t, testProxyHelpers.DefaultGetAnswer, respData.Result.Message)

				// compare timestamps
				require.NotEqual(t, firstRequestTimeStamp, respData.Result.TimeStamp)
			},
		},
		{
			name: "test request to backend, backend returns 404",
			testFunc: func() {
				req := initHTTPRequest(t)

				rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
				require.NotNil(t, rrData)

				var err error
				req.URL, err = url.Parse(req.URL.String() + "/" + uuid.New().String())
				require.Nil(t, err)

				server := testProxyHelpers.FakeBackendService(t, testProxyHelpers.DefaultFakeServiceHost)
				server.Start()
				defer server.Close()

				err = rrData.UpdateByRequest(client, req)
				require.Nil(t, err)
				require.Equal(t, http.StatusNotFound, rrData.Response.StatusCode)

				t.Log(rrData.Response.Body)
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
func TestReadAll(t *testing.T) {
	rrData := NewRequestResponseData(testRequestResponseHK, testRequestResponseMax, nil)
	require.NotNil(t, rrData)

	resp := initHTTPResponse()
	defer resp.Body.Close()

	req := initHTTPRequest(t)

	err := rrData.ReadAll(req, resp)
	require.Nil(t, err)
}

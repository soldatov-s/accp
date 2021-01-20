package rrdata

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	testProxyHelpers "github.com/soldatov-s/accp/x/test_helpers/proxy"
	"github.com/stretchr/testify/require"
)

const (
	testRequestURL         = testProxyHelpers.DefaultFakeServiceURL + testProxyHelpers.GetEndpoint
	testRequestBodyString  = "test body"
	testRequestHeaderName  = "test-header"
	testRequestHeaderValue = "test value"
	testRequestJSONData    = `{"URL":"` +
		testProxyHelpers.DefaultFakeServiceURL +
		testProxyHelpers.GetEndpoint +
		`","Method":"GET","Body":"test body","Header":{"Test-Header":["test value"]}}`
)

func initHTTPRequest(t *testing.T) *http.Request {
	req, err := http.NewRequest(http.MethodGet, testRequestURL, bytes.NewBufferString(testRequestBodyString))
	require.Nil(t, err)
	req.Header.Add(testRequestHeaderName, testRequestHeaderValue)
	return req
}

func TestNewRequestData(t *testing.T) {
	req := initHTTPRequest(t)
	reqData, err := NewRequestData(req)
	require.Nil(t, err)
	require.NotNil(t, reqData)

	require.Equal(t, testRequestBodyString, reqData.Body)
	require.Equal(t, http.MethodGet, reqData.Method)
	require.Equal(t, testRequestURL, reqData.URL)
	require.Equal(t, req.Header, reqData.Header)
}

func TestRequestData_MarshalBinary(t *testing.T) {
	req := initHTTPRequest(t)
	reqData, err := NewRequestData(req)
	require.Nil(t, err)
	require.NotNil(t, reqData)

	data, err := reqData.MarshalBinary()
	require.Nil(t, err)
	require.NotNil(t, data)
	require.Equal(t, testRequestJSONData, string(data))
}

func TestRequestData_UnmarshalBinary(t *testing.T) {
	var reqData RequestData
	err := reqData.UnmarshalBinary([]byte(testRequestJSONData))
	require.Nil(t, err)
	require.Equal(t, testRequestBodyString, reqData.Body)
	require.Equal(t, http.MethodGet, reqData.Method)
	require.Equal(t, testRequestURL, reqData.URL)
	h := make(http.Header)
	h.Add(testRequestHeaderName, testRequestHeaderValue)
	require.Equal(t, h, reqData.Header)
}

func TestRequestData_Read(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "test read request with not nil body",
			testFunc: func() {
				reqData := RequestData{}
				req := initHTTPRequest(t)
				err := reqData.Read(req)
				require.Nil(t, err)

				require.Equal(t, testRequestBodyString, reqData.Body)
				require.Equal(t, http.MethodGet, reqData.Method)
				require.Equal(t, testRequestURL, reqData.URL)
				require.Equal(t, req.Header, reqData.Header)
			},
		},
		{
			name: "test read request with nil body",
			testFunc: func() {
				reqData := RequestData{}
				req := initHTTPRequest(t)
				req.Body = nil
				err := reqData.Read(req)
				require.Nil(t, err)

				require.Equal(t, "", reqData.Body)
				require.Equal(t, http.MethodGet, reqData.Method)
				require.Equal(t, testRequestURL, reqData.URL)
				require.Equal(t, req.Header, reqData.Header)
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
	reqData := &RequestData{}
	req := initHTTPRequest(t)
	err := reqData.Read(req)
	require.Nil(t, err)

	req2, err := reqData.BuildRequest()
	require.Nil(t, err)
	require.NotNil(t, req2)

	reqBody, err := ioutil.ReadAll(req.Body)
	require.Nil(t, err)
	require.NotNil(t, reqBody)

	reqBody2, err := ioutil.ReadAll(req2.Body)
	require.Nil(t, err)
	require.NotNil(t, reqBody2)

	require.Equal(t, reqBody, reqBody2)
	t.Logf("src: %s, target: %s", string(reqBody), string(reqBody2))

	// Test empty RequestData
	reqData = nil
	req2, err = reqData.BuildRequest()
	require.NotNil(t, err)
	require.Nil(t, req2)
	require.Equal(t, ErrEmptyRequest, err)
}

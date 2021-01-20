package rrdata

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	testResponseBody        = "test body"
	testResponseHeaderName  = "test-header"
	testResponseHeaderValue = "test value"
	testResponseHK          = "test"
	testResponseMax         = 5
	testResponseTimeStamp   = int64(1611145887)
	testResponseUUID        = "4498280e-91ba-46d8-9030-6720d3ca6a9b"
	// nolint : lll
	testResponseJSONData = `{"Body":"test body","Header":{"Test-Header":["test value"]},"StatusCode":200,"TimeStamp":1611145887,"UUID":"4498280e-91ba-46d8-9030-6720d3ca6a9b"}`
)

func initHTTPResponse() *http.Response {
	resp := &http.Response{
		Header:     make(http.Header),
		StatusCode: http.StatusOK,
	}
	resp.Header.Add(testRequestHeaderName, testRequestHeaderValue)
	resp.Body = ioutil.NopCloser(bytes.NewBufferString(testResponseBody))
	return resp
}

func TestNewResponseData(t *testing.T) {
	respData := NewResponseData(testResponseHK, testResponseMax, nil)
	require.NotNil(t, respData)
}

func TestResponseData_MarshalBinary(t *testing.T) {
	resp := initHTTPResponse()
	defer resp.Body.Close()

	respData := NewResponseData(testResponseHK, testResponseMax, nil)
	require.NotNil(t, respData)

	err := respData.Read(resp)
	require.Nil(t, err)
	u, err := uuid.Parse(testResponseUUID)
	require.Nil(t, err)
	respData.UUID = u
	respData.TimeStamp = testResponseTimeStamp

	data, err := respData.MarshalBinary()
	require.Equal(t, nil, err)
	require.NotNil(t, data)
	require.Equal(t, testResponseJSONData, string(data))
}

func TestResponseData_UnmarshalBinary(t *testing.T) {
	respData := NewResponseData(testResponseHK, testResponseMax, nil)
	require.NotNil(t, respData)

	err := respData.UnmarshalBinary([]byte(testResponseJSONData))
	require.Equal(t, nil, err)
	require.Equal(t, testResponseBody, respData.Body)
	require.Equal(t, http.StatusOK, respData.StatusCode)
	h := make(http.Header)
	h.Add(testResponseHeaderName, testResponseHeaderValue)
	require.Equal(t, h, respData.Header)
	require.Equal(t, testResponseTimeStamp, respData.TimeStamp)
	u, err := uuid.Parse(testResponseUUID)
	require.Nil(t, err)
	require.Equal(t, u, respData.UUID)
}

func TestResponseData_Write(t *testing.T) {
	resp := initHTTPResponse()
	defer resp.Body.Close()

	respData := NewResponseData(testResponseHK, testResponseMax, nil)
	require.NotNil(t, respData)

	err := respData.Read(resp)
	require.Nil(t, err)

	w := httptest.NewRecorder()
	err = respData.Write(w)
	require.Nil(t, err)

	result := w.Result()
	defer result.Body.Close()

	require.Equal(t, resp.StatusCode, result.StatusCode)
	require.Equal(t, resp.Header, result.Header)
	bodyBytes, err := ioutil.ReadAll(result.Body)
	require.Nil(t, err)
	require.Equal(t, testResponseBody, string(bodyBytes))
}

func TestResponseData_Read(t *testing.T) {
	resp := initHTTPResponse()
	defer resp.Body.Close()

	respData := NewResponseData(testResponseHK, testResponseMax, nil)
	require.NotNil(t, respData)

	err := respData.Read(resp)
	require.Nil(t, err)

	require.Equal(t, testResponseBody, respData.Body)
	require.Equal(t, http.StatusOK, respData.StatusCode)
	h := make(http.Header)
	h.Add(testResponseHeaderName, testResponseHeaderValue)
	require.Equal(t, h, respData.Header)
	require.NotEmpty(t, respData.TimeStamp)
	require.NotEmpty(t, respData.UUID)
}

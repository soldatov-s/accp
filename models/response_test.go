package models

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResponseMarshalBinary(t *testing.T) {
	resp := &Response{
		Body:       "hello",
		Header:     make(http.Header),
		StatusCode: http.StatusOK,
		TimeStamp:  time.Now().Unix(),
	}
	resp.Header.Add("test_header", "test_value")

	m, err := resp.MarshalBinary()
	require.Equal(t, nil, err)

	t.Logf("%+v", m)
	t.Logf("%s", string(m))
}

func TestResponseUnmarshalBinary(t *testing.T) {
	resp := &Response{
		Body:       "hello",
		Header:     make(http.Header),
		StatusCode: http.StatusOK,
		TimeStamp:  time.Now().Unix(),
	}
	resp.Header.Add("test_header", "test_value")

	binData, err := resp.MarshalBinary()
	require.Equal(t, nil, err)

	resp = &Response{}
	err = resp.UnmarshalBinary(binData)
	require.Equal(t, nil, err)

	require.Equal(t, "hello", resp.Body)

	t.Logf("%+v", resp)
}

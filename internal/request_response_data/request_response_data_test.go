package models

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRRDataMarshalBinary(t *testing.T) {
	data := NewRequestResponseData("zzz", 2, nil)
	data.Response = &ResponseData{
		Body:       "hello",
		Header:     make(http.Header),
		StatusCode: http.StatusOK,
		TimeStamp:  time.Now().Unix(),
		UUID:       uuid.New(),
	}

	m, err := data.MarshalBinary()
	require.Equal(t, nil, err)

	t.Logf("%+v", m)
	t.Logf("%s", string(m))
}

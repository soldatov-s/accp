package models

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRRDataMarshalBinary(t *testing.T) {
	data := RRData{}
	data.RData.Refresh.Counter = 1
	data.RData.Refresh.MaxCount = 2
	data.RData.UUID = uuid.New()
	data.RData.Response = &Response{
		Body:       "hello",
		Header:     make(http.Header),
		StatusCode: http.StatusOK,
		TimeStamp:  time.Now().Unix(),
	}

	m, err := data.MarshalBinary()
	require.Equal(t, nil, err)

	t.Logf("%+v", m)
	t.Logf("%s", string(m))
}

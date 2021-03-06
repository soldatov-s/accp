package rrdata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/httputils"
	"github.com/valyala/bytebufferpool"
)

const (
	ResponseCachedHeader = "Accp-Cached"
	ResponseSourceHeader = "X-Cache-Status"
)

type ResponseSource int

const (
	ResponseBack ResponseSource = iota
	ResponseCache
	ResponseBypass
)

func (r ResponseSource) String() string {
	return []string{"MISS", "HIT", "BYPASS"}[r]
}

type ResponseData struct {
	readMu     sync.RWMutex
	Body       string       `json:"body"`
	Header     http.Header  `json:"header"`
	StatusCode int          `json:"status_code"`
	TimeStamp  int64        `json:"time_stamp"`
	UUID       uuid.UUID    `json:"uuid"`
	Refresh    *RefreshData `json:"-"`
}

func NewResponseData(hk string, maxCount int, cache *external.Cache) *ResponseData {
	return &ResponseData{
		Refresh: NewRefreshData(hk, maxCount, cache),
	}
}

func (r *ResponseData) MarshalBinary() (data []byte, err error) {
	return json.Marshal(r)
}

func (r *ResponseData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}

func (r *ResponseData) Write(w http.ResponseWriter, src fmt.Stringer) error {
	r.readMu.RLock()
	defer r.readMu.RUnlock()

	httputils.CopyHeader(w.Header(), r.Header)
	w.Header().Add(ResponseSourceHeader, src.String())
	w.WriteHeader(r.StatusCode)
	_, err := w.Write([]byte(r.Body))
	return err
}

func (r *ResponseData) Read(resp *http.Response) error {
	r.readMu.Lock()
	defer r.readMu.Unlock()

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, err := io.Copy(buf, resp.Body)
	if err != nil {
		return err
	}

	r.Body = buf.String()
	r.Header = make(http.Header)
	httputils.CopyHeader(r.Header, resp.Header)
	r.StatusCode = resp.StatusCode
	r.TimeStamp = time.Now().UTC().Unix()
	r.UUID = uuid.New()
	r.Header.Add(ResponseCachedHeader, strconv.Itoa(int(r.TimeStamp)))

	return nil
}

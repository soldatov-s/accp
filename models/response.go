package models

import (
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/soldatov-s/accp/internal/httputils"
	"github.com/soldatov-s/accp/models/protobuf"
	"github.com/valyala/bytebufferpool"
	"google.golang.org/protobuf/proto"
)

type Response struct {
	readMu     sync.RWMutex
	Body       string
	Header     http.Header
	StatusCode int
	TimeStamp  int64
}

func (r *Response) MarshalBinary() (data []byte, err error) {
	tmp := &protobuf.Response{
		Body:       r.Body,
		StatusCode: int64(r.StatusCode),
		TimeStamp:  r.TimeStamp,
	}

	tmp.Header = &protobuf.Header{
		Header: make(map[string]*protobuf.HeaderList),
	}

	for k, vv := range r.Header {
		for _, v := range vv {
			if tmp.Header.Header[k] == nil {
				tmp.Header.Header[k] = &protobuf.HeaderList{}
			}
			tmp.Header.Header[k].Header = append(tmp.Header.Header[k].Header, v)
		}
	}

	return proto.Marshal(tmp)
}

func (r *Response) UnmarshalBinary(data []byte) error {
	tmp := &protobuf.Response{}
	if err := proto.Unmarshal(data, tmp); err != nil {
		return err
	}

	r.Body = tmp.Body
	r.StatusCode = int(tmp.StatusCode)
	r.TimeStamp = tmp.TimeStamp
	if r.Header == nil {
		r.Header = make(http.Header)
	}

	for k, vv := range tmp.Header.Header {
		r.Header[k] = append(r.Header[k], vv.Header...)
	}

	return nil
}

func (r *Response) Write(w http.ResponseWriter) error {
	r.readMu.RLock()
	defer r.readMu.RUnlock()

	httputils.CopyHeader(w.Header(), r.Header)
	w.WriteHeader(r.StatusCode)
	w.Header().Add("accp-refreshed", strconv.Itoa(int(r.TimeStamp)))
	_, err := w.Write([]byte(r.Body))
	return err
}

func (r *Response) Read(resp *http.Response) error {
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

	return nil
}

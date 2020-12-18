package httputils

import (
	"net/http"

	"github.com/soldatov-s/accp/models/protobuf"
)

func CopyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func CopyHTTPHeaderToProtoHeader(dst protobuf.Header, src http.Header) {
	for k, vv := range src {
		headerList := dst.GetHeader()
		for _, v := range vv {
			headerList[k].Header = append(headerList[k].GetHeader(), v)
		}
	}
}

package limits

import (
	"net/http"
	"strings"
)

// LimitedParamsOfRequest is a map of limited params from http request
type LimitedParamsOfRequest map[string]string

func NewLimitedParamsOfRequest(mc MapConfig, r *http.Request) LimitedParamsOfRequest {
	l := make(LimitedParamsOfRequest)

	for k, v := range mc {
		for _, vv := range v.Header {
			if h := r.Header.Get(vv); h != "" {
				if strings.EqualFold(vv, "authorization") {
					splitToken := strings.Split(h, " ")
					if len(splitToken) < 2 {
						h = splitToken[0]
					} else {
						h = splitToken[1]
					}
				}
				// Always taken client IP
				if strings.EqualFold(vv, "x-forwarded-for") {
					splitIP := strings.Split(h, ",")
					h = splitIP[0]
				}
				l[strings.ToLower(k)] = h
			}
		}

		for _, vv := range v.Cookie {
			if c, err := r.Cookie(vv); err == nil {
				l[strings.ToLower(k)] = c.Value
			}
		}
	}

	return l
}

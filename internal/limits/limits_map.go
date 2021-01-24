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
				if strings.EqualFold(vv, authorizationHeader) {
					splitToken := strings.Split(h, " ")
					if len(splitToken) < 2 {
						h = strings.TrimSpace(splitToken[0])
					} else {
						h = strings.TrimSpace(splitToken[1])
					}
				}
				// Always taken client IP
				if strings.EqualFold(vv, ipHeader) {
					splitIP := strings.Split(h, ",")
					h = strings.TrimSpace(splitIP[0])
				}
				l[strings.ToLower(k)] = h
			}
		}

		for _, vv := range v.Cookie {
			if c, err := r.Cookie(vv); err == nil {
				l[strings.ToLower(k)] = strings.TrimSpace(c.Value)
			}
		}
	}

	return l
}

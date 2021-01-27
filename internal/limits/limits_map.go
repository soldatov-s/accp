package limits

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
)

// LimitedParamsOfRequest is a map of limited params from http request
type LimitedParamsOfRequest map[string]string

// nolint
func limitHash(value string) (string, error) {
	sum := sha256.New().Sum([]byte(value))
	return base64.URLEncoding.EncodeToString(sum), nil
}

func NewLimitedParamsOfRequest(mc MapConfig, r *http.Request) (LimitedParamsOfRequest, error) {
	l := make(LimitedParamsOfRequest)

	var err error
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
				h, err = limitHash(h)
				if err != nil {
					return nil, err
				}
				l[strings.ToLower(k)] = h
			}
		}

		for _, vv := range v.Cookie {
			if c, err := r.Cookie(vv); err == nil {
				h := strings.TrimSpace(c.Value)
				h, err = limitHash(h)
				if err != nil {
					return nil, err
				}
				l[strings.ToLower(k)] = h
			}
		}
	}

	return l, nil
}

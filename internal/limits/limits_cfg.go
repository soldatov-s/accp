package limits

import "time"

type LimitConfig struct {
	// Header is name of header in request for limit
	Header []string
	// Cookie is name of cookie in request for limit
	Cookie []string
	// Limit Count per Time period
	// Conter limits count of request to API
	Counter int
	// PT limits period of requests to API
	PT time.Duration
}

func (lc *LimitConfig) Merge(target *LimitConfig) *LimitConfig {
	result := &LimitConfig{
		Header:  lc.Header,
		Cookie:  lc.Cookie,
		Counter: lc.Counter,
		PT:      lc.PT,
	}

	if target == nil {
		return result
	}

	if len(target.Header) > 0 {
		result.Header = append(result.Header, target.Header...)
	}

	if len(target.Cookie) > 0 {
		result.Cookie = append(result.Cookie, target.Cookie...)
	}

	if target.Counter > 0 {
		result.Counter = target.Counter
	}

	if target.PT > 0 {
		result.PT = target.PT
	}

	return result
}

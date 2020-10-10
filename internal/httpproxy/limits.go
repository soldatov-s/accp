package httpproxy

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/soldatov-s/accp/internal/cache/external"
)

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

type Limit struct {
	Mu         sync.Mutex
	Counter    int
	LastAccess int64 // Unix time
}

type LimitTable map[interface{}]*Limit

func (l *Limit) LoadLimit(name, key string, externalStorage *external.Cache) error {
	if externalStorage != nil {
		if err := externalStorage.JSONGet(key, name+".counter", &l.Counter); err != nil {
			return err
		}
		if err := externalStorage.JSONGet(key, name+".lastaccess", &l.LastAccess); err != nil {
			return err
		}
	}

	return nil
}

func (l *Limit) UpdateLimit(route, key string, externalStorage *external.Cache) error {
	if externalStorage != nil {
		if data, err := json.Marshal(&l.Counter); err != nil {
			return err
		} else if err := externalStorage.JSONSet(route, key+".counter", string(data)); err != nil {
			return err
		}

		data, err := json.Marshal(&l.Counter)
		if err != nil {
			return err
		}

		if err := externalStorage.JSONSet(route, key+".lastaccess", string(data)); err != nil {
			return err
		}
	}

	return nil
}

func (l *Limit) CreateLimit(route, key string, externalStorage *external.Cache) error {
	if externalStorage != nil {
		if data, err := json.Marshal(&l.Counter); err != nil {
			return err
		} else if err := externalStorage.JSONSetNX(route, key+".counter", string(data)); err != nil {
			return err
		}

		data, err := json.Marshal(&l.Counter)
		if err != nil {
			return err
		}

		if err := externalStorage.JSONSetNX(route, key+".lastaccess", string(data)); err != nil {
			return err
		}
	}

	return nil
}

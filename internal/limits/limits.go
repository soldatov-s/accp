package limits

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/soldatov-s/accp/internal/cache/external"
)

// Limit is a current state of a limited parameter from http request
type Limit struct {
	Counter    int   `json:"counter"`
	LastAccess int64 `json:"lastaccess"` // Unix time
}

// LimitTable contains multiple parameters from requests and their current limits
// e.g. we set limit for authorization tokens, we recive the multiple requsts with
// different tokens. LimitTable will be contain each token and its current state of limit
type LimitTable struct {
	List       sync.Map
	clearTimer *time.Timer
	PT         time.Duration
}

func NewLimitTable(c *Config) *LimitTable {
	lt := &LimitTable{
		PT: c.PT,
	}

	lt.clearTimer = time.AfterFunc(lt.PT, lt.clearTable)
	return lt
}

func (lt *LimitTable) clearTable() {
	timeNow := time.Now().UTC()
	lt.List.Range(func(k, v interface{}) bool {
		if time.Unix(v.(*Limit).LastAccess, 0).UTC().Sub(timeNow) > lt.PT {
			lt.List.Delete(k)
		}
		return true
	})

	lt.clearTimer.Reset(lt.PT)
}

func NewLimits(lc MapConfig) map[string]*LimitTable {
	l := make(map[string]*LimitTable)
	for k, c := range lc {
		l[k] = NewLimitTable(c)
	}
	return l
}

func (l *Limit) LoadLimit(route, key string, externalStorage *external.Cache) error {
	if externalStorage == nil {
		return nil
	}

	if err := externalStorage.JSONGet(key+"_"+route, ".", l); err != nil {
		return err
	}

	return nil
}

func (l *Limit) UpdateLimit(route, key string, externalStorage *external.Cache, ttl time.Duration) error {
	if externalStorage == nil {
		return nil
	}

	limitData, err := json.Marshal(l)
	if err != nil {
		return err
	}

	if err := externalStorage.ExternalStorage.JSONSet(key+"_"+route, ".", string(limitData)); err != nil {
		return err
	}

	return externalStorage.ExternalStorage.Expire(key+"_"+route, ttl)
}

func (l *Limit) CreateLimit(route, key string, externalStorage *external.Cache, ttl time.Duration) error {
	if externalStorage == nil {
		return nil
	}

	limitData, err := json.Marshal(l)
	if err != nil {
		return err
	}

	if err := externalStorage.JSONSetNX(key+"_"+route, ".", string(limitData)); err != nil {
		return err
	}

	return externalStorage.ExternalStorage.Expire(key+"_"+route, ttl)
}

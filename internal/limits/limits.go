package limits

import (
	"sync"
	"time"

	"github.com/soldatov-s/accp/internal/cache/external"
)

// Limit is a current state of a limited parameter from http request
type Limit struct {
	Counter    int   `json:"counter"`
	LastAccess int64 `json:"lastaccess"` // Unix time

}

func NewLimit() *Limit {
	return &Limit{
		Counter:    1,
		LastAccess: time.Now().Unix(),
	}
}

// LimitTable contains multiple parameters from requests and their current limits
// e.g. we set limit for authorization tokens, we recive the multiple requsts with
// different tokens. LimitTable will be contain each token and its current state of limit
type LimitTable struct {
	list     sync.Map
	pt       time.Duration
	maxCount int

	clearTimer *time.Timer
	cache      *external.Cache
	route      string
}

func NewLimitTable(route string, c *Config, cache *external.Cache) *LimitTable {
	c.Validate()

	lt := &LimitTable{
		pt:       c.PT,
		maxCount: c.Counter,
		cache:    cache,
		route:    route,
	}

	lt.clearTimer = time.AfterFunc(lt.pt, lt.clearTable)
	return lt
}

func (t *LimitTable) Check(value string, result *bool) error {
	if err := t.Inc(value); err != nil {
		return err
	}

	if v, ok := t.list.Load(value); ok {
		if v.(*Limit).Counter >= t.maxCount {
			*result = true
		}
	}

	return nil
}

func (t *LimitTable) Inc(value string) error {
	var isNew bool

	v, ok := t.list.Load(value)
	if !ok {
		v = NewLimit()
		t.list.Store(value, v)
		isNew = true
	}

	if t.cache != nil && !isNew {
		if err := t.cache.Select(t.route+"_"+value, &v.(*Limit).Counter); err != nil {
			return err
		}
	}

	if v.(*Limit).Counter >= t.maxCount {
		return nil
	}

	v.(*Limit).Counter++
	if t.cache != nil {
		if err := t.cache.LimitTTL(t.route+"_"+value, t.pt); err != nil {
			return err
		}
	}

	return nil
}

func (t *LimitTable) clearTable() {
	timeNow := time.Now().UTC()

	t.list.Range(func(k, v interface{}) bool {
		if time.Unix(v.(*Limit).LastAccess, 0).UTC().Sub(timeNow) > t.pt {
			t.list.Delete(k)
		}
		return true
	})

	t.clearTimer.Reset(t.pt)
}

func NewLimits(route string, lc MapConfig, cache *external.Cache) map[string]*LimitTable {
	l := make(map[string]*LimitTable)
	for k, c := range lc {
		l[k] = NewLimitTable(route, c, cache)
	}
	return l
}

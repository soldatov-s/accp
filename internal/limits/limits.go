package limits

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/soldatov-s/accp/internal/cache/errors"
	"github.com/soldatov-s/accp/internal/cache/external"
)

const (
	defaultClearLimitPeriod = 1 * time.Second
)

// Limit is a current state of a limited parameter from http request
type Limit struct {
	Counter    int64     `json:"counter"`
	LastAccess time.Time `json:"lastaccess"`
}

func NewLimit() *Limit {
	return &Limit{
		Counter:    0,
		LastAccess: time.Now().UTC(),
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
	c.SetDefault()

	lt := &LimitTable{
		pt:       c.TTL,
		maxCount: c.MaxCounter,
		cache:    cache,
		route:    route,
	}

	// Start clear time every second
	lt.clearTimer = time.AfterFunc(defaultClearLimitPeriod, lt.clearTable)
	return lt
}

func (t *LimitTable) Check(value string, result *bool) error {
	*result = false
	if err := t.Inc(value); err != nil {
		return err
	}

	if v, ok := t.list.Load(value); ok {
		if v.(*Limit).Counter > int64(t.maxCount) {
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
		if err := t.cache.Select(t.route+"_"+value, &v.(*Limit).Counter); err != nil && err != errors.ErrNotFound {
			return err
		}
	}

	if v.(*Limit).Counter > int64(t.maxCount) {
		return nil
	}

	atomic.AddInt64(&v.(*Limit).Counter, 1)
	if t.cache != nil {
		if err := t.cache.LimitTTL(t.route+"_"+value, t.pt); err != nil {
			return err
		}
	}

	return nil
}

func (t *LimitTable) clearTable() {
	t.clearTimer.Reset(defaultClearLimitPeriod)
	timeNow := time.Now().UTC()

	t.list.Range(func(k, v interface{}) bool {
		if timeNow.Sub(v.(*Limit).LastAccess) >= t.pt {
			t.list.Delete(k)
		}
		return true
	})
}

func NewLimits(route string, lc MapConfig, cache *external.Cache) map[string]*LimitTable {
	l := make(map[string]*LimitTable)
	for k, c := range lc {
		l[k] = NewLimitTable(route, c, cache)
	}
	return l
}

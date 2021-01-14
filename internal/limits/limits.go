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
	List map[string]*Limit
	mu   sync.Mutex
	PT   time.Duration

	clearTimer *time.Timer
	cache      *external.Cache
	route      string
}

func NewLimitTable(route string, c *Config, cache *external.Cache) *LimitTable {
	c.Validate()

	lt := &LimitTable{
		PT:    c.PT,
		cache: cache,
		route: route,
		List:  make(map[string]*Limit),
	}

	lt.clearTimer = time.AfterFunc(lt.PT, lt.clearTable)
	return lt
}

func (t *LimitTable) Inc(value string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	v, ok := t.List[value]
	if !ok {
		l := NewLimit()
		t.List[value] = l
	}

	if t.cache != nil {
		if err := t.cache.Select(t.route+"_"+value, &v.Counter); err != nil {
			return err
		}
	}

	v.Counter++
	v.LastAccess = time.Now().Unix()
	return t.cache.LimitTTL(t.route+"_"+value, t.PT)
}

func (t *LimitTable) clearTable() {
	timeNow := time.Now().UTC()

	for k, v := range t.List {
		if time.Unix(v.LastAccess, 0).UTC().Sub(timeNow) > t.PT {
			delete(t.List, k)
		}
	}

	t.clearTimer.Reset(t.PT)
}

func NewLimits(route string, lc MapConfig, cache *external.Cache) map[string]*LimitTable {
	l := make(map[string]*LimitTable)
	for k, c := range lc {
		l[k] = NewLimitTable(route, c, cache)
	}
	return l
}

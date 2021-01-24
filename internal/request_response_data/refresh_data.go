package rrdata

import (
	"sync/atomic"

	"github.com/soldatov-s/accp/internal/cache/external"
)

const (
	defaultRefreshPrefix = "refreh_"
)

type RefreshData struct {
	maxCount int
	counter  int64
	cache    *external.Cache
	hk       string
}

func NewRefreshData(hk string, maxCount int, cache *external.Cache) *RefreshData {
	return &RefreshData{
		maxCount: maxCount,
		cache:    cache,
		hk:       hk,
	}
}

func (r *RefreshData) Inc() error {
	if r.maxCount == 0 {
		return nil
	}

	var isNew bool

	if r.counter == 0 {
		isNew = true
	}

	if r.cache != nil && !isNew {
		if err := r.cache.GetLimit(defaultRefreshPrefix+r.hk, &r.counter); err != nil {
			return err
		}
	}

	atomic.AddInt64(&r.counter, 1)
	if r.cache != nil {
		return r.cache.LimitCount(defaultRefreshPrefix+r.hk, r.maxCount)
	}

	if r.counter > int64(r.maxCount) {
		atomic.StoreInt64(&r.counter, 0)
	}

	return nil
}

func (r *RefreshData) Current() int {
	return int(r.counter)
}

func (r *RefreshData) Check() bool {
	return r.counter < int64(r.maxCount)
}

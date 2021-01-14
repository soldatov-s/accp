package models

import (
	"github.com/soldatov-s/accp/internal/cache/external"
)

type RefreshData struct {
	maxCount int
	counter  int
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

func (r *RefreshData) Inc() (*int, error) {
	if r.maxCount == 0 {
		return &r.counter, nil
	}

	if r.cache != nil {
		if err := r.cache.Select("refreh_"+r.hk, &r.counter); err != nil {
			return nil, err
		}
	}

	r.counter++
	if r.cache != nil {
		return &r.counter, r.cache.LimitCount("refreh_"+r.hk, r.maxCount)
	}

	if r.counter > r.maxCount {
		r.counter = 0
	}

	return &r.counter, nil
}

func (r *RefreshData) Current() int {
	return r.counter
}

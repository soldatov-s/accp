package cache

import (
	"context"
	"net/http"
	"sync"

	"github.com/soldatov-s/accp/internal/cache/cachedata"
	"github.com/soldatov-s/accp/internal/cache/errors"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cache/memory"
	rrdata "github.com/soldatov-s/accp/internal/request_response_data"
)

type Cache struct {
	Memory         *memory.Cache
	External       *external.Cache
	waitAnswerList map[string]chan struct{}
	waiteAnswerMu  map[string]*sync.Mutex
}

func NewCache(ctx context.Context, cfg *Config, storage external.Storage) *Cache {
	return &Cache{
		Memory:         memory.NewCache(ctx, cfg.Memory),
		External:       external.NewCache(ctx, cfg.External, storage),
		waitAnswerList: make(map[string]chan struct{}),
		waiteAnswerMu:  make(map[string]*sync.Mutex),
	}
}

func (c *Cache) Add(key string, data cachedata.CacheData) error {
	if err := c.Memory.Add(key, data); err != nil {
		return err
	}

	if c.External == nil {
		return nil
	}

	if err := c.External.Add(key, data); err != nil {
		return err
	}

	return nil
}

func (c *Cache) waitAnswer(hk string, ch chan struct{}) (*rrdata.RequestResponseData, error) {
	<-ch

	var (
		v   interface{}
		err error
	)
	if v, err = c.Memory.Select(hk); err == nil {
		value := v.(*rrdata.RequestResponseData)
		return value, nil
	}
	return nil, err
}

// nolint : cyclomatic
func (c *Cache) Select(key string) (*rrdata.RequestResponseData, error) {
	var value *rrdata.RequestResponseData
	// Search in memory cache
	if v, err := c.Memory.Select(key); err == nil {
		value = v.(*rrdata.RequestResponseData)
	} else if err != errors.ErrNotFound {
		return nil, err
	}

	// If we found key in memory cache we need to expire it in external cache
	// and to check UUID
	if value != nil && c.External != nil {
		if value.Response.StatusCode < http.StatusBadRequest {
			if err := c.External.Expire(key); err != nil {
				return nil, err
			}
		}

		// Checking that item in external cache not changed
		var UUID string
		if err := c.External.GetUUID(key, &UUID); err != nil {
			return nil, err
		}

		if value.Response.UUID.String() == UUID {
			return value, nil
		}
	} else if value != nil {
		return value, nil
	} else if c.External == nil {
		return nil, errors.ErrNotFound
	}

	// Check that we not started to handle the request to redis
	var (
		waitCh chan struct{}
		ok     bool
	)
	if waitCh, ok = c.waitAnswerList[key]; !ok {
		// If we not started to handle the request we need to add lock-channel to map
		var (
			mu *sync.Mutex
			ok bool
		)
		// Create mutex for same requests
		if mu, ok = c.waiteAnswerMu[key]; !ok {
			mu = &sync.Mutex{}
			c.waiteAnswerMu[key] = mu
		}
		mu.Lock()
		if waitCh1, ok1 := c.waitAnswerList[key]; !ok1 {
			ch := make(chan struct{})
			c.waitAnswerList[key] = ch
			mu.Unlock() // unlock mutex fast as possible

			endWait := func() {
				close(ch)
				delete(c.waitAnswerList, key)
				// Delete removes only item from map, GC remove mutex after removed all references to it.
				delete(c.waiteAnswerMu, key)
			}

			value = &rrdata.RequestResponseData{}

			if err := c.External.Select(key, value); err != nil {
				endWait()
				return nil, err
			}

			if err := c.Memory.Add(key, value); err != nil {
				endWait()
				return nil, err
			}
			endWait()
			return value, nil
		} else {
			mu.Unlock()
			return c.waitAnswer(key, waitCh1)
		}
	} else {
		return c.waitAnswer(key, waitCh)
	}
}

func (c *Cache) Delete(key string) error {
	if err := c.Memory.Delete(key); err != nil {
		return err
	}

	if c.External == nil {
		return nil
	}

	return c.External.JSONDelete(key, ".")
}

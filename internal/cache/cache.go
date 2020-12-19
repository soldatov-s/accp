package cache

import (
	"net/http"
	"sync"

	"github.com/soldatov-s/accp/internal/cache/cachedata"
	"github.com/soldatov-s/accp/internal/cache/cacheerrs"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cache/memory"
	accpmodels "github.com/soldatov-s/accp/models"
)

type Config struct {
	Memory   *memory.CacheConfig
	External *external.CacheConfig
}

func (cc *Config) Initilize() error {
	if cc.Memory == nil {
		cc.Memory = &memory.CacheConfig{}
	}

	if err := cc.Memory.Initilize(); err != nil {
		return err
	}

	if cc.External != nil {
		return cc.External.Initilize()
	}

	return nil
}

func (cc *Config) Merge(target *Config) *Config {
	result := &Config{
		Memory:   cc.Memory,
		External: cc.External,
	}

	if target == nil {
		return result
	}

	if target.Memory != nil {
		result.Memory = cc.Memory.Merge(target.Memory)
	}

	if target.External != nil {
		result.External = cc.External.Merge(target.External)
	}

	return result
}

type Cache struct {
	Mem            *memory.Cache
	External       *external.Cache
	WaitAnswerList map[string]chan struct{}
	WaiteAnswerMu  map[string]*sync.Mutex
}

func (c *Cache) Add(key string, data cachedata.CacheData) error {
	if err := c.Mem.Add(key, data); err != nil {
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

func (c *Cache) waitAnswer(hk string, ch chan struct{}) (*accpmodels.RRData, error) {
	<-ch

	var (
		v   interface{}
		err error
	)
	if v, err = c.Mem.Select(hk); err == nil {
		value := v.(*accpmodels.RRData)
		return value, nil
	}
	return nil, err
}

func (c *Cache) Select(key string) (*accpmodels.RRData, error) {
	var (
		value         *accpmodels.RRData
		refreshExpire bool
	)
	refreshExpire = true
	if v, err := c.Mem.Select(key); err == nil {
		value = v.(*accpmodels.RRData)
		if value.Response.StatusCode > http.StatusBadRequest {
			refreshExpire = false
		}
	} else if err != cacheerrs.ErrNotFoundInCache {
		return nil, err
	}

	if c.External == nil {
		return nil, cacheerrs.ErrNotFoundInCache
	}

	if refreshExpire {
		if err := c.External.Expire(key); err != nil {
			return nil, err
		}
	}

	if value != nil {
		// Checking that item in external cache not changed
		var UUID string
		if err := c.External.GetUUID(key, &UUID); err == nil {
			if value.UUID.String() == UUID {
				return value, nil
			}
		}
	}

	// Check that we not started to handle the request to redis
	var (
		waitCh chan struct{}
		ok     bool
	)
	if waitCh, ok = c.WaitAnswerList[key]; !ok {
		// If we not started to handle the request we need to add lock-channel to map
		var (
			mu *sync.Mutex
			ok bool
		)
		// Create mutex for same requests
		if mu, ok = c.WaiteAnswerMu[key]; !ok {
			mu = &sync.Mutex{}
			c.WaiteAnswerMu[key] = mu
		}
		mu.Lock()
		if waitCh1, ok1 := c.WaitAnswerList[key]; !ok1 {
			ch := make(chan struct{})
			c.WaitAnswerList[key] = ch
			mu.Unlock() // unlock mutex fast as possible

			value = &accpmodels.RRData{}

			if err := c.External.Select(key, &value); err != nil {
				return nil, err
			}

			if err := c.Mem.Add(key, value); err != nil {
				return nil, err
			}

			close(ch)
			delete(c.WaitAnswerList, key)
			// Delete removes only item from map, GC remove mutex after removed all references to it.
			delete(c.WaiteAnswerMu, key)
		} else {
			mu.Unlock()
			return c.waitAnswer(key, waitCh1)
		}
	}
	return c.waitAnswer(key, waitCh)
}

func (c *Cache) Delete(key string) error {
	c.Mem.Delete(key)

	if c.External == nil {
		return nil
	}

	return c.External.JSONDelete(key, ".")
}

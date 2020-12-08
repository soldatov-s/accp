package cache

import (
	"net/http"

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
	Mem      *memory.Cache
	External *external.Cache
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
			if value.UUID == UUID {
				return value, nil
			}
		}
	}

	value = &accpmodels.RRData{}

	if err := c.External.Select(key, &value); err != nil {
		return nil, err
	}

	if err := c.Mem.Add(key, value); err != nil {
		return nil, err
	}

	return value, nil
}

package cache

import (
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
	if v, err := c.Mem.Select(key); err == nil {
		return v.(*accpmodels.RRData), nil
	} else if err != cacheerrs.ErrNotFoundInCache {
		return nil, err
	}

	if c.External == nil {
		return nil, cacheerrs.ErrNotFoundInCache
	}

	var value accpmodels.RRData
	err := c.External.Select(key, &value)
	if err != nil {
		return nil, err
	}

	if err := c.Mem.Add(key, &value); err != nil {
		return nil, err
	}

	return &value, nil
}

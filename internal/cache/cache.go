package cache

import (
	"github.com/soldatov-s/accp/internal/cache/cachedata"
	"github.com/soldatov-s/accp/internal/cache/cacheerrs"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cache/memory"
)

type Config struct {
	Memory   *memory.CacheConfig
	External *external.CacheConfig
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

	if c.External != nil {
		return nil
	}

	if err := c.External.Add(key, data); err != nil {
		return err
	}

	return nil
}

func (c *Cache) Select(key string) (cachedata.CacheData, error) {
	if data, err := c.Mem.Select(key); err == nil {
		return data, nil
	} else if err != cacheerrs.ErrNotFoundInCache {
		return nil, err
	}

	if c.External == nil {
		return nil, cacheerrs.ErrNotFoundInCache
	}

	data, err := c.External.Select(key)
	if err != nil {
		return nil, err
	}

	if err := c.Mem.Add(key, data); err != nil {
		return nil, err
	}

	return data, nil
}

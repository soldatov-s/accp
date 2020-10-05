package external

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache/cachedata"
	"github.com/soldatov-s/accp/internal/cache/cacheerrs"
	context "github.com/soldatov-s/accp/internal/ctx"
)

type empty struct{}

type ExternalStorage interface {
	Add(key string, value interface{}, ttl time.Duration) error
	Select(key string, value interface{}) error
	Expire(key string, ttl time.Duration) error
	Update(key string, value interface{}, ttl time.Duration) error
	JSONGet(key, path string, value interface{}) error
	JSONSet(key, path, json string) error
	JSONSetNX(key, path, json string) error
}

type Cache struct {
	ctx             *context.Context
	cfg             *CacheConfig
	log             zerolog.Logger
	externalStorage ExternalStorage
}

type CacheConfig struct {
	KeyPrefix string
	TTL       time.Duration
}

func (cc *CacheConfig) Merge(target *CacheConfig) *CacheConfig {
	result := &CacheConfig{
		KeyPrefix: cc.KeyPrefix,
		TTL:       cc.TTL,
	}

	if target == nil {
		return result
	}

	if target.KeyPrefix != "" {
		result.KeyPrefix = target.KeyPrefix
	}

	if target.TTL > 0 {
		result.TTL = target.TTL
	}

	return result
}

func NewCache(ctx *context.Context, cfg *CacheConfig, externalStorage ExternalStorage) (*Cache, error) {
	if externalStorage == nil {
		return nil, nil
	}

	c := &Cache{ctx: ctx, cfg: cfg, externalStorage: externalStorage}

	c.log = ctx.GetPackageLogger(empty{})
	c.log.Info().Msg("created external cache")

	return c, nil
}

func (c *Cache) Add(key string, data cachedata.CacheData) error {
	err := c.externalStorage.Add(c.cfg.KeyPrefix+key, data, c.cfg.TTL)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("add key %s to cache", key)

	return nil
}

func (c *Cache) Select(key string) (cachedata.CacheData, error) {
	var data cachedata.CacheItem
	err := c.externalStorage.Select(c.cfg.KeyPrefix+key, &data)
	if err != nil {
		return nil, cacheerrs.ErrNotFoundInCache
	}

	err = c.externalStorage.Expire(c.cfg.KeyPrefix+key, c.cfg.TTL)
	if err != nil {
		return nil, cacheerrs.ErrNotFoundInCache
	}

	c.log.Debug().Msgf("select %s from cache", key)

	return data.Data, nil
}

func (c *Cache) Update(key string, data cachedata.CacheData) error {
	err := c.externalStorage.Update(c.cfg.KeyPrefix+key, data, c.cfg.TTL)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("update key %s in cache", key)

	return nil
}

func (c *Cache) JSONGet(key, path string, value interface{}) error {
	err := c.externalStorage.JSONGet(c.cfg.KeyPrefix+key, path, value)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonget %s:%s from cache", key, path)

	return nil
}

func (c *Cache) JSONSet(key, path string, json string) error {
	err := c.externalStorage.JSONSet(c.cfg.KeyPrefix+key, path, json)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonset %s:%s to cache", key, path)

	return nil
}

func (c *Cache) JSONSetNX(key, path string, json string) error {
	err := c.externalStorage.JSONSetNX(c.cfg.KeyPrefix+key, path, json)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonset %s:%s to cache", key, path)

	return nil
}

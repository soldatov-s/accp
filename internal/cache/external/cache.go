package external

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache/cachedata"
	"github.com/soldatov-s/accp/internal/cache/cacheerrs"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/redis"
)

type empty struct{}

const (
	defaultKeyPrefix = "accp_"
	defaultTTL       = 10 * time.Second
)

type Storage interface {
	Add(key string, value interface{}, ttl time.Duration) error
	Select(key string, value interface{}) error
	Expire(key string, ttl time.Duration) error
	Update(key string, value interface{}, ttl time.Duration) error
	JSONGet(key, path string, value interface{}) error
	JSONSet(key, path, json string) error
	JSONSetNX(key, path, json string) error
	NewMutexByID(lockID string, expire, checkInterval time.Duration) redis.IMutex
	JSONDelete(key, path string) error
	JSONNumIncrBy(key, path string, num int) error
	LimitTTL(key string, ttl time.Duration) error
	LimitCount(key string, num int) error
}

type Cache struct {
	ctx             context.Context
	cfg             *Config
	log             zerolog.Logger
	ExternalStorage Storage
}

func NewCache(ctx context.Context, cfg *Config) *Cache {
	externalStorage := redis.Get(ctx)
	if externalStorage == nil {
		return nil
	}

	cfg.Validate()

	c := &Cache{
		ctx:             ctx,
		cfg:             cfg,
		ExternalStorage: externalStorage,
		log:             logger.GetPackageLogger(ctx, empty{}),
	}

	c.log.Info().Msg("created external cache")

	return c
}

func (c *Cache) Add(key string, data cachedata.CacheData) error {
	ttl := c.cfg.TTL
	if data.GetStatusCode() >= http.StatusBadRequest {
		ttl = c.cfg.TTLErr
	}
	err := c.ExternalStorage.Add(c.cfg.KeyPrefix+key, data, ttl)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("add key %s to external cache", key)

	return nil
}

func (c *Cache) Select(key string, data interface{}) error {
	err := c.ExternalStorage.Select(c.cfg.KeyPrefix+key, data)
	if err != nil {
		return cacheerrs.ErrNotFoundInCache
	}

	c.log.Debug().Msgf("select %s from external cache", key)

	return nil
}

func (c *Cache) Expire(key string) error {
	err := c.ExternalStorage.Expire(c.cfg.KeyPrefix+key, c.cfg.TTL)
	if err != nil {
		return cacheerrs.ErrNotFoundInCache
	}

	c.log.Debug().Msgf("expire %s from external cache", key)

	return nil
}

func (c *Cache) Update(key string, data cachedata.CacheData) error {
	err := c.ExternalStorage.Update(c.cfg.KeyPrefix+key, data, c.cfg.TTL)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("update key %s in external cache", key)

	return nil
}

func (c *Cache) JSONGet(key, path string, value interface{}) error {
	err := c.ExternalStorage.JSONGet(c.cfg.KeyPrefix+key, path, value)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonget %s:%s from external cache", key, path)

	return nil
}

func (c *Cache) JSONSet(key, path, json string) error {
	err := c.ExternalStorage.JSONSet(c.cfg.KeyPrefix+key, path, json)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonset %s:%s to external cache", key, path)

	return nil
}

func (c *Cache) JSONSetNX(key, path, json string) error {
	err := c.ExternalStorage.JSONSetNX(c.cfg.KeyPrefix+key, path, json)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonset %s:%s to external cache", key, path)

	return nil
}

func (c *Cache) GetUUID(key string, uuid *string) error {
	err := c.ExternalStorage.JSONGet(c.cfg.KeyPrefix+key, "UUID", uuid)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonget %s:%s from external cache", key, "UUID")

	return nil
}

func (c *Cache) NewMutexByID(lockID string, expire, checkInterval time.Duration) redis.IMutex {
	return c.ExternalStorage.NewMutexByID("mu_"+lockID, expire, checkInterval)
}

func (c *Cache) JSONDelete(key, path string) error {
	err := c.ExternalStorage.JSONDelete(c.cfg.KeyPrefix+key, path)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("delete %s from external cache", key)

	return nil
}

func (c *Cache) LimitTTL(key string, ttl time.Duration) error {
	err := c.ExternalStorage.LimitTTL(c.cfg.KeyPrefix+key, ttl)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("limit ttl %s external cache", key)

	return nil
}

func (c *Cache) LimitCount(key string, num int) error {
	err := c.ExternalStorage.LimitCount(c.cfg.KeyPrefix+key, num)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("limit count %s external cache", key)

	return nil
}

func (c *Cache) JSONNumIncrBy(key, path string, num int) error {
	err := c.ExternalStorage.JSONNumIncrBy(c.cfg.KeyPrefix+key, path, num)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonNumIncrByExecute %s:%s:%d to external cache", key, path, num)

	return nil
}

package external

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache/cachedata"
	"github.com/soldatov-s/accp/internal/cache/cacheerrs"
	context "github.com/soldatov-s/accp/internal/ctx"
	externalcache "github.com/soldatov-s/accp/internal/redis"
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
	NewMutexByID(lockID string, expire, checkInterval time.Duration) (*externalcache.Mutex, error)
}

type Cache struct {
	ctx             *context.Context
	cfg             *CacheConfig
	log             zerolog.Logger
	externalStorage Storage
}

type CacheConfig struct {
	KeyPrefix string
	TTL       time.Duration
	TTLErr    time.Duration
}

func (cc *CacheConfig) Initilize() error {
	if cc.KeyPrefix == "" {
		cc.KeyPrefix = defaultKeyPrefix
	}

	if cc.TTL == 0 {
		cc.TTL = defaultTTL
	}

	return nil
}

func (cc *CacheConfig) Merge(target *CacheConfig) *CacheConfig {
	if cc == nil {
		return target
	}

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

	if target.TTLErr > 0 {
		result.TTLErr = target.TTLErr
	}

	return result
}

func NewCache(ctx *context.Context, cfg *CacheConfig, externalStorage Storage) (*Cache, error) {
	if externalStorage == nil {
		return nil, nil
	}

	c := &Cache{ctx: ctx, cfg: cfg, externalStorage: externalStorage}

	c.log = ctx.GetPackageLogger(empty{})
	c.log.Info().Msg("created external cache")

	return c, nil
}

func (c *Cache) Add(key string, data cachedata.CacheData) error {
	ttl := c.cfg.TTL
	if data.GetStatusCode() >= http.StatusBadRequest {
		ttl = c.cfg.TTLErr
	}
	err := c.externalStorage.Add(c.cfg.KeyPrefix+key, data, ttl)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("add key %s to external cache", key)

	return nil
}

func (c *Cache) Select(key string, data interface{}) error {
	err := c.externalStorage.Select(c.cfg.KeyPrefix+key, data)
	if err != nil {
		return cacheerrs.ErrNotFoundInCache
	}

	c.log.Debug().Msgf("select %s from external cache", key)

	return nil
}

func (c *Cache) Expire(key string) error {
	err := c.externalStorage.Expire(c.cfg.KeyPrefix+key, c.cfg.TTL)
	if err != nil {
		return cacheerrs.ErrNotFoundInCache
	}

	c.log.Debug().Msgf("expire %s from external cache", key)

	return nil
}

func (c *Cache) Update(key string, data cachedata.CacheData) error {
	err := c.externalStorage.Update(c.cfg.KeyPrefix+key, data, c.cfg.TTL)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("update key %s in external cache", key)

	return nil
}

func (c *Cache) JSONGet(key, path string, value interface{}) error {
	err := c.externalStorage.JSONGet(c.cfg.KeyPrefix+key, path, value)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonget %s:%s from external cache", key, path)

	return nil
}

func (c *Cache) JSONSet(key, path, json string) error {
	err := c.externalStorage.JSONSet(c.cfg.KeyPrefix+key, path, json)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonset %s:%s to external cache", key, path)

	return nil
}

func (c *Cache) JSONSetNX(key, path, json string) error {
	err := c.externalStorage.JSONSetNX(c.cfg.KeyPrefix+key, path, json)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonset %s:%s to external cache", key, path)

	return nil
}

func (c *Cache) GetUUID(key string, uuid *string) error {
	err := c.externalStorage.JSONGet(c.cfg.KeyPrefix+key, "UUID", uuid)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonget %s:%s from external cache", key, "UUID")

	return nil
}

func (c *Cache) NewMutexByID(lockID string, expire, checkInterval time.Duration) (*externalcache.Mutex, error) {
	return c.externalStorage.NewMutexByID(lockID, expire, checkInterval)
}

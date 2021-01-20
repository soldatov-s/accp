package memory

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache/cachedata"
	"github.com/soldatov-s/accp/internal/cache/errors"
	"github.com/soldatov-s/accp/internal/logger"
)

type empty struct{}

type Cache struct {
	ctx           context.Context
	cfg           *Config
	log           zerolog.Logger
	clearTimer    *time.Timer
	clearErrTimer *time.Timer
	storage       sync.Map
}

func NewCache(ctx context.Context, cfg *Config) *Cache {
	cfg.SetDefault()

	c := &Cache{
		ctx: ctx,
		cfg: cfg,
		log: logger.GetPackageLogger(ctx, empty{}),
	}

	c.log.Info().Msg("created inmemory cache")

	if c.cfg.TTL > 0 {
		c.clearTimer = time.AfterFunc(c.cfg.TTL, c.ClearCache)
	}

	if c.cfg.TTLErr > 0 {
		c.clearErrTimer = time.AfterFunc(c.cfg.TTLErr, c.ClearErrCache)
	}

	return c
}

func (c *Cache) Add(key string, data interface{}) error {
	if _, ok := c.storage.Load(key); !ok {
		c.storage.Store(key, &cachedata.CacheItem{
			Data:      data,
			TimeStamp: time.Now().UTC(),
		})

		c.log.Debug().Msgf("add key %s to cache", key)
	}
	return nil
}

func (c *Cache) Select(key string) (interface{}, error) {
	if v, ok := c.storage.Load(key); ok {
		c.log.Debug().Msgf("select %s from inmemory cache", key)
		cacheItem := v.(*cachedata.CacheItem)
		cacheData := cacheItem.Data.(cachedata.CacheData)
		// expire only good status code
		if cacheData.GetStatusCode() < http.StatusBadRequest {
			cacheItem.TimeStamp = time.Now()
		}
		return cacheItem.Data, nil
	}

	return nil, errors.ErrNotFound
}

func (c *Cache) Delete(key string) error {
	c.storage.Delete(key)

	c.log.Debug().Msgf("delete %s from inmemory cache", key)
	return nil
}

func (c *Cache) Size() int {
	length := 0

	c.storage.Range(func(_, _ interface{}) bool {
		length++

		return true
	})

	c.log.Debug().Msgf("cache size is %d", length)

	return length
}

func (c *Cache) clear(k, v interface{}, timeNow time.Time) bool {
	cacheItem := v.(*cachedata.CacheItem)
	cacheData := cacheItem.Data.(cachedata.CacheData)
	if (cacheData.GetStatusCode() < http.StatusBadRequest && timeNow.Sub(cacheItem.TimeStamp) > c.cfg.TTL) ||
		(cacheData.GetStatusCode() >= http.StatusBadRequest && timeNow.Sub(cacheItem.TimeStamp) > c.cfg.TTLErr) {
		c.storage.Delete(k)
	}

	c.log.Debug().Msgf("remove expired from cache: %s", k)

	return true
}

func (c *Cache) ClearCache() {
	timeNow := time.Now().UTC()
	c.storage.Range(func(k, v interface{}) bool {
		return c.clear(k, v, timeNow)
	})

	c.clearTimer.Reset(c.cfg.TTL)
}

func (c *Cache) ClearErrCache() {
	timeNow := time.Now().UTC()
	c.storage.Range(func(k, v interface{}) bool {
		return c.clear(k, v, timeNow)
	})

	c.clearErrTimer.Reset(c.cfg.TTLErr)
}

func (c *Cache) Range(f func(key, value interface{}) bool) {
	c.storage.Range(f)
}

package memory

import (
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/cache/cachedata"
	"github.com/soldatov-s/accp/internal/cache/cacheerrs"
	context "github.com/soldatov-s/accp/internal/ctx"
)

type empty struct{}

const (
	defaultTTL = 10 * time.Second
)

type CacheConfig struct {
	TTL    time.Duration
	TTLErr time.Duration
}

func (cc *CacheConfig) Initilize() error {
	if cc.TTL == 0 {
		cc.TTL = defaultTTL
	}

	return nil
}

func (cc *CacheConfig) Merge(target *CacheConfig) *CacheConfig {
	result := &CacheConfig{
		TTL: cc.TTL,
	}

	if target == nil {
		return result
	}

	if target.TTL > 0 {
		result.TTL = target.TTL
	}

	if target.TTLErr > 0 {
		result.TTLErr = target.TTLErr
	}

	return result
}

type Cache struct {
	ctx           *context.Context
	cfg           *CacheConfig
	log           zerolog.Logger
	clearTimer    *time.Timer
	clearErrTimer *time.Timer
	sync.Map
}

func NewCache(ctx *context.Context, cfg *CacheConfig) (*Cache, error) {
	c := &Cache{ctx: ctx, cfg: cfg}

	c.log = ctx.GetPackageLogger(empty{})
	c.log.Info().Msg("created inmemory cache")

	if c.cfg.TTL > 0 {
		c.clearTimer = time.AfterFunc(c.cfg.TTL, c.ClearCache)
	}

	if c.cfg.TTLErr > 0 {
		c.clearErrTimer = time.AfterFunc(c.cfg.TTLErr, c.ClearErrCache)
	}

	return c, nil
}

func (c *Cache) Add(key string, data interface{}) error {
	if _, ok := c.Load(key); !ok {
		c.Store(key, &cachedata.CacheItem{
			Data:      data,
			TimeStamp: time.Now().UTC(),
		})

		c.log.Debug().Msgf("add key %s to cache", key)
	}
	return nil
}

func (c *Cache) Select(key string) (interface{}, error) {
	if v, ok := c.Load(key); ok {
		c.log.Debug().Msgf("select %s from inmemory cache", key)
		cacheItem := v.(*cachedata.CacheItem)
		cacheItem.TimeStamp = time.Now()
		return cacheItem.Data, nil
	}

	return nil, cacheerrs.ErrNotFoundInCache
}

func (c *Cache) Size() int {
	length := 0

	c.Range(func(_, _ interface{}) bool {
		length++

		return true
	})

	c.log.Debug().Msgf("cache size is %d", length)

	return length
}

func (c *Cache) ClearCache() {
	timeNow := time.Now().UTC()
	c.Range(func(k, v interface{}) bool {
		if timeNow.Sub(v.(*cachedata.CacheItem).TimeStamp) > c.cfg.TTL {
			c.Delete(k)
			c.log.Debug().Msgf("remove expired from cache: %s", k)
		}
		return true
	})

	c.clearTimer.Reset(c.cfg.TTL)
}

func (c *Cache) ClearErrCache() {
	timeNow := time.Now().UTC()
	c.Range(func(k, v interface{}) bool {
		cacheItem := v.(*cachedata.CacheItem)
		cacheData := cacheItem.Data.(cachedata.CacheData)
		if cacheData.GetStatusCode() >= http.StatusBadRequest && timeNow.Sub(cacheItem.TimeStamp) > c.cfg.TTLErr {
			c.Delete(k)
			c.log.Debug().Msgf("remove expired from cache: %s", k)
		}
		return true
	})

	c.clearErrTimer.Reset(c.cfg.TTLErr)
}

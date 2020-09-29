package memory

import (
	"encoding"
	"log"
	"sync"
	"time"
)

type Cache struct {
	sync.Map
}

// CacheData is a data which putting in cache
type CacheData interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

// CacheItem is an item of cache
type CacheItem struct {
	Data      CacheData
	TimeStamp time.Time
	UUID      string
}

func (c *Cache) Add(key string, data CacheData) error {
	if _, ok := c.Load(key); !ok {
		c.Store(key, &CacheItem{
			Data:      data,
			TimeStamp: time.Now().UTC(),
		})

		log.Printf("add key %s to cache", key)
	}
	return nil
}

func (c *Cache) Select(key string) (CacheData, error) {
	if v, ok := c.Load(key); ok {
		dd := v.(*CacheItem)
		// log.Printf("select %s from cache", key)
		return dd.Data, nil
	}

	return nil, ErrNotFoundInCache
}

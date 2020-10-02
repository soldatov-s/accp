package cachedata

import (
	"encoding"
	"time"
)

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

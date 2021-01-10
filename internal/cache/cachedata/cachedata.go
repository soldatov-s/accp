package cachedata

import (
	"encoding"
	"time"
)

// CacheData is a data which putting in cache
type CacheData interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	GetStatusCode() int
}

// CacheItem is an item of cache
type CacheItem struct {
	Data      interface{}
	TimeStamp time.Time
}

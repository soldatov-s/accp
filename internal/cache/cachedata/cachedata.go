package cachedata

import (
	"encoding"
	"encoding/json"
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

func (c *CacheItem) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, c)
}

func (c *CacheItem) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

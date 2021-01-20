package memory

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/cache/cachedata"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/meta"
	"github.com/stretchr/testify/require"
)

type testCacheData struct {
	Name       string
	StatusCode int
}

func (d *testCacheData) MarshalBinary() ([]byte, error) {
	return json.Marshal(d)
}

func (d *testCacheData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, d)
}

func (d *testCacheData) GetStatusCode() int {
	return d.StatusCode
}

func initApp(ctx context.Context) context.Context {
	return meta.SetAppInfo(ctx, "accp", "", "", "", "test")
}

func initLogger(ctx context.Context) context.Context {
	// Registrate logger
	logCfg := &logger.Config{
		Level:           logger.LoggerLevelDebug,
		NoColoredOutput: true,
		WithTrace:       false,
	}
	ctx = logger.RegistrateAndInitilize(ctx, logCfg)

	return ctx
}

func initConfig() *Config {
	return &Config{
		TTL:    10 * time.Second,
		TTLErr: 5 * time.Second,
	}
}

func initCache(t *testing.T) *Cache {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()
	c := NewCache(ctx, cfg)
	require.NotNil(t, c)

	return c
}
func TestNewCache(t *testing.T) {
	_ = initCache(t)
}

func TestAdd(t *testing.T) {
	c := initCache(t)

	tesdData := &testCacheData{Name: "test data", StatusCode: 200}

	_ = c.Add("test", tesdData)
	v, ok := c.storage.Load("test")
	require.True(t, ok)
	require.NotNil(t, v)

	item, ok := v.(*cachedata.CacheItem)
	require.True(t, ok)

	require.Equal(t, tesdData, item.Data)
}

func TestSelect(t *testing.T) {
	c := initCache(t)

	tesdData := &testCacheData{Name: "test data", StatusCode: 200}

	_ = c.Add("test", tesdData)
	item, err := c.Select("test")
	require.Nil(t, err)
	require.NotNil(t, item)

	require.Equal(t, tesdData, item)
}

func TestDelete(t *testing.T) {
	c := initCache(t)

	tesdData := &testCacheData{Name: "test data", StatusCode: 200}

	_ = c.Add("test", tesdData)
	v, ok := c.storage.Load("test")
	require.True(t, ok)
	require.NotNil(t, v)

	_ = c.Delete("test")
	v, ok = c.storage.Load("test")
	require.False(t, ok)
	require.Nil(t, v)
}

func TestSize(t *testing.T) {
	c := initCache(t)

	tesdData := &testCacheData{Name: "test data", StatusCode: 200}

	_ = c.Add("test", tesdData)
	cnt := c.Size()
	require.Equal(t, 1, cnt)
	_ = c.Delete("test")
	cnt = c.Size()
	require.Equal(t, 0, cnt)
}

func TestClearCache(t *testing.T) {
	c := initCache(t)

	tesdData := &testCacheData{Name: "test data", StatusCode: 200}

	_ = c.Add("test", tesdData)
	v, ok := c.storage.Load("test")
	require.True(t, ok)
	item, ok := v.(*cachedata.CacheItem)
	require.True(t, ok)

	item.TimeStamp = item.TimeStamp.Add(-10 * time.Second)
	c.ClearCache()
	cnt := c.Size()
	require.Equal(t, 0, cnt)
}

func TestClearErrCache(t *testing.T) {
	c := initCache(t)

	tests := []struct {
		name              string
		testData          *testCacheData
		expectedCacheSize int
	}{
		{
			name:              "status code 400",
			testData:          &testCacheData{Name: "test data", StatusCode: 400},
			expectedCacheSize: 0,
		},
		{
			name:              "status code 200",
			testData:          &testCacheData{Name: "test data", StatusCode: 200},
			expectedCacheSize: 1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_ = c.Add("test", tt.testData)
			v, ok := c.storage.Load("test")
			require.True(t, ok)
			item, ok := v.(*cachedata.CacheItem)
			require.True(t, ok)

			item.TimeStamp = item.TimeStamp.Add(-5 * time.Second)
			c.ClearErrCache()
			cnt := c.Size()
			require.Equal(t, tt.expectedCacheSize, cnt)
		})
	}
}

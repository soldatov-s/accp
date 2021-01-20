package cache

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soldatov-s/accp/internal/cache/external"
	"github.com/soldatov-s/accp/internal/cache/memory"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/meta"
	"github.com/soldatov-s/accp/internal/redis"
	rrdata "github.com/soldatov-s/accp/internal/request_response_data"
	"github.com/soldatov-s/accp/x/dockertest"
	"github.com/soldatov-s/accp/x/test_helpers/resilience"
	"github.com/stretchr/testify/require"
)

const (
	defaultKey = "test"
)

type testCacheData struct {
	Name       string
	StatusCode int
}

type testCacheDataWithUUID struct {
	Data *testCacheData
	UUID uuid.UUID
}

func (d *testCacheDataWithUUID) MarshalBinary() ([]byte, error) {
	return json.Marshal(d)
}

func (d *testCacheDataWithUUID) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, d)
}

func (d *testCacheDataWithUUID) GetStatusCode() int {
	return d.Data.StatusCode
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

func initExternal(ctx context.Context, t *testing.T, dsn string) *redis.Client {
	if dsn == "" {
		t.Log("dsn is empty")
		return nil
	}

	cfg := &redis.Config{
		MinIdleConnections:    10,
		MaxOpenedConnections:  30,
		MaxConnectionLifetime: 30 * time.Second,
	}

	cfg.DSN = dsn

	t.Logf("connecting to redis: %s", dsn)

	client, err := redis.NewClient(ctx, cfg)
	require.Nil(t, err)
	require.NotNil(t, client)

	err = resilience.Retry(
		t,
		time.Second*5,
		time.Minute*5,
		func() (err error) {
			err = client.Start()
			return err
		},
	)

	require.Nil(t, err)
	require.NotNil(t, client)

	t.Logf("connected to redis: %s", dsn)

	return client
}

func initConfig() *Config {
	return &Config{
		Memory: &memory.Config{
			TTL:    5 * time.Second,
			TTLErr: 3 * time.Second,
		},
		External: &external.Config{
			TTL:       10 * time.Second,
			TTLErr:    6 * time.Second,
			KeyPrefix: "accp_",
		},
	}
}

func initCache(t *testing.T, dsn string) (*Cache, *redis.Client) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()

	externalSorage := initExternal(ctx, t, dsn)
	c := NewCache(ctx, cfg, externalSorage)
	require.NotNil(t, c)

	return c, externalSorage
}

func TestNewCache(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "external cache is not nil",
			testFunc: func() {
				dsn, err := dockertest.RunRedis()
				require.Nil(t, err)
				defer dockertest.KillAllDockers()

				c, _ := initCache(t, dsn)
				require.NotNil(t, c.Memory)
				require.NotNil(t, c.External)
			},
		},
		{
			name: "external cache is nil",
			testFunc: func() {
				c, _ := initCache(t, "")
				require.NotNil(t, c.Memory)
				require.Nil(t, c.External)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc()
		})
	}
}

func TestAdd(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c, client := initCache(t, dsn)

	cfg := initConfig()

	tesdData := &testCacheDataWithUUID{
		Data: &testCacheData{Name: "test data", StatusCode: 200},
		UUID: uuid.New(),
	}
	err = c.Add(defaultKey, tesdData)
	require.Nil(t, err)

	// Test get item from external
	var resultExt testCacheDataWithUUID
	err = c.External.Select(defaultKey, &resultExt)
	require.Nil(t, err)
	require.Equal(t, tesdData, &resultExt)

	d, err := client.Conn.TTL(client.Conn.Context(), cfg.External.KeyPrefix+defaultKey).Result()
	require.Nil(t, err)
	require.Equal(t, cfg.External.TTL, d)

	// Test get item from inmemory
	resultMemory, err := c.Memory.Select(defaultKey)
	require.Nil(t, err)
	require.Equal(t, tesdData, resultMemory)

	// Test add StatusCode 400
	tesdData.Data.StatusCode = http.StatusBadRequest
	err = c.Add(defaultKey, tesdData)
	require.Nil(t, err)
	d, err = client.Conn.TTL(client.Conn.Context(), cfg.External.KeyPrefix+defaultKey).Result()
	require.Nil(t, err)
	require.Equal(t, cfg.External.TTLErr, d)
}

func initRefreshData(cache *external.Cache) *rrdata.RefreshData {
	return rrdata.NewRefreshData(defaultKey, 5, cache)
}

func initRequestResponseData(cache *external.Cache) *rrdata.RequestResponseData {
	testData := rrdata.NewRequestResponseData(defaultKey, 5, cache)
	testData.Response.Body = "test body"
	testData.Response.StatusCode = http.StatusOK

	return testData
}

func TestSelect(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "external cache is not nil",
			testFunc: func() {
				dsn, err := dockertest.RunRedis()
				require.Nil(t, err)
				defer dockertest.KillAllDockers()

				cfg := initConfig()
				c, client := initCache(t, dsn)
				testData := initRequestResponseData(c.External)

				err = c.Add(defaultKey, testData)
				require.Nil(t, err)

				// Sleep for partial expire
				time.Sleep(1 * time.Second)
				result, err := c.Select(defaultKey)
				require.Nil(t, err)
				require.NotNil(t, result)
				require.Equal(t, testData, result)

				d, err := client.Conn.TTL(client.Conn.Context(), cfg.External.KeyPrefix+defaultKey).Result()
				require.Nil(t, err)
				require.Equal(t, cfg.External.TTL, d)
			},
		},
		{
			name: "memory cache is expire",
			testFunc: func() {
				dsn, err := dockertest.RunRedis()
				require.Nil(t, err)
				defer dockertest.KillAllDockers()

				c, _ := initCache(t, dsn)
				testData := initRequestResponseData(c.External)

				err = c.Add(defaultKey, testData)
				require.Nil(t, err)

				err = c.Memory.Delete(defaultKey)
				require.Nil(t, err)

				result, err := c.Select(defaultKey)
				require.Nil(t, err)
				require.NotNil(t, result)
				result.Response.Refresh = initRefreshData(c.External)
				require.Equal(t, testData.Response, result.Response)
			},
		},
		{
			name: "external cache is nil",
			testFunc: func() {
				c, _ := initCache(t, "")
				testData := initRequestResponseData(nil)

				err := c.Add(defaultKey, testData)
				require.Nil(t, err)

				result, err := c.Select(defaultKey)
				require.Nil(t, err)
				require.NotNil(t, result)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc()
		})
	}
}

// This test tests multiple requests to external cache
func TestExternalCacheMultipleRequests(t *testing.T) {
	workers := 10

	barrier := make(chan struct{})
	errorsCh := make(chan error, workers)

	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c, _ := initCache(t, dsn)
	testData := initRequestResponseData(c.External)

	err = c.Add(defaultKey, testData)
	require.Nil(t, err)

	err = c.Memory.Delete(defaultKey)
	require.Nil(t, err)

	var wg sync.WaitGroup
	go func() {
		for w := 0; w < workers; w++ {
			t.Logf("start worker %d", w)
			wg.Add(1)
			go func() {
				defer wg.Done()
				// nolint
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
				// all workers will block here until the for loop above has launched all the worker go-routines
				// this is to ensure we fire all the workers off at the same
				<-barrier

				t.Log("take answer from cache")
				v, err := c.Select(defaultKey)
				require.Nil(t, err)
				require.NotNil(t, v)

				err = nil
				if v.GetStatusCode() != http.StatusOK {
					err = errors.New("concurent access to external cache")
				} else {
					var result rrdata.RequestResponseData
					err1 := c.External.Select(defaultKey, &result)
					require.Nil(t, err1)

					t.Logf("status code in memory cache %d, in external cache %d", v.GetStatusCode(), result.GetStatusCode())
				}

				errorsCh <- err
			}()
		}

		// wait until all workers have completed their work
		wg.Wait()
		close(errorsCh)
	}()

	stopCounter := false
	go func() {
		// nolint
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		<-barrier
		for {
			testData.Response.StatusCode++
			err := c.External.Update(defaultKey, testData)
			require.Nil(t, err)
			if stopCounter {
				break
			}
		}
	}()

	// let the race begin!
	// all worker go-routines will now attempt to hit the "CachedHandler" method
	close(barrier)

	var successCount int
	for err := range errorsCh {
		if err != nil {
			t.Errorf("failed: %s", err)
		} else {
			successCount++
		}
	}

	stopCounter = true

	require.Equal(t, workers, successCount)
}

func TestDelete(t *testing.T) {
	dsn, err := dockertest.RunRedis()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c, _ := initCache(t, dsn)
	testData := initRequestResponseData(c.External)

	err = c.Add(defaultKey, testData)
	require.Nil(t, err)

	result, err := c.Select(defaultKey)
	require.Nil(t, err)
	require.NotNil(t, result)
	require.Equal(t, testData, result)

	err = c.Delete(defaultKey)
	require.Nil(t, err)

	result, err = c.Select(defaultKey)
	require.NotNil(t, err)
	require.Nil(t, result)
}

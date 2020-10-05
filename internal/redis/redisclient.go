package externalcache

import (
	"time"

	"github.com/KromDaniel/rejonson"
	"github.com/go-redis/redis"
	"github.com/rs/zerolog"
	context "github.com/soldatov-s/accp/internal/ctx"
)

type empty struct{}

type RedisConfig struct {
	DSN                   string
	MinIdleConnections    int
	MaxOpenedConnections  int
	MaxConnectionLifetime time.Duration
}

type RedisClient struct {
	*rejonson.Client
	ctx *context.Context
	log zerolog.Logger
}

func NewRedisClient(ctx *context.Context, cfg *RedisConfig) (*RedisClient, error) {
	// Connect to database.
	connOptions, err := redis.ParseURL(cfg.DSN)
	if err != nil {
		return nil, err
	}

	// Set connection pooling options.
	connOptions.MaxConnAge = cfg.MaxConnectionLifetime
	connOptions.MinIdleConns = cfg.MinIdleConnections
	connOptions.PoolSize = cfg.MaxOpenedConnections

	r := &RedisClient{Client: rejonson.ExtendClient(redis.NewClient(connOptions))}
	if r.Ping().Err() != nil {
		return nil, err
	}

	r.ctx = ctx
	r.log = ctx.GetPackageLogger(empty{})

	r.log.Info().Msg("Redis connection established")
	return r, nil
}

func (r *RedisClient) Add(key string, value interface{}, ttl time.Duration) error {
	_, err := r.SetNX(key, value, ttl).Result()
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("add key %s to cache", key)

	return nil
}

func (r *RedisClient) Select(key string, value interface{}) error {
	cmdString := r.Get(key)
	_, err := cmdString.Result()

	if err != nil {
		return err
	}

	err = cmdString.Scan(value)
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("select %s from cache", key)

	return nil
}

func (r *RedisClient) Expire(key string, ttl time.Duration) error {
	cmdBool := r.Client.Expire(key, ttl)
	_, err := cmdBool.Result()
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("expire %s in cache", key)

	return nil
}

func (r *RedisClient) Update(key string, value interface{}, ttl time.Duration) error {
	_, err := r.Set(key, value, ttl).Result()
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("update key %s in cache", key)

	return nil
}

// JSONGet item from cache by key.
func (r *RedisClient) JSONGet(key, path string, value interface{}) error {
	cmdString := r.JsonGet(key, path)
	_, err := cmdString.Result()

	if err != nil {
		return err
	}

	err = cmdString.Scan(value)
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("JSONGet value by key %s from cache, path %s", key, path)

	return nil
}

// JSONSet item in cache by key.
func (r *RedisClient) JSONSet(key, path, json string) error {
	_, err := r.JsonSet(key, path, json).Result()
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("JsonSet key %s in cache, path %s, json %s", key, path, json)

	return nil
}

// JSONSetNX item in cache by key.
func (r *RedisClient) JSONSetNX(key, path, json string) error {
	_, err := r.JsonSet(key, path, json, "NX").Result()
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("JsonSetNX key %s in cache, path %s, json %s", key, path, json)

	return nil
}

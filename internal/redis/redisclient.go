package externalcache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
	intcxt "github.com/soldatov-s/accp/internal/ctx"
	"github.com/soldatov-s/accp/x/rejson"
)

type empty struct{}

type RedisConfig struct {
	DSN                   string
	MinIdleConnections    int
	MaxOpenedConnections  int
	MaxConnectionLifetime time.Duration
}

type RedisClient struct {
	*rejson.Client
	ctx    context.Context
	intctx *intcxt.Context
	log    zerolog.Logger
}

func NewRedisClient(ctx *intcxt.Context, cfg *RedisConfig) (*RedisClient, error) {
	// Connect to database.
	connOptions, err := redis.ParseURL(cfg.DSN)
	if err != nil {
		return nil, err
	}

	// Set connection pooling options.
	connOptions.MaxConnAge = cfg.MaxConnectionLifetime
	connOptions.MinIdleConns = cfg.MinIdleConnections
	connOptions.PoolSize = cfg.MaxOpenedConnections

	r := &RedisClient{ctx: context.Background()}

	client := redis.NewClient(connOptions)
	r.Client = rejson.ExtendClient(r.ctx, client)

	if err := r.Ping(r.ctx).Err(); err != nil {
		return nil, err
	}

	r.intctx = ctx
	r.log = ctx.GetPackageLogger(empty{})

	r.log.Info().Msg("Redis connection established")
	return r, nil
}

func (r *RedisClient) Add(key string, value interface{}, ttl time.Duration) error {
	err := r.JsonSetWithExpire(key, ".", value, ttl)
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("add key %s to cache", key)

	return nil
}

func (r *RedisClient) Select(key string, value interface{}) error {
	cmdString := r.Get(r.ctx, key)
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
	cmdBool := r.Client.Expire(r.ctx, key, ttl)
	_, err := cmdBool.Result()
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("expire %s in cache", key)

	return nil
}

func (r *RedisClient) Update(key string, value interface{}, ttl time.Duration) error {
	_, err := r.Set(r.ctx, key, value, ttl).Result()
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

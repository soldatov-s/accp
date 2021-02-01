package redis

import (
	"context"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/metrics"
	"github.com/soldatov-s/accp/internal/utils"
	"github.com/soldatov-s/accp/x/rejson"
)

type empty struct{}

type Client struct {
	Conn *rejson.Client
	opt  *redis.Options
	ctx  context.Context
	log  zerolog.Logger
	cfg  *Config

	// Metrics
	metrics.Service
}

func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, nil
	}

	cfg.SetDefault()

	// Connect to database.
	connOptions, err := cfg.Options()
	if err != nil {
		return nil, err
	}

	r := &Client{
		ctx: ctx,
		cfg: cfg,
		log: logger.GetPackageLogger(ctx, empty{}),
		opt: connOptions,
	}

	r.log.Info().Msg("redis client created")

	return r, nil
}

func (r *Client) Start() error {
	client := redis.NewClient(r.opt)
	if err := client.Ping(r.ctx).Err(); err != nil {
		return err
	}

	r.Conn = rejson.ExtendClient(client)

	r.log.Info().Msg("redis connection established")

	return nil
}

func (r *Client) Add(key string, value interface{}, ttl time.Duration) error {
	err := r.Conn.JSONSetWithExpire(r.ctx, key, ".", value, ttl)
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("add key %s to cache", key)

	return nil
}

func (r *Client) Select(key string, value interface{}) error {
	cmdString := r.Conn.JSONGet(r.ctx, key)
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

func (r *Client) Expire(key string, ttl time.Duration) error {
	cmdBool := r.Conn.Expire(r.ctx, key, ttl)
	_, err := cmdBool.Result()
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("expire %s in cache", key)

	return nil
}

func (r *Client) Update(key string, value interface{}, ttl time.Duration) error {
	err := r.Conn.JSONSetWithExpire(r.ctx, key, ".", value, ttl)
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("add key %s to cache", key)

	return nil
}

// JSONGet item from cache by key.
func (r *Client) JSONGet(key, path string, value interface{}) error {
	cmdString := r.Conn.JSONGet(r.ctx, key, path)
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
func (r *Client) JSONSet(key, path, json string) error {
	_, err := r.Conn.JSONSet(r.ctx, key, path, json).Result()
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("JsonSet key %s in cache, path %s, json %s", key, path, json)

	return nil
}

// JSONSetNX item in cache by key.
func (r *Client) JSONSetNX(key, path, json string) error {
	_, err := r.Conn.JSONSet(r.ctx, key, path, json, "NX").Result()
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("JsonSetNX key %s in cache, path %s, json %s", key, path, json)

	return nil
}

func (r *Client) JSONDelete(key, path string) error {
	_, err := r.Conn.JSONDel(r.ctx, key, path).Result()
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("JsonDelete key %s in cache", key)

	return nil
}

func formatSec(dur time.Duration) int64 {
	if dur > 0 && dur < time.Second {
		return 1
	}
	return int64(dur / time.Second)
}

func (r *Client) LimitTTL(key string, ttl time.Duration) error {
	_, err := r.Conn.Eval(r.ctx,
		`local current
	current = redis.call("incr",KEYS[1])
	if tonumber(current) == 1 then
		redis.call("expire",KEYS[1],ARGV[1])
	end`, []string{key}, formatSec(ttl)).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	r.log.Debug().Msgf("limit key %s in cache", key)

	return nil
}

func (r *Client) LimitCount(key string, num int) error {
	_, err := r.Conn.Eval(r.ctx,
		`local current
	current = redis.call("incr",KEYS[1])
	if tonumber(current) > tonumber(ARGV[1]) then
		redis.call("set",KEYS[1],0)
	end`, []string{key}, num).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	r.log.Debug().Msgf("limit key %s in cache", key)

	return nil
}

func (r *Client) GetLimit(key string, value interface{}) error {
	cmdString := r.Conn.Get(r.ctx, key)
	_, err := cmdString.Result()

	if err != nil {
		return err
	}

	err = cmdString.Scan(value)
	if err != nil {
		return err
	}

	r.log.Debug().Msgf("get limit %s from cache", key)

	return nil
}

// GetMetrics return map of the metrics from cache connection
func (r *Client) GetMetrics() metrics.MapMetricsOptions {
	_ = r.Service.GetMetrics()
	r.Metrics[ProviderName+"_status"] = &metrics.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: ProviderName + "_status",
				Help: ProviderName + " status link to " + utils.RedactedDSN(r.cfg.DSN),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(0)
			_, err := r.Conn.Ping(r.ctx).Result()
			if err == nil {
				(m.(prometheus.Gauge)).Set(1)
			}
		},
	}
	return r.Metrics
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (r *Client) GetReadyHandlers() metrics.MapCheckFunc {
	_ = r.Service.GetReadyHandlers()
	r.ReadyHandlers[strings.ToUpper(ProviderName+"_notfailed")] = func() (bool, string) {
		if _, err := r.Conn.Ping(r.ctx).Result(); err != nil {
			return false, err.Error()
		}

		return true, ""
	}
	return r.ReadyHandlers
}

func (r *Client) Shutdown() error {
	return r.Conn.Close()
}

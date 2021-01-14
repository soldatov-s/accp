package rejson

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
)

type redisProcessor struct {
	ctx     context.Context
	Process func(ctx context.Context, cmd redis.Cmder) error
}

/*
Client is an extended redis.Client, stores a pointer to the original redis.Client
*/
type Client struct {
	*redis.Client
	*redisProcessor
}

func ExtendClient(ctx context.Context, client *redis.Client) *Client {
	return &Client{
		client,
		&redisProcessor{
			ctx:     ctx,
			Process: client.Process,
		},
	}
}

func (cl *Client) JSONSetWithExpire(key, path string, object interface{}, expiration time.Duration, args ...interface{}) error {
	jsonData, err := json.Marshal(object)
	if err != nil {
		return err
	}

	if _, err = cl.redisProcessor.JSONSET(key, path, string(jsonData), args...).Result(); err != nil {
		return err
	}

	_, err = cl.Expire(cl.ctx, key, expiration).Result()

	return err
}

/*
JSONGET

Possible args:

(Optional) INDENT + indent-string
(Optional) NEWLINE + line-break-string
(Optional) SPACE + space-string
(Optional) NOESCAPE
(Optional) path ...string

returns stringCmd -> the JSON string
read more: https://oss.redislabs.com/rejson/commands/#jsonget
*/
func (cl *redisProcessor) JSONGET(key string, args ...interface{}) *redis.StringCmd {
	return jsonGetExecute(cl, append([]interface{}{key}, args...)...)
}

/*
jsonSet
Possible args:
(Optional)
*/
func (cl *redisProcessor) JSONSET(key, path, jsonData string, args ...interface{}) *redis.StatusCmd {
	return jsonSetExecute(cl, append([]interface{}{key, path, jsonData}, args...)...)
}

/*
JsonDel
returns intCmd -> deleted 1 or 0
read more: https://oss.redislabs.com/rejson/commands/#jsondel
*/
func (cl *redisProcessor) JSONDEL(key, path string) *redis.IntCmd {
	return jsonDelExecute(cl, key, path)
}

func (cl *redisProcessor) JsonNumIncrBy(key, path string, num int) *redis.StringCmd {
	return jsonNumIncrByExecute(cl, key, path, num)
}

func concatWithCmd(cmdName string, args []interface{}) []interface{} {
	res := make([]interface{}, 1)
	res[0] = cmdName
	for _, v := range args {
		if str, ok := v.(string); ok {
			if str == "" {
				continue
			}
		}
		res = append(res, v)
	}
	return res
}

func jsonGetExecute(c *redisProcessor, args ...interface{}) *redis.StringCmd {
	cmd := redis.NewStringCmd(c.ctx, concatWithCmd("JSON.GET", args)...)
	_ = c.Process(c.ctx, cmd)
	return cmd
}

func jsonSetExecute(c *redisProcessor, args ...interface{}) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(c.ctx, concatWithCmd("JSON.SET", args)...)
	_ = c.Process(c.ctx, cmd)
	return cmd
}

func jsonDelExecute(c *redisProcessor, args ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(c.ctx, concatWithCmd("JSON.DEL", args)...)
	_ = c.Process(c.ctx, cmd)
	return cmd
}

func jsonNumIncrByExecute(c *redisProcessor, args ...interface{}) *redis.StringCmd {
	cmd := redis.NewStringCmd(c.ctx, concatWithCmd("JSON.NUMINCRBY", args)...)
	_ = c.Process(c.ctx, cmd)
	return cmd
}

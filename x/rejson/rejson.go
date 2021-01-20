package rejson

import (
	"context"
	"encoding"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

type redisProcessor struct {
	Process func(ctx context.Context, cmd redis.Cmder) error
}

/*
Client is an extended redis.Client, stores a pointer to the original redis.Client
*/
type Client struct {
	*redis.Client
	*redisProcessor
}

func ExtendClient(client *redis.Client) *Client {
	return &Client{
		client,
		&redisProcessor{
			Process: client.Process,
		},
	}
}

func (cl *Client) JSONSetWithExpire(ctx context.Context,
	key, path string, object interface{}, expiration time.Duration, args ...interface{}) error {
	v, ok := object.(encoding.BinaryMarshaler)
	if !ok {
		return errors.New("typecast to BinaryMarshaler failed")
	}
	jsonData, err := v.MarshalBinary()
	if err != nil {
		return err
	}

	if _, err = cl.redisProcessor.JSONSet(ctx, key, path, string(jsonData), args...).Result(); err != nil {
		return err
	}

	_, err = cl.Expire(ctx, key, expiration).Result()

	return err
}

/*
JSONGet

Possible args:

(Optional) INDENT + indent-string
(Optional) NEWLINE + line-break-string
(Optional) SPACE + space-string
(Optional) NOESCAPE
(Optional) path ...string

returns stringCmd -> the JSON string
read more: https://oss.redislabs.com/rejson/commands/#jsonget
*/
func (cl *redisProcessor) JSONGet(ctx context.Context, key string, args ...interface{}) *redis.StringCmd {
	return jsonGetExecute(ctx, cl, append([]interface{}{key}, args...)...)
}

/*
jsonSet
Possible args:
(Optional)
*/
func (cl *redisProcessor) JSONSet(ctx context.Context, key, path, jsonData string, args ...interface{}) *redis.StatusCmd {
	return jsonSetExecute(ctx, cl, append([]interface{}{key, path, jsonData}, args...)...)
}

/*
JsonDel
returns intCmd -> deleted 1 or 0
read more: https://oss.redislabs.com/rejson/commands/#jsondel
*/
func (cl *redisProcessor) JSONDel(ctx context.Context, key, path string) *redis.IntCmd {
	return jsonDelExecute(ctx, cl, key, path)
}

func (cl *redisProcessor) JSONNUMINCRBy(ctx context.Context, key, path string, num int) *redis.StringCmd {
	return jsonNumIncrByExecute(ctx, cl, key, path, num)
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

func jsonGetExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, concatWithCmd("JSON.GET", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonSetExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, concatWithCmd("JSON.SET", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonDelExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.DEL", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonNumIncrByExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, concatWithCmd("JSON.NUMINCRBY", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

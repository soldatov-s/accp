package redis

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultLockID        = "65b4e86b4d214b63b1b39a3ca1ac1604"
	defaultCheckInterval = 100 * time.Millisecond
	defaultExpire        = 15 * time.Second
)

var (
	ErrDBConnNotEstablished = errors.New("cache connection not established")
	ErrNotLockKey           = errors.New("passed lockID is not string")
)

type IMutex interface {
	Lock() (err error)
	Unlock() (err error)
	Extend(timeout time.Duration) (err error)
}

// Mutex provides a distributed mutex across multiple instances via Redis
type Mutex struct {
	conn          *RedisClient
	lockKey       string
	lockValue     string
	checkInterval time.Duration
	expire        time.Duration
	locked        bool
	// RWMutex to lock inside an instance
	mu sync.RWMutex
}

// NewMutex creates new distributed redis mutex
func (r *RedisClient) NewMutex(conn *RedisClient, expire, checkInterval time.Duration) IMutex {
	return r.NewMutexByID(defaultLockID, expire, checkInterval)
}

// NewMutexByID creates new distributed redis mutex by ID
func (r *RedisClient) NewMutexByID(id string, expire, checkInterval time.Duration) IMutex {
	checkIntervalValue := checkInterval
	if checkIntervalValue == 0 {
		checkIntervalValue = defaultCheckInterval
	}

	expireValue := expire
	if expireValue == 0 {
		expireValue = defaultExpire
	}

	return &Mutex{
		conn:          r,
		lockKey:       id,
		checkInterval: checkIntervalValue,
		expire:        expireValue,
		locked:        false,
	}
}

// Lock sets Redis-lock item. It is blocking call which will wait until
// redis lock key will be deleted, pretty much like simple mutex.
func (redisLock *Mutex) Lock() (err error) {
	redisLock.mu.Lock()
	if redisLock.conn == nil {
		return nil
	}
	return redisLock.commonLock()
}

// Unlock deletes Redis-lock item.
func (redisLock *Mutex) Unlock() (err error) {
	redisLock.mu.Unlock()
	if redisLock.conn == nil {
		return nil
	}
	return redisLock.commonUnlock()
}

// Extend attempts to extend the timeout of a Redis-lock.
func (redisLock *Mutex) Extend(timeout time.Duration) (err error) {
	if redisLock.conn == nil {
		return nil
	}
	return redisLock.commonExtend(timeout)
}

// checkDBConn check that connection not nil and active
func (redisLock *Mutex) checkDBConn() (conn *RedisClient, err error) {
	conn = redisLock.conn

	if conn == nil {
		return nil, ErrDBConnNotEstablished
	}

	_, err = conn.Ping(conn.ctx).Result()
	if err != nil {
		return nil, ErrDBConnNotEstablished
	}

	return conn, nil
}

func (redisLock *Mutex) commonLock() (err error) {
	var result bool

	conn, err := redisLock.checkDBConn()
	if err != nil {
		return err
	}

	newUUID, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	redisLock.lockValue = newUUID.String()

	result, err = conn.SetNX(conn.ctx, redisLock.lockKey, redisLock.lockValue, redisLock.expire).Result()
	if err != nil {
		return err
	}

	if !result {
		redisLock.locked = true

		for {
			conn, err := redisLock.checkDBConn()
			if err != nil {
				return err
			}

			result, err = conn.SetNX(conn.ctx, redisLock.lockKey, redisLock.lockValue, redisLock.expire).Result()
			if err != nil {
				return err
			}

			if result || !redisLock.locked {
				return nil
			}

			time.Sleep(redisLock.checkInterval)
		}
	}

	return nil
}

func (redisLock *Mutex) commonUnlock() (err error) {
	if redisLock.locked {
		conn, err := redisLock.checkDBConn()
		if err != nil {
			return err
		}

		cmdString, err := conn.Get(conn.ctx, redisLock.lockKey).Result()
		if err != nil {
			return err
		}

		if redisLock.lockValue == cmdString {
			_, err = conn.Del(conn.ctx, redisLock.lockKey).Result()
			if err != nil {
				return err
			}

			redisLock.locked = false
		}
	}

	return nil
}

func (redisLock *Mutex) commonExtend(timeout time.Duration) (err error) {
	if redisLock.locked {
		conn, err := redisLock.checkDBConn()
		if err != nil {
			return err
		}

		cmdString, err := conn.Get(conn.ctx, redisLock.lockKey).Result()
		if err != nil {
			return err
		}

		if redisLock.lockValue == cmdString {
			err = conn.Expire(redisLock.lockKey, timeout)
			if err != nil {
				return err
			}

			redisLock.locked = false
		}
	}

	return nil
}

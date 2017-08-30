package rediLock

import (
	"net/url"
	"time"

	"github.com/garyburd/redigo/redis"
)

func newRedisPool(addr string) *redis.Pool {
	u, err := url.Parse(addr)
	if err != nil {
		panic(err)
	}
	pool := redis.Pool{
		MaxIdle:     2048,
		IdleTimeout: 60 * time.Second,
		MaxActive:   2048,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			if u.Scheme != "redis" {
				return redis.Dial("tcp", addr)
			}

			c, err := redis.Dial("tcp", u.Host)
			if err != nil {
				return nil, err
			}
			if u.User != nil {
				if _, err := c.Do("AUTH", u.User); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return &pool
}

package rediLock

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	//log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
)

type RediSync struct {
	pool *redis.Pool
}

type Mutex struct {
	name, value   string
	expire        time.Duration
	retryInterval time.Duration
	pool          *redis.Pool
}

type Option interface {
	Apply(*Mutex)
}

// OptionFunc is a function that configures a mutex.
type OptionFunc func(*Mutex)

// Apply calls f(mutex)
func (f OptionFunc) Apply(mutex *Mutex) {
	f(mutex)
}

// Set expire of a mutex.
func SetExpire(expire time.Duration) Option {
	return OptionFunc(func(m *Mutex) {
		m.expire = expire
	})
}

// Set retry interval of locking a mutex.
func SetRetryInterval(interval time.Duration) Option {
	return OptionFunc(func(m *Mutex) {
		m.retryInterval = interval
	})
}

// Create new redis sync
func NewRediSync(pool *redis.Pool) *RediSync {
	return &RediSync{
		pool: pool,
	}
}

// Create a new mutex with: name, setting options.
func (s *RediSync) NewMutex(name string, options ...Option) *Mutex {
	value, err := genValue()
	if err != nil {
		//log.Infof("gen value for mutex fail: %v", err)
		panic(err)
	}

	mutex := &Mutex{
		name:          name,
		value:         value,
		expire:        time.Second * 5,
		retryInterval: time.Millisecond * 100,
		pool:          s.pool,
	}

	for _, opt := range options {
		opt.Apply(mutex)
	}
	return mutex
}

// Safe delete of mutex key.
var deleteScript = redis.NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
`)

// Generate unique value for a mutex.
func genValue() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// Trying to lock using SETNX, sleep and retry if failed.
func (m *Mutex) Lock() {
	c := m.pool.Get()
	defer c.Close()

	for {
		_, err := redis.String(c.Do("SET", m.name, m.value, "PX", int(m.expire/time.Millisecond), "NX"))
		if err == nil {
			break
		}
		time.Sleep(m.retryInterval)
	}
}

// Trying to unlock using delete script.
func (m *Mutex) UnLock() {
	c := m.pool.Get()
	defer c.Close()
	for {
		_, err := deleteScript.Do(c, m.name, m.value)
		if err == nil {
			break
		} else {
			//log.Infof("try to unlock fail: %v", err)
		}
	}
}

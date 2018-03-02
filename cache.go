package distcache

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
)

type entry struct {
	key, value string
	expires    time.Time
}

type cache struct {
	mu  sync.RWMutex
	lru *lru.Cache

	nBytes int64
}

func newCache() *cache {
	c := &cache{}
	c.lru = &lru.Cache{
		OnEvicted: func(key lru.Key, value interface{}) {
			val := value.(string)
			c.nBytes -= int64(len(key.(string))) + int64(len(val))
		},
	}
	return c
}

func (c *cache) Get(key string) (val string, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.lru.Get(key)
	if !ok {
		return
	}
	v, ok := value.(*entry)
	if !ok {
		panic(fmt.Errorf("expected type entry, received: %s", reflect.TypeOf(value)))
	}
	if !v.expires.IsZero() && time.Now().UTC().After(v.expires) {
		return
	}
	return v.value, ok
}

func (c *cache) Set(key, value string) {
	c.mu.Lock()
	c.lru.Add(key, &entry{key: key, value: value})
	c.nBytes += int64(len(key)) + int64(len(value))
	c.mu.Unlock()
}

func (c *cache) SetTTL(key, value string, ttl time.Duration) {
	c.mu.Lock()
	c.lru.Add(key, &entry{
		key:     key,
		value:   value,
		expires: time.Now().UTC().Add(ttl),
	})
	c.nBytes += int64(len(key)) + int64(len(value))
	c.mu.Unlock()
}

func (c *cache) Delete(key string) {
	c.lru.Remove(key)
}

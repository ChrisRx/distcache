package distcache

import (
	"container/list"
	"sync"
	"time"
)

type entry struct {
	key, value string
	expires    int64
}

func (e *entry) Len() int {
	return len(e.key) + len(e.value) + 8
}

type cache struct {
	mu    sync.RWMutex
	list  *list.List
	items map[string]*list.Element

	nBytes int64

	MaxCacheBytes int64
}

func newCache() *cache {
	c := &cache{
		list:          list.New(),
		items:         make(map[string]*list.Element),
		MaxCacheBytes: 1 << 28,
	}
	return c
}

func (c *cache) Get(key string) (val string, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if item, hit := c.items[key]; hit {
		c.list.MoveToFront(item)
		kv := item.Value.(*entry)
		if kv.expires != 0 && kv.expires > time.Now().Unix() {
			return
		}
		return kv.value, true
	}
	return
}

func (c *cache) Set(key, value string) {
	c.mu.Lock()
	c.setEntry(&entry{key: key, value: value})
	c.mu.Unlock()
}

func (c *cache) SetTTL(key, value string, ttl int64) {
	c.mu.Lock()
	now := time.Now().Unix()
	c.setEntry(&entry{key, value, now + ttl})
	c.mu.Unlock()
}

func (c *cache) setEntry(e *entry) {
	if item, ok := c.items[e.key]; ok {
		c.list.MoveToFront(item)
		kv := item.Value.(*entry)
		c.nBytes += int64(e.Len() - kv.Len())
		kv.value = e.value
		kv.expires = e.expires
	}
	c.items[e.key] = c.list.PushFront(e)
	c.nBytes += int64(e.Len())
	for {
		if c.nBytes <= c.MaxCacheBytes {
			return
		}
		if item := c.list.Back(); item != nil {
			c.removeElement(item)
		}
	}

}

func (c *cache) Delete(key string) {
	if item, ok := c.items[key]; ok {
		c.removeElement(item)
	}
}

func (c *cache) removeElement(e *list.Element) {
	c.list.Remove(e)
	kv := e.Value.(*entry)
	delete(c.items, kv.key)
	c.nBytes -= int64(kv.Len())
}

func (c *cache) Keys() []string {
	keys := make([]string, 0)
	for k, v := range c.items {
		kv := v.Value.(*entry)
		if kv.expires != 0 && kv.expires > time.Now().Unix() {
			continue
		}
		keys = append(keys, k)
	}
	return keys
}

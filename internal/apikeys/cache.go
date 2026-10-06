package apikeys

import (
	"sync"
	"time"
)

type Entry struct {
	KeyType  string
	UserID   string
	Revoked  bool
	LoadedAt time.Time
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]Entry
	ttl   time.Duration
}

func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		items: map[string]Entry{},
		ttl:   ttl,
	}
}

func (c *Cache) Get(key string) (Entry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[key]
	if !ok || time.Since(e.LoadedAt) > c.ttl {
		return Entry{}, false
	}
	return e, true
}

func (c *Cache) Set(key, keyType, userID string, revoked bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = Entry{
		KeyType:  keyType,
		UserID:   userID,
		Revoked:  revoked,
		LoadedAt: time.Now(),
	}
}

func (c *Cache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *Cache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, e := range c.items {
		if now.Sub(e.LoadedAt) > c.ttl {
			delete(c.items, k)
		}
	}
}

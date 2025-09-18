package cache

import (
	"math/rand"
	"sync"
	"time"
)

func NewCache[T any](settings ...time.Duration) *KeyCache[T] {
	expires := 24 * time.Hour // default
	cleanup := 1 * time.Hour  // default
	if len(settings) > 0 {
		expires = settings[0]
	}
	if len(settings) > 1 {
		cleanup = settings[1]
	}
	newCache := KeyCache[T]{
		data:         make(map[string]cachedValue[T]),
		expiresAfter: expires, // default
	}
	// Adding jitter for cleanup to prevent all caches from running cleanup at the same time
	// Generate a random duration between 1 and 2 seconds
	min := 1 * time.Second
	max := 2 * time.Second
	randomDuration := min + time.Duration(rand.Int63n(int64(max-min)))
	go newCache.cleanupExpiredJob(cleanup + randomDuration)
	return &newCache
}

type KeyCache[T any] struct {
	data         map[string]cachedValue[T]
	mu           sync.RWMutex
	expiresAfter time.Duration
}

type cachedValue[T any] struct {
	value     T
	expiresAt time.Time
}

func (c *KeyCache[T]) Set(key string, value T) {
	c.SetWithExp(key, value, c.expiresAfter)
}

func (c *KeyCache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *KeyCache[T]) SetWithExp(key string, value T, exp time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cachedValue[T]{
		value:     value,
		expiresAt: time.Now().Add(exp),
	}
}

func (c *KeyCache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cached, ok := c.data[key]
	if ok && time.Now().After(cached.expiresAt) {
		ok = false
	}
	return cached.value, ok
}

func (c *KeyCache[T]) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for key, cached := range c.data {
		if now.After(cached.expiresAt) {
			delete(c.data, key)
		}
	}
}

// should automatically run for all cache types as part of init.
func (c *KeyCache[T]) cleanupExpiredJob(frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for range ticker.C {
		c.cleanupExpired()
	}
}

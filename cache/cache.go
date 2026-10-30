package cache

import (
	"math/rand"
	"sync"
	"time"
)

// NewCache returns a new typed TTL cache. Caches are intended to be long-lived
// for the lifetime of the program; use Close when discarding a cache (for example
// in tests) to stop the background cleanup goroutine.
//
// Optional settings: default entry TTL (settings[0], default 24h), background
// cleanup interval (settings[1], default 1h). A non-positive cleanup interval
// falls back to the default.
func NewCache[T any](settings ...time.Duration) *KeyCache[T] {
	expires := 24 * time.Hour // default
	cleanup := 1 * time.Hour  // default
	if len(settings) > 0 {
		expires = settings[0]
	}
	if len(settings) > 1 {
		cleanup = settings[1]
		if cleanup <= 0 {
			cleanup = 1 * time.Hour
		}
	}
	newCache := KeyCache[T]{
		data:         make(map[string]cachedValue[T]),
		expiresAfter: expires,
		stopCleanup:  make(chan struct{}),
	}
	// Adding jitter for cleanup to prevent all caches from running cleanup at the same time
	// Generate a random duration between 1 and 2 seconds
	min := 1 * time.Second
	max := 2 * time.Second
	randomDuration := min + time.Duration(rand.Int63n(int64(max-min)))
	go newCache.cleanupExpiredJob(cleanup + randomDuration)
	return &newCache
}

// KeyCache is a string-keyed, generically typed TTL cache safe for concurrent use.
// Instances are meant to be kept for the process lifetime; call Close when the
// cache is no longer needed.
type KeyCache[T any] struct {
	data         map[string]cachedValue[T]
	mu           sync.RWMutex
	expiresAfter time.Duration
	stopCleanup  chan struct{}
	closeOnce    sync.Once
}

type cachedValue[T any] struct {
	value     T
	expiresAt time.Time
}

// Set stores value under key using this cache's default TTL.
func (c *KeyCache[T]) Set(key string, value T) {
	c.SetWithExp(key, value, c.expiresAfter)
}

// Delete removes key from the cache.
func (c *KeyCache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// ClearAll removes all entries from the cache.
func (c *KeyCache[T]) ClearAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]cachedValue[T])
}

// SetWithExp stores value under key with a custom TTL.
func (c *KeyCache[T]) SetWithExp(key string, value T, exp time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cachedValue[T]{
		value:     value,
		expiresAt: time.Now().Add(exp),
	}
}

// Get returns the value for key and whether it is present and unexpired.
// Missing or expired keys return the zero value and false.
func (c *KeyCache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cached, ok := c.data[key]
	if !ok {
		var zero T
		return zero, false
	}
	if time.Now().After(cached.expiresAt) {
		var zero T
		return zero, false
	}
	return cached.value, true
}

// Close stops the background cleanup goroutine and ticker. It is idempotent and
// safe to call from multiple goroutines. After Close, Get, Set, Delete, and
// ClearAll remain usable, but expired entries are no longer removed automatically
// (call Delete, ClearAll, or discard the cache).
func (c *KeyCache[T]) Close() {
	c.closeOnce.Do(func() {
		close(c.stopCleanup)
	})
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

func (c *KeyCache[T]) cleanupExpiredJob(frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCleanup:
			return
		case <-ticker.C:
			c.cleanupExpired()
		}
	}
}

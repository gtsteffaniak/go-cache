# go-cache
simple generic cache module in go

import and use:

```go
import "github.com/gtsteffaniak/go-cache/cache"

// cache.NewCache[T](ExpirationTime, cleanupInterval)
var defaultCache = cache.NewCache[string]()
var cacheWithCustomExp = cache.NewCache[string](1*time.Minute)
var cacheWithCustomExpAndCleanup = cache.NewCache[string](60*time.Minute, 48*time.Hour)

func main() {
	// Set a value
	defaultCache.Set("someUniqueKey", "hello world")

	// Get a value - no type assertion needed with generics!
	value, ok := defaultCache.Get("someUniqueKey")
	if !ok {
		return "", fmt.Errorf("key not found")
	}
	return value, nil
}
```

## Features

- **Type-safe**: Uses Go generics for compile-time type safety
- **Automatic cleanup**: Expired entries are automatically removed
- **Configurable expiration**: Set custom expiration times per cache instance
- **Thread-safe**: Safe for concurrent use with read-write mutex
- **Jittered cleanup**: Prevents all caches from running cleanup simultaneously

## API

```go
// Create a new cache with type T
func NewCache[T any](settings ...time.Duration) *KeyCache[T]

// Set a value with default expiration
func (c *KeyCache[T]) Set(key string, value T)

// Set a value with custom expiration
func (c *KeyCache[T]) SetWithExp(key string, value T, exp time.Duration)

// Get a value (returns value, found)
func (c *KeyCache[T]) Get(key string) (T, bool)

// Delete a value
func (c *KeyCache[T]) Delete(key string)
```

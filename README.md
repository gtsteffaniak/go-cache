# go-cache
simple cache module in go

import and use:

```
import "github.com/gtsteffaniak/go-logger/logger"

// cache.NewCache(ExpirationTime,cleanupInterval)
var defaultCache = cache.NewCache()
var cacheWithCustomExp = cache.NewCache(1*time.Minute)
var cacheWithCustomExpAndCleanup = cache.NewCache(60*time.Minute, 48*time.Hour)

func main() {
    // check for stored string value
    value, ok := cache.defaultCache.Get("someUniqueKey").(string)
	if !ok {
		return value, fmt.Errorf("key not found")
	}
	return value, nil
}
```
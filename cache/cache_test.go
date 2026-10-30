package cache

import (
	"sync"
	"testing"
	"time"
)

func entryCount[T any](c *KeyCache[T]) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

func TestGetMissingReturnsZeroValue(t *testing.T) {
	c := NewCache[int]()
	defer c.Close()

	got, ok := c.Get("missing")
	if ok || got != 0 {
		t.Fatalf("Get(missing) = %d, %v; want 0, false", got, ok)
	}
}

func TestGetHitAndMiss(t *testing.T) {
	c := NewCache[string]()
	defer c.Close()

	c.Set("k", "v")
	got, ok := c.Get("k")
	if !ok || got != "v" {
		t.Fatalf("Get() = %q, %v; want v, true", got, ok)
	}

	_, ok = c.Get("missing")
	if ok {
		t.Fatal("Get(missing) ok = true; want false")
	}
}

func TestGetExpiredReturnsZeroValue(t *testing.T) {
	c := NewCache[string]()
	defer c.Close()

	c.SetWithExp("k", "secret", -time.Millisecond)
	got, ok := c.Get("k")
	if ok {
		t.Fatalf("Get() ok = true for expired key")
	}
	if got != "" {
		t.Fatalf("Get() = %q; want zero value when expired", got)
	}
}

func TestGetExpiryBeforeAndAfter(t *testing.T) {
	c := NewCache[string]()
	defer c.Close()

	c.SetWithExp("future", "v", time.Hour)
	got, ok := c.Get("future")
	if !ok || got != "v" {
		t.Fatalf("Get(future) = %q, %v", got, ok)
	}

	c.SetWithExp("past", "old", -time.Nanosecond)
	_, ok = c.Get("past")
	if ok {
		t.Fatal("Get(past) ok = true; want false")
	}
}

func TestCleanupExpiredRemovesEntries(t *testing.T) {
	c := NewCache[string]()
	defer c.Close()

	c.SetWithExp("a", "1", -time.Millisecond)
	c.SetWithExp("b", "2", time.Hour)
	c.cleanupExpired()

	if entryCount(c) != 1 {
		t.Fatalf("entry count = %d; want 1", entryCount(c))
	}
	_, ok := c.Get("b")
	if !ok {
		t.Fatal("Get(b) ok = false; want true")
	}
}

func TestNewCacheNonPositiveCleanupInterval(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewCache with zero cleanup panicked: %v", r)
		}
	}()
	c := NewCache[string](time.Hour, 0)
	c.Close()
}

func TestCloseIdempotent(t *testing.T) {
	c := NewCache[int]()
	c.Close()
	c.Close()
}

func TestCloseConcurrent(t *testing.T) {
	c := NewCache[int]()
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Close()
		}()
	}
	wg.Wait()
}

func TestCloseStopsBackgroundCleanup(t *testing.T) {
	c := NewCache[string](time.Hour, 10*time.Millisecond)
	c.SetWithExp("expired", "x", -time.Millisecond)

	deadline := time.Now().Add(3 * time.Second)
	for entryCount(c) > 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if entryCount(c) != 0 {
		t.Fatalf("cleanup did not run before Close; entries = %d", entryCount(c))
	}

	c.Close()

	c.SetWithExp("expired2", "y", -time.Millisecond)
	time.Sleep(3 * time.Second)
	if entryCount(c) != 1 {
		t.Fatalf("entries = %d after Close; want 1 (cleanup stopped)", entryCount(c))
	}
	_, ok := c.Get("expired2")
	if ok {
		t.Fatal("Get expired key ok = true; want false")
	}
}

func TestCloseAllowsContinuedUse(t *testing.T) {
	c := NewCache[string]()
	c.Close()
	c.Set("k", "v")
	got, ok := c.Get("k")
	if !ok || got != "v" {
		t.Fatalf("after Close: Get() = %q, %v", got, ok)
	}
	c.Delete("k")
	c.ClearAll()
}

func TestConcurrentGetSetDelete(t *testing.T) {
	c := NewCache[int]()
	defer c.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := string(rune('a' + (n % 26)))
			c.Set(key, n)
			c.Get(key)
			if n%3 == 0 {
				c.Delete(key)
			}
		}(i)
	}
	wg.Wait()
}

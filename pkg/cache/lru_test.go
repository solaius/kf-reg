package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestLRUCache(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"SetAndGet", testSetAndGet},
		{"GetMiss", testGetMiss},
		{"GetExpired", testGetExpired},
		{"SetOverMaxSizeEvictsOldest", testSetOverMaxSizeEvictsOldest},
		{"InvalidateRemovesEntry", testInvalidateRemovesEntry},
		{"InvalidateAllClearsCache", testInvalidateAllClearsCache},
		{"SetUpdatesExisting", testSetUpdatesExisting},
		{"ConcurrentAccess", testConcurrentAccess},
		{"SizeReflectsEntryCount", testSizeReflectsEntryCount},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func testSetAndGet(t *testing.T) {
	c := NewLRUCache(10, 5*time.Second)
	c.Set("key1", []byte("value1"))

	got, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}
	if string(got) != "value1" {
		t.Fatalf("expected %q, got %q", "value1", string(got))
	}
}

func testGetMiss(t *testing.T) {
	c := NewLRUCache(10, 5*time.Second)

	got, ok := c.Get("nonexistent")
	if ok {
		t.Fatal("expected cache miss, got hit")
	}
	if got != nil {
		t.Fatalf("expected nil value on miss, got %q", string(got))
	}
}

func testGetExpired(t *testing.T) {
	c := NewLRUCache(10, 50*time.Millisecond)
	c.Set("key1", []byte("value1"))

	// Verify it's there initially.
	if _, ok := c.Get("key1"); !ok {
		t.Fatal("expected cache hit before expiry")
	}

	// Wait for TTL to expire.
	time.Sleep(100 * time.Millisecond)

	got, ok := c.Get("key1")
	if ok {
		t.Fatal("expected cache miss after expiry, got hit")
	}
	if got != nil {
		t.Fatalf("expected nil value after expiry, got %q", string(got))
	}

	// Expired entry should be lazily removed.
	if c.Size() != 0 {
		t.Fatalf("expected size 0 after expired get, got %d", c.Size())
	}
}

func testSetOverMaxSizeEvictsOldest(t *testing.T) {
	c := NewLRUCache(3, 5*time.Second)

	c.Set("a", []byte("1"))
	time.Sleep(time.Millisecond) // Ensure distinct timestamps.
	c.Set("b", []byte("2"))
	time.Sleep(time.Millisecond)
	c.Set("c", []byte("3"))

	if c.Size() != 3 {
		t.Fatalf("expected size 3, got %d", c.Size())
	}

	// Adding a 4th entry should evict "a" (oldest).
	c.Set("d", []byte("4"))

	if c.Size() != 3 {
		t.Fatalf("expected size 3 after eviction, got %d", c.Size())
	}

	if _, ok := c.Get("a"); ok {
		t.Fatal("expected 'a' to be evicted")
	}

	// "b", "c", "d" should still be present.
	for _, key := range []string{"b", "c", "d"} {
		if _, ok := c.Get(key); !ok {
			t.Fatalf("expected %q to still be in cache", key)
		}
	}
}

func testInvalidateRemovesEntry(t *testing.T) {
	c := NewLRUCache(10, 5*time.Second)
	c.Set("key1", []byte("value1"))
	c.Set("key2", []byte("value2"))

	c.Invalidate("key1")

	if _, ok := c.Get("key1"); ok {
		t.Fatal("expected 'key1' to be invalidated")
	}

	// key2 should still be present.
	if _, ok := c.Get("key2"); !ok {
		t.Fatal("expected 'key2' to still be in cache")
	}
}

func testInvalidateAllClearsCache(t *testing.T) {
	c := NewLRUCache(10, 5*time.Second)
	c.Set("key1", []byte("value1"))
	c.Set("key2", []byte("value2"))
	c.Set("key3", []byte("value3"))

	c.InvalidateAll()

	if c.Size() != 0 {
		t.Fatalf("expected size 0 after InvalidateAll, got %d", c.Size())
	}

	for _, key := range []string{"key1", "key2", "key3"} {
		if _, ok := c.Get(key); ok {
			t.Fatalf("expected %q to be cleared", key)
		}
	}
}

func testSetUpdatesExisting(t *testing.T) {
	c := NewLRUCache(10, 5*time.Second)
	c.Set("key1", []byte("old"))
	c.Set("key1", []byte("new"))

	got, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != "new" {
		t.Fatalf("expected %q, got %q", "new", string(got))
	}

	// Size should not increase on update.
	if c.Size() != 1 {
		t.Fatalf("expected size 1 after update, got %d", c.Size())
	}
}

func testConcurrentAccess(t *testing.T) {
	c := NewLRUCache(100, 5*time.Second)

	var wg sync.WaitGroup
	goroutines := 50
	ops := 100

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < ops; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				c.Set(key, []byte(fmt.Sprintf("value-%d-%d", id, j)))
				c.Get(key)
				if j%10 == 0 {
					c.Invalidate(key)
				}
			}
		}(i)
	}

	wg.Wait()

	// No panics or data races means success. Size should be <= maxSize.
	if c.Size() > 100 {
		t.Fatalf("expected size <= 100, got %d", c.Size())
	}
}

func testSizeReflectsEntryCount(t *testing.T) {
	c := NewLRUCache(10, 5*time.Second)

	if c.Size() != 0 {
		t.Fatalf("expected initial size 0, got %d", c.Size())
	}

	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))

	if c.Size() != 2 {
		t.Fatalf("expected size 2, got %d", c.Size())
	}

	c.Invalidate("a")
	if c.Size() != 1 {
		t.Fatalf("expected size 1 after invalidation, got %d", c.Size())
	}
}

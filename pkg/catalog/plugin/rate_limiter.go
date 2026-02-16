package plugin

import (
	"sync"
	"time"
)

// RefreshRateLimiter enforces per-source rate limiting for refresh operations
// using a token bucket algorithm. Each (plugin, sourceID) pair gets an
// independent bucket.
type RefreshRateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	interval time.Duration // minimum time between refreshes
}

// bucket tracks the last allowed refresh time for a single key.
type bucket struct {
	lastAllowed time.Time
}

// NewRefreshRateLimiter creates a rate limiter that allows one refresh per
// source per interval. The default interval is 30 seconds.
func NewRefreshRateLimiter(interval time.Duration) *RefreshRateLimiter {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &RefreshRateLimiter{
		buckets:  make(map[string]*bucket),
		interval: interval,
	}
}

// Allow checks whether a refresh for the given key is permitted.
// Returns true if allowed, or false with the duration until the next allowed
// attempt. The key is typically "pluginName:sourceID" or "pluginName:*" for
// refresh-all.
func (rl *RefreshRateLimiter) Allow(key string) (bool, time.Duration) {
	return rl.allowAt(key, time.Now())
}

// allowAt is the testable core of Allow that accepts a "now" parameter.
func (rl *RefreshRateLimiter) allowAt(key string, now time.Time) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &bucket{lastAllowed: now}
		return true, 0
	}

	nextAllowed := b.lastAllowed.Add(rl.interval)
	if now.Before(nextAllowed) {
		return false, nextAllowed.Sub(now)
	}

	b.lastAllowed = now
	return true, 0
}

// Reset removes all tracked buckets. Useful for testing.
func (rl *RefreshRateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.buckets = make(map[string]*bucket)
}

// RefreshKey builds a rate limiter key for a specific source.
func RefreshKey(pluginName, sourceID string) string {
	return pluginName + ":" + sourceID
}

// RefreshAllKey builds a rate limiter key for a refresh-all operation.
func RefreshAllKey(pluginName string) string {
	return pluginName + ":*"
}

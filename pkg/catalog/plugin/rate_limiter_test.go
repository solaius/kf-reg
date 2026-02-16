package plugin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRefreshRateLimiter_FirstCallAllowed(t *testing.T) {
	rl := NewRefreshRateLimiter(30 * time.Second)
	allowed, retryAfter := rl.Allow("models:src1")
	assert.True(t, allowed)
	assert.Zero(t, retryAfter)
}

func TestRefreshRateLimiter_SecondCallBlocked(t *testing.T) {
	rl := NewRefreshRateLimiter(30 * time.Second)

	now := time.Now()
	allowed, _ := rl.allowAt("models:src1", now)
	assert.True(t, allowed)

	// Immediately retry should be blocked.
	allowed, retryAfter := rl.allowAt("models:src1", now.Add(1*time.Second))
	assert.False(t, allowed)
	assert.InDelta(t, 29*time.Second, retryAfter, float64(1*time.Second))
}

func TestRefreshRateLimiter_AllowedAfterInterval(t *testing.T) {
	rl := NewRefreshRateLimiter(30 * time.Second)

	now := time.Now()
	allowed, _ := rl.allowAt("models:src1", now)
	assert.True(t, allowed)

	// After the interval, should be allowed again.
	allowed, retryAfter := rl.allowAt("models:src1", now.Add(31*time.Second))
	assert.True(t, allowed)
	assert.Zero(t, retryAfter)
}

func TestRefreshRateLimiter_IndependentKeys(t *testing.T) {
	rl := NewRefreshRateLimiter(30 * time.Second)

	// First source.
	allowed, _ := rl.Allow("models:src1")
	assert.True(t, allowed)

	// Different source should not be affected.
	allowed, _ = rl.Allow("models:src2")
	assert.True(t, allowed)

	// Different plugin should not be affected.
	allowed, _ = rl.Allow("datasets:src1")
	assert.True(t, allowed)
}

func TestRefreshRateLimiter_RefreshAllIndependent(t *testing.T) {
	rl := NewRefreshRateLimiter(30 * time.Second)

	// Source-specific refresh.
	allowed, _ := rl.Allow(RefreshKey("models", "src1"))
	assert.True(t, allowed)

	// Refresh-all for the same plugin should be independent.
	allowed, _ = rl.Allow(RefreshAllKey("models"))
	assert.True(t, allowed)
}

func TestRefreshRateLimiter_Reset(t *testing.T) {
	rl := NewRefreshRateLimiter(30 * time.Second)

	allowed, _ := rl.Allow("models:src1")
	assert.True(t, allowed)

	// Blocked immediately.
	allowed, _ = rl.Allow("models:src1")
	assert.False(t, allowed)

	// After reset, should be allowed again.
	rl.Reset()
	allowed, _ = rl.Allow("models:src1")
	assert.True(t, allowed)
}

func TestRefreshRateLimiter_DefaultInterval(t *testing.T) {
	// Zero interval should default to 30s.
	rl := NewRefreshRateLimiter(0)

	now := time.Now()
	allowed, _ := rl.allowAt("k", now)
	assert.True(t, allowed)

	allowed, retryAfter := rl.allowAt("k", now.Add(29*time.Second))
	assert.False(t, allowed)
	assert.Greater(t, retryAfter, time.Duration(0))

	allowed, _ = rl.allowAt("k", now.Add(31*time.Second))
	assert.True(t, allowed)
}

func TestRefreshKey(t *testing.T) {
	assert.Equal(t, "models:src1", RefreshKey("models", "src1"))
}

func TestRefreshAllKey(t *testing.T) {
	assert.Equal(t, "models:*", RefreshAllKey("models"))
}

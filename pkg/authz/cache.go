package authz

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// DefaultCacheTTL is the default time-to-live for cached authorization results.
const DefaultCacheTTL = 10 * time.Second

// cacheEntry stores a cached authorization result with its expiration time.
type cacheEntry struct {
	allowed   bool
	expiresAt time.Time
}

// CachedAuthorizer wraps another Authorizer with a short-lived in-memory cache
// to reduce the number of SAR calls to the Kubernetes API server.
type CachedAuthorizer struct {
	inner Authorizer
	ttl   time.Duration
	mu    sync.RWMutex
	cache map[string]cacheEntry
}

// NewCachedAuthorizer creates a CachedAuthorizer that wraps inner with the given TTL.
func NewCachedAuthorizer(inner Authorizer, ttl time.Duration) *CachedAuthorizer {
	return &CachedAuthorizer{
		inner: inner,
		ttl:   ttl,
		cache: make(map[string]cacheEntry),
	}
}

// Authorize checks the cache first and delegates to the inner Authorizer on miss.
func (c *CachedAuthorizer) Authorize(ctx context.Context, req AuthzRequest) (bool, error) {
	key := cacheKey(req)

	// Check cache (read lock).
	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()

	if ok && time.Now().Before(entry.expiresAt) {
		return entry.allowed, nil
	}

	// Cache miss or expired — call inner authorizer.
	allowed, err := c.inner.Authorize(ctx, req)
	if err != nil {
		return false, err
	}

	// Store result.
	c.mu.Lock()
	c.cache[key] = cacheEntry{
		allowed:   allowed,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return allowed, nil
}

// cacheKey builds a deterministic cache key from an AuthzRequest.
func cacheKey(req AuthzRequest) string {
	return fmt.Sprintf("%s:%s:%s:%s:%s",
		req.User,
		strings.Join(req.Groups, ","),
		req.Resource,
		req.Verb,
		req.Namespace,
	)
}

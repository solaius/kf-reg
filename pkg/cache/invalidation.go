package cache

import (
	"fmt"
	"net/http"
)

// CacheManager holds separate cache instances for discovery and capabilities
// endpoints, each with its own TTL. It provides targeted invalidation methods
// so that config changes only clear the affected caches.
type CacheManager struct {
	discovery    *LRUCache
	capabilities *LRUCache
}

// NewCacheManager creates a CacheManager from the given configuration.
// If cfg is nil or disabled, it returns nil.
func NewCacheManager(cfg *CacheConfig) *CacheManager {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	return &CacheManager{
		discovery:    NewLRUCache(cfg.MaxSize, cfg.DiscoveryTTL),
		capabilities: NewLRUCache(cfg.MaxSize, cfg.CapabilitiesTTL),
	}
}

// InvalidatePlugin invalidates the capabilities cache entry for a specific
// plugin. It also invalidates the discovery cache since plugin info may have
// changed (e.g., source count, health status).
func (cm *CacheManager) InvalidatePlugin(pluginName string) {
	if cm == nil {
		return
	}
	// Invalidate all capability cache entries that match this plugin.
	// The key format is /api/plugins/{pluginName}/capabilities.
	key := fmt.Sprintf("/api/plugins/%s/capabilities", pluginName)
	cm.capabilities.Invalidate(key)
	// Also clear discovery since plugin metadata may change.
	cm.discovery.InvalidateAll()
}

// InvalidateAll clears both the discovery and capabilities caches entirely.
func (cm *CacheManager) InvalidateAll() {
	if cm == nil {
		return
	}
	cm.discovery.InvalidateAll()
	cm.capabilities.InvalidateAll()
}

// DiscoveryMiddleware returns HTTP middleware that caches responses for the
// /api/plugins endpoint using the discovery cache.
func (cm *CacheManager) DiscoveryMiddleware() func(http.Handler) http.Handler {
	return CacheMiddleware(cm.discovery)
}

// CapabilitiesMiddleware returns HTTP middleware that caches responses for
// /api/plugins/{name}/capabilities endpoints using the capabilities cache.
func (cm *CacheManager) CapabilitiesMiddleware() func(http.Handler) http.Handler {
	return CacheMiddleware(cm.capabilities)
}

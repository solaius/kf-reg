package cache

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// CacheConfig holds configuration for the caching layer.
type CacheConfig struct {
	// Enabled controls whether caching is active. When false, no middleware
	// is applied and all requests pass through uncached.
	Enabled bool

	// DiscoveryTTL is the TTL for the /api/plugins endpoint cache.
	DiscoveryTTL time.Duration

	// CapabilitiesTTL is the TTL for /api/plugins/{name}/capabilities caches.
	CapabilitiesTTL time.Duration

	// MaxSize is the maximum number of entries per cache instance.
	MaxSize int
}

// DefaultCacheConfig returns a CacheConfig with sensible defaults.
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		Enabled:         true,
		DiscoveryTTL:    60 * time.Second,
		CapabilitiesTTL: 30 * time.Second,
		MaxSize:         1000,
	}
}

// CacheConfigFromEnv reads cache configuration from environment variables,
// falling back to defaults for any unset variable.
//
// Environment variables:
//   - CATALOG_CACHE_ENABLED: "true" or "false" (default: "true")
//   - CATALOG_CACHE_DISCOVERY_TTL: duration in seconds (default: 60)
//   - CATALOG_CACHE_CAPABILITIES_TTL: duration in seconds (default: 30)
//   - CATALOG_CACHE_MAX_SIZE: max entries per cache (default: 1000)
func CacheConfigFromEnv() *CacheConfig {
	cfg := DefaultCacheConfig()

	if v := os.Getenv("CATALOG_CACHE_ENABLED"); v != "" {
		cfg.Enabled = strings.EqualFold(v, "true") || v == "1"
	}

	if v := os.Getenv("CATALOG_CACHE_DISCOVERY_TTL"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			cfg.DiscoveryTTL = time.Duration(secs) * time.Second
		}
	}

	if v := os.Getenv("CATALOG_CACHE_CAPABILITIES_TTL"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			cfg.CapabilitiesTTL = time.Duration(secs) * time.Second
		}
	}

	if v := os.Getenv("CATALOG_CACHE_MAX_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxSize = n
		}
	}

	return cfg
}

package cache

import (
	"testing"
	"time"
)

func TestCacheManager(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"NewCacheManagerDisabled", testNewCacheManagerDisabled},
		{"NewCacheManagerNilConfig", testNewCacheManagerNilConfig},
		{"InvalidatePluginClearsCapabilities", testInvalidatePluginClearsCapabilities},
		{"InvalidatePluginClearsDiscovery", testInvalidatePluginClearsDiscovery},
		{"InvalidateAllClearsBothCaches", testInvalidateAllClearsBothCaches},
		{"NilCacheManagerSafe", testNilCacheManagerSafe},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func testNewCacheManagerDisabled(t *testing.T) {
	cfg := &CacheConfig{Enabled: false}
	cm := NewCacheManager(cfg)
	if cm != nil {
		t.Fatal("expected nil CacheManager when disabled")
	}
}

func testNewCacheManagerNilConfig(t *testing.T) {
	cm := NewCacheManager(nil)
	if cm != nil {
		t.Fatal("expected nil CacheManager for nil config")
	}
}

func testInvalidatePluginClearsCapabilities(t *testing.T) {
	cfg := &CacheConfig{
		Enabled:         true,
		DiscoveryTTL:    5 * time.Second,
		CapabilitiesTTL: 5 * time.Second,
		MaxSize:         100,
	}
	cm := NewCacheManager(cfg)

	// Populate capabilities cache for two plugins.
	cm.capabilities.Set("/api/plugins/mcp/capabilities", []byte(`{"mcp": true}`))
	cm.capabilities.Set("/api/plugins/model/capabilities", []byte(`{"model": true}`))

	// Invalidate only the MCP plugin.
	cm.InvalidatePlugin("mcp")

	// MCP capabilities should be gone.
	if _, ok := cm.capabilities.Get("/api/plugins/mcp/capabilities"); ok {
		t.Fatal("expected mcp capabilities to be invalidated")
	}

	// Model capabilities should still be present.
	if _, ok := cm.capabilities.Get("/api/plugins/model/capabilities"); !ok {
		t.Fatal("expected model capabilities to still be cached")
	}
}

func testInvalidatePluginClearsDiscovery(t *testing.T) {
	cfg := &CacheConfig{
		Enabled:         true,
		DiscoveryTTL:    5 * time.Second,
		CapabilitiesTTL: 5 * time.Second,
		MaxSize:         100,
	}
	cm := NewCacheManager(cfg)

	cm.discovery.Set("/api/plugins", []byte(`{"plugins": []}`))

	cm.InvalidatePlugin("mcp")

	// Discovery should be cleared since plugin metadata may have changed.
	if _, ok := cm.discovery.Get("/api/plugins"); ok {
		t.Fatal("expected discovery cache to be cleared after plugin invalidation")
	}
}

func testInvalidateAllClearsBothCaches(t *testing.T) {
	cfg := &CacheConfig{
		Enabled:         true,
		DiscoveryTTL:    5 * time.Second,
		CapabilitiesTTL: 5 * time.Second,
		MaxSize:         100,
	}
	cm := NewCacheManager(cfg)

	cm.discovery.Set("/api/plugins", []byte(`{"plugins": []}`))
	cm.capabilities.Set("/api/plugins/mcp/capabilities", []byte(`{"mcp": true}`))
	cm.capabilities.Set("/api/plugins/model/capabilities", []byte(`{"model": true}`))

	cm.InvalidateAll()

	if cm.discovery.Size() != 0 {
		t.Fatalf("expected discovery cache empty, got size %d", cm.discovery.Size())
	}
	if cm.capabilities.Size() != 0 {
		t.Fatalf("expected capabilities cache empty, got size %d", cm.capabilities.Size())
	}
}

func testNilCacheManagerSafe(t *testing.T) {
	// All methods on a nil CacheManager should be no-ops (not panic).
	var cm *CacheManager
	cm.InvalidatePlugin("mcp")
	cm.InvalidateAll()
}

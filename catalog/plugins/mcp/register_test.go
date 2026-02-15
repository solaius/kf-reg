package mcp

import (
	"testing"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

func TestPluginRegistered(t *testing.T) {
	// The init() function in register.go should have registered the plugin
	names := plugin.Names()
	found := false
	for _, name := range names {
		if name == PluginName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("plugin %q not found in registered plugins: %v", PluginName, names)
	}
}

func TestPluginMetadata(t *testing.T) {
	p := &McpServerCatalogPlugin{}

	if p.Name() != "mcp" {
		t.Errorf("expected name 'mcp', got %q", p.Name())
	}
	if p.Version() != "v1alpha1" {
		t.Errorf("expected version 'v1alpha1', got %q", p.Version())
	}
	if p.BasePath() != "/api/mcp_catalog/v1alpha1" {
		t.Errorf("expected basePath '/api/mcp_catalog/v1alpha1', got %q", p.BasePath())
	}
	if p.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestPluginHealthyDefault(t *testing.T) {
	p := &McpServerCatalogPlugin{}

	// Should be false before start
	if p.Healthy() {
		t.Error("expected plugin to be unhealthy before start")
	}
}

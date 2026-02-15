package providers

import (
	"testing"
)

func TestParseMcpServerCatalog(t *testing.T) {
	data := []byte(`
mcpservers:
  - name: "filesystem-server"
    description: "MCP server providing filesystem operations"
    serverUrl: "https://mcp.example.com/filesystem"
    transportType: "stdio"
    toolCount: 5
    resourceCount: 3
    promptCount: 2
  - name: "database-query-server"
    description: "MCP server for database queries"
    serverUrl: "https://mcp.example.com/database"
    transportType: "sse"
    toolCount: 10
    resourceCount: 8
    promptCount: 0
`)

	records, err := parseMcpServerCatalog(data, nil)
	if err != nil {
		t.Fatalf("unexpected error parsing catalog: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// Verify first record
	r0 := records[0]
	attrs0 := r0.Entity.GetAttributes()
	if attrs0 == nil || attrs0.Name == nil || *attrs0.Name != "filesystem-server" {
		t.Errorf("expected first entity name 'filesystem-server', got %v", attrs0)
	}

	// Verify properties are set
	props0 := r0.Entity.GetProperties()
	if props0 == nil {
		t.Fatal("expected properties on first entity")
	}

	propMap := make(map[string]interface{})
	for _, p := range *props0 {
		if p.StringValue != nil {
			propMap[p.Name] = *p.StringValue
		} else if p.IntValue != nil {
			propMap[p.Name] = *p.IntValue
		}
	}

	if v, ok := propMap["description"].(string); !ok || v != "MCP server providing filesystem operations" {
		t.Errorf("expected description 'MCP server providing filesystem operations', got %v", propMap["description"])
	}
	if v, ok := propMap["serverUrl"].(string); !ok || v != "https://mcp.example.com/filesystem" {
		t.Errorf("expected serverUrl 'https://mcp.example.com/filesystem', got %v", propMap["serverUrl"])
	}
	if v, ok := propMap["transportType"].(string); !ok || v != "stdio" {
		t.Errorf("expected transportType 'stdio', got %v", propMap["transportType"])
	}
	if v, ok := propMap["toolCount"].(int32); !ok || v != 5 {
		t.Errorf("expected toolCount 5, got %v", propMap["toolCount"])
	}
	if v, ok := propMap["resourceCount"].(int32); !ok || v != 3 {
		t.Errorf("expected resourceCount 3, got %v", propMap["resourceCount"])
	}
	if v, ok := propMap["promptCount"].(int32); !ok || v != 2 {
		t.Errorf("expected promptCount 2, got %v", propMap["promptCount"])
	}

	// Verify second record
	r1 := records[1]
	attrs1 := r1.Entity.GetAttributes()
	if attrs1 == nil || attrs1.Name == nil || *attrs1.Name != "database-query-server" {
		t.Errorf("expected second entity name 'database-query-server', got %v", attrs1)
	}
}

func TestParseMcpServerCatalogMinimal(t *testing.T) {
	// Test with only required fields
	data := []byte(`
mcpservers:
  - name: "minimal-server"
    serverUrl: "https://mcp.example.com/minimal"
`)

	records, err := parseMcpServerCatalog(data, nil)
	if err != nil {
		t.Fatalf("unexpected error parsing catalog: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	attrs := records[0].Entity.GetAttributes()
	if attrs == nil || attrs.Name == nil || *attrs.Name != "minimal-server" {
		t.Errorf("expected name 'minimal-server', got %v", attrs)
	}

	props := records[0].Entity.GetProperties()
	if props == nil {
		t.Fatal("expected properties on entity")
	}

	// Should have serverUrl but not optional properties like transportType
	hasServerUrl := false
	hasTransportType := false
	for _, p := range *props {
		if p.Name == "serverUrl" {
			hasServerUrl = true
		}
		if p.Name == "transportType" {
			hasTransportType = true
		}
	}
	if !hasServerUrl {
		t.Error("expected serverUrl property")
	}
	if hasTransportType {
		t.Error("did not expect transportType property for minimal record")
	}
}

func TestParseMcpServerCatalogEmpty(t *testing.T) {
	data := []byte(`
mcpservers: []
`)

	records, err := parseMcpServerCatalog(data, nil)
	if err != nil {
		t.Fatalf("unexpected error parsing catalog: %v", err)
	}

	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
}

func TestParseMcpServerCatalogWithCustomProperties(t *testing.T) {
	data := []byte(`
mcpservers:
  - name: "custom-server"
    serverUrl: "https://mcp.example.com/custom"
    customProperties:
      category: "development"
      version: "1.0"
`)

	records, err := parseMcpServerCatalog(data, nil)
	if err != nil {
		t.Fatalf("unexpected error parsing catalog: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	customProps := records[0].Entity.GetCustomProperties()
	if customProps == nil {
		t.Fatal("expected custom properties")
	}

	found := make(map[string]string)
	for _, p := range *customProps {
		if p.StringValue != nil {
			found[p.Name] = *p.StringValue
		}
	}

	if found["category"] != "development" {
		t.Errorf("expected custom property 'category' = 'development', got %q", found["category"])
	}
	if found["version"] != "1.0" {
		t.Errorf("expected custom property 'version' = '1.0', got %q", found["version"])
	}
}

package providers

import (
	"testing"
)

func TestParseMcpServerCatalog(t *testing.T) {
	data := []byte(`
mcpservers:
  - name: "kubernetes-mcp-server"
    description: "MCP server for Kubernetes cluster management"
    serverUrl: "stdio://kubernetes-mcp-server"
    transportType: "stdio"
    deploymentMode: "local"
    image: "quay.io/kubeflow/kubernetes-mcp-server:latest"
    supportedTransports: "stdio,http"
    license: "Apache-2.0"
    verified: true
    certified: true
    provider: "Red Hat"
    category: "Red Hat"
    toolCount: 12
    resourceCount: 8
    promptCount: 3
  - name: "github-mcp-server"
    description: "MCP server for GitHub repository management"
    serverUrl: "https://api.github.com/mcp"
    transportType: "http"
    deploymentMode: "remote"
    endpoint: "https://api.github.com/mcp"
    supportedTransports: "http,sse"
    license: "MIT"
    verified: true
    certified: false
    provider: "GitHub"
    category: "DevOps"
    toolCount: 20
    resourceCount: 15
    promptCount: 5
`)

	records, err := parseMcpServerCatalog(data, nil)
	if err != nil {
		t.Fatalf("unexpected error parsing catalog: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// Verify first record (kubernetes-mcp-server)
	r0 := records[0]
	attrs0 := r0.Entity.GetAttributes()
	if attrs0 == nil || attrs0.Name == nil || *attrs0.Name != "kubernetes-mcp-server" {
		t.Errorf("expected first entity name 'kubernetes-mcp-server', got %v", attrs0)
	}

	// Verify properties are set
	props0 := r0.Entity.GetProperties()
	if props0 == nil {
		t.Fatal("expected properties on first entity")
	}

	propMapStr := make(map[string]string)
	propMapInt := make(map[string]int32)
	propMapBool := make(map[string]bool)
	for _, p := range *props0 {
		if p.StringValue != nil {
			propMapStr[p.Name] = *p.StringValue
		} else if p.IntValue != nil {
			propMapInt[p.Name] = *p.IntValue
		} else if p.BoolValue != nil {
			propMapBool[p.Name] = *p.BoolValue
		}
	}

	// Check existing fields
	if v, ok := propMapStr["description"]; !ok || v != "MCP server for Kubernetes cluster management" {
		t.Errorf("expected description 'MCP server for Kubernetes cluster management', got %v", propMapStr["description"])
	}
	if v, ok := propMapStr["serverUrl"]; !ok || v != "stdio://kubernetes-mcp-server" {
		t.Errorf("expected serverUrl 'stdio://kubernetes-mcp-server', got %v", propMapStr["serverUrl"])
	}
	if v, ok := propMapStr["transportType"]; !ok || v != "stdio" {
		t.Errorf("expected transportType 'stdio', got %v", propMapStr["transportType"])
	}
	if v, ok := propMapInt["toolCount"]; !ok || v != 12 {
		t.Errorf("expected toolCount 12, got %v", propMapInt["toolCount"])
	}
	if v, ok := propMapInt["resourceCount"]; !ok || v != 8 {
		t.Errorf("expected resourceCount 8, got %v", propMapInt["resourceCount"])
	}
	if v, ok := propMapInt["promptCount"]; !ok || v != 3 {
		t.Errorf("expected promptCount 3, got %v", propMapInt["promptCount"])
	}

	// Check new fields
	if v, ok := propMapStr["deploymentMode"]; !ok || v != "local" {
		t.Errorf("expected deploymentMode 'local', got %q", v)
	}
	if v, ok := propMapStr["image"]; !ok || v != "quay.io/kubeflow/kubernetes-mcp-server:latest" {
		t.Errorf("expected image 'quay.io/kubeflow/kubernetes-mcp-server:latest', got %q", v)
	}
	if v, ok := propMapStr["supportedTransports"]; !ok || v != "stdio,http" {
		t.Errorf("expected supportedTransports 'stdio,http', got %q", v)
	}
	if v, ok := propMapStr["license"]; !ok || v != "Apache-2.0" {
		t.Errorf("expected license 'Apache-2.0', got %q", v)
	}
	if v, ok := propMapBool["verified"]; !ok || v != true {
		t.Errorf("expected verified true, got %v", v)
	}
	if v, ok := propMapBool["certified"]; !ok || v != true {
		t.Errorf("expected certified true, got %v", v)
	}
	if v, ok := propMapStr["provider"]; !ok || v != "Red Hat" {
		t.Errorf("expected provider 'Red Hat', got %q", v)
	}
	if v, ok := propMapStr["category"]; !ok || v != "Red Hat" {
		t.Errorf("expected category 'Red Hat', got %q", v)
	}

	// Verify second record (github-mcp-server) - remote deployment
	r1 := records[1]
	attrs1 := r1.Entity.GetAttributes()
	if attrs1 == nil || attrs1.Name == nil || *attrs1.Name != "github-mcp-server" {
		t.Errorf("expected second entity name 'github-mcp-server', got %v", attrs1)
	}

	props1 := r1.Entity.GetProperties()
	if props1 == nil {
		t.Fatal("expected properties on second entity")
	}

	propMapStr1 := make(map[string]string)
	propMapBool1 := make(map[string]bool)
	for _, p := range *props1 {
		if p.StringValue != nil {
			propMapStr1[p.Name] = *p.StringValue
		} else if p.BoolValue != nil {
			propMapBool1[p.Name] = *p.BoolValue
		}
	}

	if v, ok := propMapStr1["deploymentMode"]; !ok || v != "remote" {
		t.Errorf("expected deploymentMode 'remote', got %q", v)
	}
	if v, ok := propMapStr1["endpoint"]; !ok || v != "https://api.github.com/mcp" {
		t.Errorf("expected endpoint 'https://api.github.com/mcp', got %q", v)
	}
	if v, ok := propMapBool1["certified"]; !ok || v != false {
		t.Errorf("expected certified false, got %v", v)
	}
}

func TestParseMcpServerCatalogMinimal(t *testing.T) {
	// Test with only required fields
	data := []byte(`
mcpservers:
  - name: "minimal-server"
    serverUrl: "https://mcp.example.com/minimal"
    deploymentMode: "remote"
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

	// Should have serverUrl and deploymentMode but not optional properties
	hasServerUrl := false
	hasTransportType := false
	hasDeploymentMode := false
	hasProvider := false
	for _, p := range *props {
		switch p.Name {
		case "serverUrl":
			hasServerUrl = true
		case "transportType":
			hasTransportType = true
		case "deploymentMode":
			hasDeploymentMode = true
		case "provider":
			hasProvider = true
		}
	}
	if !hasServerUrl {
		t.Error("expected serverUrl property")
	}
	if hasTransportType {
		t.Error("did not expect transportType property for minimal record")
	}
	if !hasDeploymentMode {
		t.Error("expected deploymentMode property")
	}
	if hasProvider {
		t.Error("did not expect provider property for minimal record")
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

func TestParseMcpServerCatalogValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "missing name",
			data: `
mcpservers:
  - serverUrl: "https://mcp.example.com"
    deploymentMode: "remote"
`,
		},
		{
			name: "missing deploymentMode",
			data: `
mcpservers:
  - name: "test-server"
    serverUrl: "https://mcp.example.com"
`,
		},
		{
			name: "invalid deploymentMode",
			data: `
mcpservers:
  - name: "test-server"
    serverUrl: "https://mcp.example.com"
    deploymentMode: "invalid"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseMcpServerCatalog([]byte(tt.data), nil)
			if err == nil {
				t.Errorf("expected validation error for %s, got nil", tt.name)
			}
		})
	}
}

func TestParseMcpServerCatalogWithCustomProperties(t *testing.T) {
	data := []byte(`
mcpservers:
  - name: "custom-server"
    serverUrl: "https://mcp.example.com/custom"
    deploymentMode: "remote"
    customProperties:
      team: "platform"
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

	if found["team"] != "platform" {
		t.Errorf("expected custom property 'team' = 'platform', got %q", found["team"])
	}
	if found["version"] != "1.0" {
		t.Errorf("expected custom property 'version' = '1.0', got %q", found["version"])
	}
}

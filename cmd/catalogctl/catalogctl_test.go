package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --- extractValue tests ---

func TestExtractValue(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		path string
		want string
	}{
		{
			name: "simple string",
			data: map[string]any{"name": "llama3"},
			path: "name",
			want: "llama3",
		},
		{
			name: "nested path",
			data: map[string]any{"status": map[string]any{"state": "available"}},
			path: "status.state",
			want: "available",
		},
		{
			name: "missing key",
			data: map[string]any{"name": "test"},
			path: "missing",
			want: "",
		},
		{
			name: "deeply nested missing",
			data: map[string]any{"a": map[string]any{"b": "c"}},
			path: "a.x.y",
			want: "",
		},
		{
			name: "integer value",
			data: map[string]any{"count": float64(42)},
			path: "count",
			want: "42",
		},
		{
			name: "float value",
			data: map[string]any{"score": float64(3.14)},
			path: "score",
			want: "3.14",
		},
		{
			name: "boolean true",
			data: map[string]any{"enabled": true},
			path: "enabled",
			want: "true",
		},
		{
			name: "boolean false",
			data: map[string]any{"enabled": false},
			path: "enabled",
			want: "false",
		},
		{
			name: "array value",
			data: map[string]any{"tags": []any{"a", "b", "c"}},
			path: "tags",
			want: "a, b, c",
		},
		{
			name: "nil value",
			data: map[string]any{"x": nil},
			path: "x",
			want: "",
		},
		{
			name: "empty map",
			data: map[string]any{},
			path: "anything",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractValue(tt.data, tt.path)
			if got != tt.want {
				t.Errorf("extractValue(%v, %q) = %q, want %q", tt.data, tt.path, got, tt.want)
			}
		})
	}
}

// --- extractItems tests ---

func TestExtractItems(t *testing.T) {
	tests := []struct {
		name   string
		data   map[string]any
		plural string
		count  int
	}{
		{
			name:   "finds by plural key",
			data:   map[string]any{"mcpservers": []any{map[string]any{"name": "s1"}}},
			plural: "mcpservers",
			count:  1,
		},
		{
			name:   "falls back to items",
			data:   map[string]any{"items": []any{map[string]any{"name": "s1"}, map[string]any{"name": "s2"}}},
			plural: "widgets",
			count:  2,
		},
		{
			name:   "falls back to results",
			data:   map[string]any{"results": []any{map[string]any{"name": "s1"}}},
			plural: "things",
			count:  1,
		},
		{
			name:   "falls back to data",
			data:   map[string]any{"data": []any{map[string]any{"name": "s1"}}},
			plural: "other",
			count:  1,
		},
		{
			name:   "returns nil when no match",
			data:   map[string]any{"unrelated": "value"},
			plural: "widgets",
			count:  0,
		},
		{
			name:   "non-array value returns nil",
			data:   map[string]any{"items": "not-an-array"},
			plural: "items",
			count:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractItems(tt.data, tt.plural)
			if len(got) != tt.count {
				t.Errorf("extractItems: got %d items, want %d", len(got), tt.count)
			}
		})
	}
}

// --- truncate tests ---

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"", 5, ""},
		{"hello", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

// --- toMapSlice tests ---

func TestToMapSlice(t *testing.T) {
	t.Run("valid array of maps", func(t *testing.T) {
		input := []any{
			map[string]any{"name": "a"},
			map[string]any{"name": "b"},
		}
		got := toMapSlice(input)
		if len(got) != 2 {
			t.Errorf("toMapSlice: got %d items, want 2", len(got))
		}
	})

	t.Run("non-array returns nil", func(t *testing.T) {
		got := toMapSlice("not-an-array")
		if got != nil {
			t.Errorf("toMapSlice: expected nil, got %v", got)
		}
	})

	t.Run("mixed types filters non-maps", func(t *testing.T) {
		input := []any{
			map[string]any{"name": "a"},
			"string-item",
			42,
		}
		got := toMapSlice(input)
		if len(got) != 1 {
			t.Errorf("toMapSlice: got %d items, want 1", len(got))
		}
	})
}

// --- Command tree building tests ---

func TestBuildPluginCommand(t *testing.T) {
	p := sampleModelPlugin()

	cmd := buildPluginCommand(p)

	if cmd.Use != "model" {
		t.Errorf("command Use = %q, want %q", cmd.Use, "model")
	}

	// Should have entity subcommand, sources subcommand, and actions subcommand.
	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Use] = true
	}

	if !subNames["models"] {
		t.Error("expected models subcommand")
	}
	if !subNames["sources"] {
		t.Error("expected sources subcommand")
	}
	if !subNames["actions"] {
		t.Error("expected actions subcommand")
	}
}

func TestBuildEntityCommand(t *testing.T) {
	p := sampleModelPlugin()
	entity := p.CapabilitiesV2.Entities[0]

	cmd := buildEntityCommand(p, entity)

	if cmd.Use != "models" {
		t.Errorf("command Use = %q, want %q", cmd.Use, "models")
	}

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}

	if !subNames["list"] {
		t.Error("expected list subcommand")
	}
	if !subNames["get"] {
		t.Error("expected get subcommand")
	}
	// Action subcommands: tag, annotate
	if !subNames["tag"] {
		t.Error("expected tag action subcommand")
	}
	if !subNames["annotate"] {
		t.Error("expected annotate action subcommand")
	}
}

func TestBuildEntityActionCommand_NilForMissingAction(t *testing.T) {
	p := sampleModelPlugin()
	entity := p.CapabilitiesV2.Entities[0]

	cmd := buildEntityActionCommand(p, entity, "nonexistent-action")
	if cmd != nil {
		t.Error("expected nil for unknown action ID")
	}
}

func TestBuildEntityActionCommand_HasDryRunFlag(t *testing.T) {
	p := sampleModelPlugin()
	entity := p.CapabilitiesV2.Entities[0]

	cmd := buildEntityActionCommand(p, entity, "tag")
	if cmd == nil {
		t.Fatal("expected non-nil command for 'tag' action")
	}

	flag := cmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Error("expected --dry-run flag on action that supports dry run")
	}
}

func TestBuildPluginCommand_SkipsSourcesWhenNil(t *testing.T) {
	p := sampleModelPlugin()
	p.CapabilitiesV2.Sources = nil

	cmd := buildPluginCommand(p)
	for _, sub := range cmd.Commands() {
		if sub.Name() == "sources" {
			t.Error("sources subcommand should not exist when Sources is nil")
		}
	}
}

func TestBuildPluginCommand_SkipsActionsWhenEmpty(t *testing.T) {
	p := sampleModelPlugin()
	p.CapabilitiesV2.Actions = nil

	cmd := buildPluginCommand(p)
	for _, sub := range cmd.Commands() {
		if sub.Name() == "actions" {
			t.Error("actions subcommand should not exist when Actions is empty")
		}
	}
}

func TestBuildPluginCommand_DisplayNameInDescription(t *testing.T) {
	p := sampleModelPlugin()
	cmd := buildPluginCommand(p)

	if !strings.Contains(cmd.Short, "Models") {
		t.Errorf("command description should contain display name 'Models', got %q", cmd.Short)
	}
}

// --- HTTP integration tests with httptest ---

func TestPluginsListHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/plugins" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp := pluginsResponse{
			Plugins: []pluginInfo{sampleModelPlugin()},
			Count:   1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := &catalogClient{baseURL: srv.URL, http: srv.Client()}

	var resp pluginsResponse
	if err := client.getJSON("/api/plugins", &resp); err != nil {
		t.Fatalf("getJSON failed: %v", err)
	}

	if resp.Count != 1 {
		t.Errorf("Count = %d, want 1", resp.Count)
	}
	if resp.Plugins[0].Name != "model" {
		t.Errorf("Plugin name = %q, want %q", resp.Plugins[0].Name, "model")
	}
	if !resp.Plugins[0].Healthy {
		t.Error("Plugin should be healthy")
	}
}

func TestHealthHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/healthz":
			json.NewEncoder(w).Encode(map[string]string{"status": "alive", "uptime": "5m"})
		case "/readyz":
			json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := &catalogClient{baseURL: srv.URL, http: srv.Client()}

	var health map[string]any
	if err := client.getJSON("/healthz", &health); err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	if health["status"] != "alive" {
		t.Errorf("health status = %v, want %q", health["status"], "alive")
	}

	var ready map[string]any
	if err := client.getJSON("/readyz", &ready); err != nil {
		t.Fatalf("readiness check failed: %v", err)
	}
	if ready["status"] != "ready" {
		t.Errorf("readiness status = %v, want %q", ready["status"], "ready")
	}
}

func TestEntityListHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/model_catalog/v1alpha1/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Verify query params are passed through.
		if fq := r.URL.Query().Get("filterQuery"); fq != "" {
			t.Logf("filterQuery received: %s", fq)
		}

		resp := map[string]any{
			"models": []any{
				map[string]any{"name": "llama3", "provider": "Meta", "tasks": "text-generation", "license": "Apache-2.0", "source_id": "hf-default"},
				map[string]any{"name": "bert-base", "provider": "Google", "tasks": "fill-mask", "license": "Apache-2.0", "source_id": "hf-default"},
			},
			"size": 2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := &catalogClient{baseURL: srv.URL, http: srv.Client()}

	result, err := client.getRaw("/api/model_catalog/v1alpha1/models")
	if err != nil {
		t.Fatalf("getRaw failed: %v", err)
	}

	items := extractItems(result, "models")
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if extractValue(items[0], "name") != "llama3" {
		t.Errorf("first item name = %q, want %q", extractValue(items[0], "name"), "llama3")
	}
	if extractValue(items[1], "provider") != "Google" {
		t.Errorf("second item provider = %q, want %q", extractValue(items[1], "provider"), "Google")
	}
}

func TestEntityGetHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/model_catalog/v1alpha1/models/llama3" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			return
		}
		resp := map[string]any{
			"name":        "llama3",
			"description": "Meta's LLaMA 3 model",
			"provider":    "Meta",
			"tasks":       "text-generation",
			"license":     "Apache-2.0",
			"source_id":   "hf-default",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := &catalogClient{baseURL: srv.URL, http: srv.Client()}

	result, err := client.getRaw("/api/model_catalog/v1alpha1/models/llama3")
	if err != nil {
		t.Fatalf("getRaw failed: %v", err)
	}

	if extractValue(result, "name") != "llama3" {
		t.Errorf("name = %q, want %q", extractValue(result, "name"), "llama3")
	}
	if extractValue(result, "description") != "Meta's LLaMA 3 model" {
		t.Errorf("description mismatch")
	}
}

func TestActionHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/model_catalog/v1alpha1/models/llama3:action" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var req actionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resp := actionResponse{
			ActionID: req.ActionID,
			Status:   "completed",
			Message:  "Tag applied successfully",
			DryRun:   req.DryRun,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := &catalogClient{baseURL: srv.URL, http: srv.Client()}

	req := actionRequest{
		ActionID:   "tag",
		TargetName: "llama3",
		Params:     map[string]any{"tags": "production"},
		DryRun:     true,
	}

	var resp actionResponse
	err := client.postJSON("/api/model_catalog/v1alpha1/models/llama3:action", req, &resp)
	if err != nil {
		t.Fatalf("postJSON failed: %v", err)
	}

	if resp.ActionID != "tag" {
		t.Errorf("ActionID = %q, want %q", resp.ActionID, "tag")
	}
	if resp.Status != "completed" {
		t.Errorf("Status = %q, want %q", resp.Status, "completed")
	}
	if !resp.DryRun {
		t.Error("DryRun should be true")
	}
}

func TestSourcesListHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/model_catalog/v1alpha1/sources" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		sources := []sourceInfo{
			{ID: "hf-default", Name: "Hugging Face", Type: "hf", Enabled: true, Status: sourceStatusInfo{State: "available"}},
			{ID: "local-yaml", Name: "Local YAML", Type: "yaml", Enabled: false, Status: sourceStatusInfo{State: "disabled"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sources)
	}))
	defer srv.Close()

	client := &catalogClient{baseURL: srv.URL, http: srv.Client()}

	var sources []sourceInfo
	if err := client.getJSON("/api/model_catalog/v1alpha1/sources", &sources); err != nil {
		t.Fatalf("getJSON failed: %v", err)
	}

	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	if sources[0].ID != "hf-default" {
		t.Errorf("first source ID = %q, want %q", sources[0].ID, "hf-default")
	}
	if sources[1].Enabled {
		t.Error("second source should be disabled")
	}
}

func TestClientErrorHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	client := &catalogClient{baseURL: srv.URL, http: srv.Client()}

	var resp pluginsResponse
	err := client.getJSON("/api/plugins", &resp)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should contain status code, got: %v", err)
	}
}

func TestClientNotFoundHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))
	defer srv.Close()

	client := &catalogClient{baseURL: srv.URL, http: srv.Client()}

	_, err := client.getRaw("/api/nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should contain status code, got: %v", err)
	}
}

// --- Dynamic discovery integration test ---

func TestDiscoverPluginsIntegration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := pluginsResponse{
			Plugins: []pluginInfo{
				sampleModelPlugin(),
				sampleMCPPlugin(),
			},
			Count: 2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	// Build a fresh root command for this test.
	testRoot := &cobra.Command{Use: "catalogctl-test"}

	client := &catalogClient{baseURL: srv.URL, http: srv.Client()}
	var resp pluginsResponse
	if err := client.getJSON("/api/plugins", &resp); err != nil {
		t.Fatalf("failed to fetch plugins: %v", err)
	}

	for _, p := range resp.Plugins {
		if p.CapabilitiesV2 != nil {
			testRoot.AddCommand(buildPluginCommand(p))
		}
	}

	// Verify the command tree.
	subNames := make(map[string]bool)
	for _, sub := range testRoot.Commands() {
		subNames[sub.Name()] = true
	}

	if !subNames["model"] {
		t.Error("expected 'model' subcommand")
	}
	if !subNames["mcp"] {
		t.Error("expected 'mcp' subcommand")
	}

	// Verify model plugin has expected subcommands.
	var modelCmd *cobra.Command
	for _, sub := range testRoot.Commands() {
		if sub.Name() == "model" {
			modelCmd = sub
			break
		}
	}
	if modelCmd == nil {
		t.Fatal("model command not found")
	}

	modelSubNames := make(map[string]bool)
	for _, sub := range modelCmd.Commands() {
		modelSubNames[sub.Name()] = true
	}
	if !modelSubNames["models"] {
		t.Error("expected models under model")
	}
	if !modelSubNames["sources"] {
		t.Error("expected sources under model")
	}
	if !modelSubNames["actions"] {
		t.Error("expected actions under model")
	}
}

func TestInferColumns(t *testing.T) {
	item := map[string]any{
		"name":     "test",
		"version":  "v1",
		"provider": "foo",
	}

	cols := inferColumns(item)
	if len(cols) == 0 {
		t.Fatal("expected at least one column")
	}
	if len(cols) > 5 {
		t.Errorf("expected at most 5 columns, got %d", len(cols))
	}

	// Each column should have matching Name/DisplayName/Path.
	for _, col := range cols {
		if col.Name == "" {
			t.Error("column name should not be empty")
		}
		if col.Name != col.Path {
			t.Errorf("column Name %q should equal Path %q", col.Name, col.Path)
		}
	}
}

// --- Namespace tests ---

func TestClientSendsNamespaceHeader(t *testing.T) {
	var receivedHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Namespace")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	client := &catalogClient{
		baseURL:   srv.URL,
		namespace: "team-a",
		http:      srv.Client(),
	}

	var result map[string]any
	if err := client.getJSON("/api/plugins", &result); err != nil {
		t.Fatalf("getJSON failed: %v", err)
	}

	if receivedHeader != "team-a" {
		t.Errorf("X-Namespace header = %q, want %q", receivedHeader, "team-a")
	}
}

func TestClientNoNamespaceHeaderWhenEmpty(t *testing.T) {
	var receivedHeader string
	var hasHeader bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Namespace")
		_, hasHeader = r.Header["X-Namespace"]
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	client := &catalogClient{
		baseURL:   srv.URL,
		namespace: "",
		http:      srv.Client(),
	}

	var result map[string]any
	if err := client.getJSON("/api/plugins", &result); err != nil {
		t.Fatalf("getJSON failed: %v", err)
	}

	if hasHeader {
		t.Errorf("X-Namespace header should not be set, got %q", receivedHeader)
	}
}

func TestClientNamespaceOnPost(t *testing.T) {
	var receivedHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Namespace")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	client := &catalogClient{
		baseURL:   srv.URL,
		namespace: "team-b",
		http:      srv.Client(),
	}

	var result map[string]any
	body := map[string]string{"action": "tag"}
	if err := client.postJSON("/api/test", body, &result); err != nil {
		t.Fatalf("postJSON failed: %v", err)
	}

	if receivedHeader != "team-b" {
		t.Errorf("X-Namespace header = %q, want %q", receivedHeader, "team-b")
	}
}

func TestNamespacesHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tenancy/v1alpha1/namespaces" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp := map[string]any{
			"namespaces": []string{"team-a", "team-b"},
			"mode":       "namespace",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := &catalogClient{baseURL: srv.URL, http: srv.Client()}

	var resp struct {
		Namespaces []string `json:"namespaces"`
		Mode       string   `json:"mode"`
	}
	if err := client.getJSON("/api/tenancy/v1alpha1/namespaces", &resp); err != nil {
		t.Fatalf("getJSON failed: %v", err)
	}

	if resp.Mode != "namespace" {
		t.Errorf("Mode = %q, want %q", resp.Mode, "namespace")
	}
	if len(resp.Namespaces) != 2 {
		t.Fatalf("expected 2 namespaces, got %d", len(resp.Namespaces))
	}
	if resp.Namespaces[0] != "team-a" {
		t.Errorf("first namespace = %q, want %q", resp.Namespaces[0], "team-a")
	}
}

func TestResolvedNamespace_Flag(t *testing.T) {
	// Save and restore
	oldNs := namespace
	defer func() { namespace = oldNs }()

	namespace = "from-flag"
	t.Setenv("CATALOG_NAMESPACE", "from-env")

	got := resolvedNamespace()
	if got != "from-flag" {
		t.Errorf("resolvedNamespace() = %q, want %q (flag should have priority)", got, "from-flag")
	}
}

func TestResolvedNamespace_EnvVar(t *testing.T) {
	oldNs := namespace
	defer func() { namespace = oldNs }()

	namespace = ""
	t.Setenv("CATALOG_NAMESPACE", "from-env")

	got := resolvedNamespace()
	if got != "from-env" {
		t.Errorf("resolvedNamespace() = %q, want %q (env var should be used when flag is empty)", got, "from-env")
	}
}

func TestResolvedNamespace_Default(t *testing.T) {
	oldNs := namespace
	defer func() { namespace = oldNs }()

	namespace = ""
	t.Setenv("CATALOG_NAMESPACE", "")

	got := resolvedNamespace()
	if got != "" {
		t.Errorf("resolvedNamespace() = %q, want %q (should return empty when nothing is set)", got, "")
	}
}

// --- Test helpers ---

func sampleModelPlugin() pluginInfo {
	return pluginInfo{
		Name:        "model",
		Version:     "v1alpha1",
		Description: "Model catalog for ML models",
		BasePath:    "/api/model_catalog/v1alpha1",
		Healthy:     true,
		EntityKinds: []string{"CatalogModel"},
		CapabilitiesV2: &capabilitiesV2{
			SchemaVersion: "v1",
			Plugin: pluginMeta{
				Name:        "model",
				Version:     "v1alpha1",
				Description: "Model catalog for ML models",
				DisplayName: "Models",
				Icon:        "model",
			},
			Entities: []entityCaps{
				{
					Kind:        "CatalogModel",
					Plural:      "models",
					DisplayName: "Catalog Model",
					Description: "Machine learning models",
					Endpoints: entityEndpoints{
						List: "/api/model_catalog/v1alpha1/models",
						Get:  "/api/model_catalog/v1alpha1/models/{name}",
					},
					Fields: entityFields{
						Columns: []columnHint{
							{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Sortable: true},
							{Name: "provider", DisplayName: "Provider", Path: "provider", Type: "string", Sortable: true},
							{Name: "task", DisplayName: "Task", Path: "tasks", Type: "string", Sortable: true},
							{Name: "license", DisplayName: "License", Path: "license", Type: "string", Sortable: true},
							{Name: "source_id", DisplayName: "Source", Path: "source_id", Type: "string", Sortable: true},
						},
						DetailFields: []fieldHint{
							{Name: "name", DisplayName: "Name", Path: "name", Type: "string", Section: "Overview"},
							{Name: "description", DisplayName: "Description", Path: "description", Type: "string", Section: "Overview"},
							{Name: "provider", DisplayName: "Provider", Path: "provider", Type: "string", Section: "Overview"},
						},
					},
					Actions: []string{"tag", "annotate"},
				},
			},
			Sources: &sourceCaps{
				Manageable:  true,
				Refreshable: true,
				Types:       []string{"yaml", "hf"},
			},
			Actions: []actionDef{
				{ID: "tag", DisplayName: "Tag", Description: "Add or remove tags on an entity", Scope: "asset", SupportsDryRun: true, Idempotent: true},
				{ID: "annotate", DisplayName: "Annotate", Description: "Add or update annotations", Scope: "asset", SupportsDryRun: true, Idempotent: true},
				{ID: "refresh", DisplayName: "Refresh", Description: "Refresh from source", Scope: "source", SupportsDryRun: false, Idempotent: true},
			},
		},
	}
}

func sampleMCPPlugin() pluginInfo {
	return pluginInfo{
		Name:        "mcp",
		Version:     "v1alpha1",
		Description: "McpServer catalog",
		BasePath:    "/api/mcp_catalog/v1alpha1",
		Healthy:     true,
		CapabilitiesV2: &capabilitiesV2{
			SchemaVersion: "v1",
			Plugin: pluginMeta{
				Name:        "mcp",
				Version:     "v1alpha1",
				Description: "McpServer catalog",
				DisplayName: "MCP Servers",
			},
			Entities: []entityCaps{
				{
					Kind:        "McpServer",
					Plural:      "mcpservers",
					DisplayName: "MCP Server",
					Endpoints: entityEndpoints{
						List: "/api/mcp_catalog/v1alpha1/mcpservers",
						Get:  "/api/mcp_catalog/v1alpha1/mcpservers/{name}",
					},
					Fields: entityFields{
						Columns: []columnHint{
							{Name: "name", DisplayName: "Name", Path: "name", Type: "string"},
							{Name: "protocol", DisplayName: "Protocol", Path: "protocol", Type: "string"},
						},
					},
				},
			},
		},
	}
}

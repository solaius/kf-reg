// Package agents provides the Agent catalog plugin for the unified catalog server.
// This is an in-memory plugin that loads agent entries from YAML files
// and serves them via REST endpoints. It demonstrates that a new plugin can appear
// in UI and CLI with zero frontend/CLI code changes.
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/kubeflow/model-registry/pkg/catalog"
	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
	gitprovider "github.com/kubeflow/model-registry/pkg/catalog/providers/git"
)

const (
	// PluginName is the identifier for this plugin.
	PluginName = "agents"

	// PluginVersion is the API version.
	PluginVersion = "v1alpha1"
)

// AgentPlugin implements the CatalogPlugin interface for agent catalogs.
type AgentPlugin struct {
	cfg        plugin.Config
	logger     *slog.Logger
	db         *gorm.DB
	healthy    atomic.Bool
	started    atomic.Bool
	mu         sync.RWMutex
	sources    map[string][]AgentEntry // sourceID -> entries
	gitCancels map[string]context.CancelFunc // sourceID -> cancel for git background sync
}

// AgentEntry is the in-memory representation of an agent.
type AgentEntry struct {
	Name             string           `yaml:"name" json:"name"`
	ExternalId       string           `yaml:"externalId" json:"externalId,omitempty"`
	Description      *string          `yaml:"description" json:"description,omitempty"`
	AgentType        *string          `yaml:"agentType" json:"agentType,omitempty"`
	Instructions     *string          `yaml:"instructions" json:"instructions,omitempty"`
	Version          *string          `yaml:"version" json:"version,omitempty"`
	ModelConfig      map[string]any   `yaml:"modelConfig" json:"modelConfig,omitempty"`
	Tools            []map[string]any `yaml:"tools" json:"tools,omitempty"`
	Knowledge        []map[string]any `yaml:"knowledge" json:"knowledge,omitempty"`
	Guardrails       []map[string]any `yaml:"guardrails" json:"guardrails,omitempty"`
	Policies         []map[string]any `yaml:"policies" json:"policies,omitempty"`
	PromptRefs       []map[string]any `yaml:"promptRefs" json:"promptRefs,omitempty"`
	Dependencies     []map[string]any `yaml:"dependencies" json:"dependencies,omitempty"`
	InputSchema      map[string]any   `yaml:"inputSchema" json:"inputSchema,omitempty"`
	OutputSchema     map[string]any   `yaml:"outputSchema" json:"outputSchema,omitempty"`
	Examples         []map[string]any `yaml:"examples" json:"examples,omitempty"`
	Author           *string          `yaml:"author" json:"author,omitempty"`
	License          *string          `yaml:"license" json:"license,omitempty"`
	Category         *string          `yaml:"category" json:"category,omitempty"`
	CustomProperties map[string]any   `yaml:"customProperties" json:"customProperties,omitempty"`
	SourceId         string           `yaml:"-" json:"sourceId,omitempty"`
}

// agentCatalog is the YAML wrapper for deserialization.
type agentCatalog struct {
	Agents []AgentEntry `yaml:"agents"`
}

// Name returns the plugin name.
func (p *AgentPlugin) Name() string {
	return PluginName
}

// Version returns the plugin API version.
func (p *AgentPlugin) Version() string {
	return PluginVersion
}

// Description returns a human-readable description.
func (p *AgentPlugin) Description() string {
	return "Agent catalog for AI agents and multi-agent orchestrations"
}

// BasePath returns the API base path for this plugin.
func (p *AgentPlugin) BasePath() string {
	return "/api/agents_catalog/v1alpha1"
}

// Healthy returns true if the plugin is functioning correctly.
func (p *AgentPlugin) Healthy() bool {
	return p.healthy.Load()
}

// Init initializes the plugin with configuration.
func (p *AgentPlugin) Init(ctx context.Context, cfg plugin.Config) error {
	p.cfg = cfg
	p.logger = cfg.Logger
	if p.logger == nil {
		p.logger = slog.Default()
	}
	p.db = cfg.DB
	p.sources = make(map[string][]AgentEntry)
	p.gitCancels = make(map[string]context.CancelFunc)

	p.logger.Info("initializing agents plugin")

	// Load data from configured sources.
	for _, src := range cfg.Section.Sources {
		if !src.IsEnabled() {
			continue
		}
		switch src.Type {
		case "yaml":
			entries, err := loadYAMLSource(src)
			if err != nil {
				p.logger.Error("failed to load agent source", "source", src.ID, "error", err)
				continue
			}
			for i := range entries {
				entries[i].SourceId = src.ID
			}
			p.sources[src.ID] = entries
			p.logger.Info("loaded agent source", "source", src.ID, "type", "yaml", "entries", len(entries))
		case "git":
			entries, cancel, err := loadGitSource(ctx, src, p.logger)
			if err != nil {
				p.logger.Error("failed to load agent git source", "source", src.ID, "error", err)
				continue
			}
			for i := range entries {
				entries[i].SourceId = src.ID
			}
			p.sources[src.ID] = entries
			if cancel != nil {
				p.gitCancels[src.ID] = cancel
			}
			p.logger.Info("loaded agent source", "source", src.ID, "type", "git", "entries", len(entries))
		default:
			p.logger.Warn("unsupported source type", "source", src.ID, "type", src.Type)
		}
	}

	p.healthy.Store(true)
	p.logger.Info("agents plugin initialized", "sources", len(p.sources))
	return nil
}

// Start begins background operations.
func (p *AgentPlugin) Start(ctx context.Context) error {
	p.logger.Info("starting agents plugin")
	p.started.Store(true)
	p.logger.Info("agents plugin started")
	return nil
}

// Stop gracefully shuts down the plugin.
func (p *AgentPlugin) Stop(ctx context.Context) error {
	p.logger.Info("stopping agents plugin")
	// Cancel any running git provider goroutines.
	for id, cancel := range p.gitCancels {
		p.logger.Info("canceling git provider", "source", id)
		cancel()
	}
	p.gitCancels = nil
	p.started.Store(false)
	p.healthy.Store(false)
	return nil
}

// RegisterRoutes mounts the plugin's HTTP routes on the provided router.
func (p *AgentPlugin) RegisterRoutes(router chi.Router) error {
	p.logger.Info("registering agents routes")
	router.Get("/agents", p.listHandler)
	router.Get("/agents/{name}", p.getHandler)
	return nil
}

// Migrations returns database migrations for this plugin.
func (p *AgentPlugin) Migrations() []plugin.Migration {
	// In-memory only, no DB persistence needed.
	return nil
}

// allEntries returns all entries from all loaded sources.
func (p *AgentPlugin) allEntries() []AgentEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var all []AgentEntry
	for _, entries := range p.sources {
		all = append(all, entries...)
	}
	return all
}

// listHandler returns all agents, optionally filtered by filterQuery.
func (p *AgentPlugin) listHandler(w http.ResponseWriter, r *http.Request) {
	entries := p.allEntries()

	// Apply basic filterQuery support.
	filterQuery := r.URL.Query().Get("filterQuery")
	if filterQuery != "" {
		entries = applyFilter(entries, filterQuery)
	}

	response := map[string]any{
		"items":    entries,
		"size":     len(entries),
		"pageSize": len(entries),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		p.logger.Error("failed to encode response", "error", err)
	}
}

// getHandler returns a single agent by name.
func (p *AgentPlugin) getHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	entries := p.allEntries()

	for _, entry := range entries {
		if entry.Name == name {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(entry); err != nil {
				p.logger.Error("failed to encode response", "error", err)
			}
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{
		"error": fmt.Sprintf("agent %q not found", name),
	})
}

// loadYAMLSource reads and parses a YAML file with agent entries.
func loadYAMLSource(src plugin.SourceConfig) ([]AgentEntry, error) {
	yamlPath, ok := src.Properties["yamlCatalogPath"].(string)
	if !ok || yamlPath == "" {
		// Fall back to "path" property.
		yamlPath, ok = src.Properties["path"].(string)
		if !ok || yamlPath == "" {
			return nil, fmt.Errorf("source %q missing yamlCatalogPath or path property", src.ID)
		}
	}

	// Resolve relative paths against the source origin directory.
	if !filepath.IsAbs(yamlPath) && src.Origin != "" {
		yamlPath = filepath.Join(filepath.Dir(src.Origin), yamlPath)
	}

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", yamlPath, err)
	}

	var catalog agentCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", yamlPath, err)
	}

	return catalog.Agents, nil
}

// loadGitSource clones a Git repository and loads agent entries from matching YAML files.
// Returns the loaded entries and a cancel function to stop background sync.
func loadGitSource(ctx context.Context, src plugin.SourceConfig, logger *slog.Logger) ([]AgentEntry, context.CancelFunc, error) {
	// Convert plugin.SourceConfig to catalog.Source for the git provider.
	catalogSource := &catalog.Source{
		ID:         src.ID,
		Name:       src.Name,
		Type:       src.Type,
		Properties: src.Properties,
		Origin:     src.Origin,
	}
	if src.Enabled != nil {
		catalogSource.Enabled = src.Enabled
	}

	gitCfg := gitprovider.Config[AgentEntry, any]{
		Parse:  parseAgentRecords,
		Logger: &slogGitLogger{logger: logger},
	}

	provider, err := gitprovider.NewProvider(gitCfg, catalogSource, "")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create git provider: %w", err)
	}

	// Create a cancellable context for the provider's background sync.
	providerCtx, cancel := context.WithCancel(ctx)

	ch, err := provider.Records(providerCtx)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to start git provider: %w", err)
	}

	// Drain the initial batch (records until the zero-value batch marker).
	var entries []AgentEntry
	for record := range ch {
		if record.Entity.Name == "" {
			// Batch completion marker (zero-value entity).
			break
		}
		entries = append(entries, record.Entity)
	}

	logger.Info("loaded agents from git",
		"source", src.ID,
		"entries", len(entries),
		"commit", provider.LastCommit(),
	)

	return entries, cancel, nil
}

// parseAgentRecords parses YAML bytes into catalog records for the git provider.
func parseAgentRecords(data []byte) ([]catalog.Record[AgentEntry, any], error) {
	var cat agentCatalog
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, err
	}
	records := make([]catalog.Record[AgentEntry, any], len(cat.Agents))
	for i, agent := range cat.Agents {
		records[i] = catalog.Record[AgentEntry, any]{Entity: agent}
	}
	return records, nil
}

// slogGitLogger adapts slog.Logger to the git provider's Logger interface.
type slogGitLogger struct {
	logger *slog.Logger
}

func (l *slogGitLogger) Infof(format string, args ...any) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l *slogGitLogger) Errorf(format string, args ...any) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

// applyFilter applies a simple filterQuery to the entries.
// Supports basic equality filters: field='value' and AND combinations.
func applyFilter(entries []AgentEntry, query string) []AgentEntry {
	conditions := parseFilterConditions(query)
	if len(conditions) == 0 {
		return entries
	}

	var filtered []AgentEntry
	for _, entry := range entries {
		if matchesAllConditions(entry, conditions) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

type filterCondition struct {
	field    string
	operator string
	value    string
}

func parseFilterConditions(query string) []filterCondition {
	var conditions []filterCondition

	parts := splitByAND(query)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Try operators in order of length to avoid partial matches.
		for _, op := range []string{"!=", ">=", "<=", "=", ">", "<", " LIKE "} {
			idx := strings.Index(part, op)
			if idx >= 0 {
				field := strings.TrimSpace(part[:idx])
				value := strings.TrimSpace(part[idx+len(op):])
				value = strings.Trim(value, "'\"")
				conditions = append(conditions, filterCondition{
					field:    field,
					operator: strings.TrimSpace(op),
					value:    value,
				})
				break
			}
		}
	}
	return conditions
}

func splitByAND(s string) []string {
	upper := strings.ToUpper(s)
	var parts []string
	for {
		idx := strings.Index(upper, " AND ")
		if idx < 0 {
			parts = append(parts, s)
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+5:]
		upper = upper[idx+5:]
	}
	return parts
}

func matchesAllConditions(entry AgentEntry, conditions []filterCondition) bool {
	for _, cond := range conditions {
		if !matchesCondition(entry, cond) {
			return false
		}
	}
	return true
}

func matchesCondition(entry AgentEntry, cond filterCondition) bool {
	fieldValue := getFieldValue(entry, cond.field)

	switch cond.operator {
	case "=":
		return strings.EqualFold(fieldValue, cond.value)
	case "!=":
		return !strings.EqualFold(fieldValue, cond.value)
	case "LIKE":
		pattern := strings.ReplaceAll(strings.ToLower(cond.value), "%", "")
		return strings.Contains(strings.ToLower(fieldValue), pattern)
	default:
		return strings.EqualFold(fieldValue, cond.value)
	}
}

func getFieldValue(entry AgentEntry, field string) string {
	switch field {
	case "name":
		return entry.Name
	case "agentType":
		return ptrStr(entry.AgentType)
	case "category":
		return ptrStr(entry.Category)
	case "version":
		return ptrStr(entry.Version)
	case "author":
		return ptrStr(entry.Author)
	case "license":
		return ptrStr(entry.License)
	case "sourceId":
		return entry.SourceId
	default:
		return ""
	}
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

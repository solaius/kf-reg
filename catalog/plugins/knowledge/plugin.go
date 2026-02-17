// Package knowledge provides the KnowledgeSource catalog plugin for the unified catalog server.
// This is an in-memory plugin that loads knowledge source entries from YAML files
// and serves them via REST endpoints. It demonstrates that a new plugin can appear
// in UI and CLI with zero frontend/CLI code changes.
package knowledge

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

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

const (
	// PluginName is the identifier for this plugin.
	PluginName = "knowledge"

	// PluginVersion is the API version.
	PluginVersion = "v1alpha1"
)

// KnowledgeSourcePlugin implements the CatalogPlugin interface for knowledge source catalogs.
type KnowledgeSourcePlugin struct {
	cfg     plugin.Config
	logger  *slog.Logger
	db      *gorm.DB
	healthy atomic.Bool
	started atomic.Bool
	mu      sync.RWMutex
	sources map[string][]KnowledgeSourceEntry // sourceID -> entries
}

// KnowledgeSourceEntry is the in-memory representation of a knowledge source.
type KnowledgeSourceEntry struct {
	Name             string         `yaml:"name" json:"name"`
	ExternalId       string         `yaml:"externalId" json:"externalId,omitempty"`
	Description      *string        `yaml:"description" json:"description,omitempty"`
	SourceType       *string        `yaml:"sourceType" json:"sourceType,omitempty"`
	Location         *string        `yaml:"location" json:"location,omitempty"`
	ContentType      *string        `yaml:"contentType" json:"contentType,omitempty"`
	Provider         *string        `yaml:"provider" json:"provider,omitempty"`
	Status           *string        `yaml:"status" json:"status,omitempty"`
	DocumentCount    *int32         `yaml:"documentCount" json:"documentCount,omitempty"`
	VectorDimensions *int32         `yaml:"vectorDimensions" json:"vectorDimensions,omitempty"`
	IndexType        *string        `yaml:"indexType" json:"indexType,omitempty"`
	CustomProperties map[string]any `yaml:"customProperties" json:"customProperties,omitempty"`
	SourceId         string         `yaml:"-" json:"sourceId,omitempty"`
}

// knowledgeSourceCatalog is the YAML wrapper for deserialization.
type knowledgeSourceCatalog struct {
	KnowledgeSources []KnowledgeSourceEntry `yaml:"knowledgesources"`
}

// Name returns the plugin name.
func (p *KnowledgeSourcePlugin) Name() string {
	return PluginName
}

// Version returns the plugin API version.
func (p *KnowledgeSourcePlugin) Version() string {
	return PluginVersion
}

// Description returns a human-readable description.
func (p *KnowledgeSourcePlugin) Description() string {
	return "Knowledge source catalog for documents, vector stores, and graph stores"
}

// BasePath returns the API base path for this plugin.
func (p *KnowledgeSourcePlugin) BasePath() string {
	return "/api/knowledge_catalog/v1alpha1"
}

// Healthy returns true if the plugin is functioning correctly.
func (p *KnowledgeSourcePlugin) Healthy() bool {
	return p.healthy.Load()
}

// Init initializes the plugin with configuration.
func (p *KnowledgeSourcePlugin) Init(ctx context.Context, cfg plugin.Config) error {
	p.cfg = cfg
	p.logger = cfg.Logger
	if p.logger == nil {
		p.logger = slog.Default()
	}
	p.db = cfg.DB
	p.sources = make(map[string][]KnowledgeSourceEntry)

	p.logger.Info("initializing knowledge plugin")

	// Load data from configured YAML sources.
	for _, src := range cfg.Section.Sources {
		if !src.IsEnabled() {
			continue
		}
		if src.Type == "yaml" {
			entries, err := loadYAMLSource(src)
			if err != nil {
				p.logger.Error("failed to load knowledge source", "source", src.ID, "error", err)
				continue
			}
			// Set sourceId on each entry.
			for i := range entries {
				entries[i].SourceId = src.ID
			}
			p.sources[src.ID] = entries
			p.logger.Info("loaded knowledge source", "source", src.ID, "entries", len(entries))
		}
	}

	p.healthy.Store(true)
	p.logger.Info("knowledge plugin initialized", "sources", len(p.sources))
	return nil
}

// Start begins background operations.
func (p *KnowledgeSourcePlugin) Start(ctx context.Context) error {
	p.logger.Info("starting knowledge plugin")
	p.started.Store(true)
	p.logger.Info("knowledge plugin started")
	return nil
}

// Stop gracefully shuts down the plugin.
func (p *KnowledgeSourcePlugin) Stop(ctx context.Context) error {
	p.logger.Info("stopping knowledge plugin")
	p.started.Store(false)
	p.healthy.Store(false)
	return nil
}

// RegisterRoutes mounts the plugin's HTTP routes on the provided router.
func (p *KnowledgeSourcePlugin) RegisterRoutes(router chi.Router) error {
	p.logger.Info("registering knowledge routes")
	router.Get("/knowledgesources", p.listHandler)
	router.Get("/knowledgesources/{name}", p.getHandler)
	return nil
}

// Migrations returns database migrations for this plugin.
func (p *KnowledgeSourcePlugin) Migrations() []plugin.Migration {
	// In-memory only, no DB persistence needed.
	return nil
}

// allEntries returns all entries from all loaded sources.
func (p *KnowledgeSourcePlugin) allEntries() []KnowledgeSourceEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var all []KnowledgeSourceEntry
	for _, entries := range p.sources {
		all = append(all, entries...)
	}
	return all
}

// listHandler returns all knowledge sources, optionally filtered by filterQuery.
func (p *KnowledgeSourcePlugin) listHandler(w http.ResponseWriter, r *http.Request) {
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

// getHandler returns a single knowledge source by name.
func (p *KnowledgeSourcePlugin) getHandler(w http.ResponseWriter, r *http.Request) {
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
		"error": fmt.Sprintf("knowledge source %q not found", name),
	})
}

// loadYAMLSource reads and parses a YAML file with knowledge source entries.
func loadYAMLSource(src plugin.SourceConfig) ([]KnowledgeSourceEntry, error) {
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

	var catalog knowledgeSourceCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", yamlPath, err)
	}

	return catalog.KnowledgeSources, nil
}

// applyFilter applies a simple filterQuery to the entries.
// Supports basic equality filters: field='value' and AND combinations.
func applyFilter(entries []KnowledgeSourceEntry, query string) []KnowledgeSourceEntry {
	conditions := parseFilterConditions(query)
	if len(conditions) == 0 {
		return entries
	}

	var filtered []KnowledgeSourceEntry
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

func matchesAllConditions(entry KnowledgeSourceEntry, conditions []filterCondition) bool {
	for _, cond := range conditions {
		if !matchesCondition(entry, cond) {
			return false
		}
	}
	return true
}

func matchesCondition(entry KnowledgeSourceEntry, cond filterCondition) bool {
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

func getFieldValue(entry KnowledgeSourceEntry, field string) string {
	switch field {
	case "name":
		return entry.Name
	case "sourceType":
		return ptrStr(entry.SourceType)
	case "location":
		return ptrStr(entry.Location)
	case "contentType":
		return ptrStr(entry.ContentType)
	case "provider":
		return ptrStr(entry.Provider)
	case "status":
		return ptrStr(entry.Status)
	case "indexType":
		return ptrStr(entry.IndexType)
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

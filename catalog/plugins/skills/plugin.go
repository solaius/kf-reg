// Package skills provides the Skills catalog plugin for the unified catalog server.
// This is an in-memory plugin that loads skill entries from YAML files
// and serves them via REST endpoints. It demonstrates that a new plugin can appear
// in UI and CLI with zero frontend/CLI code changes.
package skills

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
	PluginName = "skills"

	// PluginVersion is the API version.
	PluginVersion = "v1alpha1"
)

// SkillPlugin implements the CatalogPlugin interface for skill catalogs.
type SkillPlugin struct {
	cfg     plugin.Config
	logger  *slog.Logger
	db      *gorm.DB
	healthy atomic.Bool
	started atomic.Bool
	mu      sync.RWMutex
	sources map[string][]SkillEntry // sourceID -> entries
}

// SkillEntry is the in-memory representation of a skill.
type SkillEntry struct {
	Name             string         `yaml:"name" json:"name"`
	ExternalId       string         `yaml:"externalId" json:"externalId,omitempty"`
	Description      *string        `yaml:"description" json:"description,omitempty"`
	SkillType        *string        `yaml:"skillType" json:"skillType,omitempty"`
	InputSchema      map[string]any `yaml:"inputSchema" json:"inputSchema,omitempty"`
	OutputSchema     map[string]any `yaml:"outputSchema" json:"outputSchema,omitempty"`
	Execution        map[string]any `yaml:"execution" json:"execution,omitempty"`
	Safety           map[string]any `yaml:"safety" json:"safety,omitempty"`
	RateLimit        map[string]any `yaml:"rateLimit" json:"rateLimit,omitempty"`
	TimeoutSeconds   *int32         `yaml:"timeoutSeconds" json:"timeoutSeconds,omitempty"`
	RetryPolicy      map[string]any `yaml:"retryPolicy" json:"retryPolicy,omitempty"`
	Compatibility    map[string]any `yaml:"compatibility" json:"compatibility,omitempty"`
	Version          *string        `yaml:"version" json:"version,omitempty"`
	Author           *string        `yaml:"author" json:"author,omitempty"`
	License          *string        `yaml:"license" json:"license,omitempty"`
	CustomProperties map[string]any `yaml:"customProperties" json:"customProperties,omitempty"`
	SourceId         string         `yaml:"-" json:"sourceId,omitempty"`
}

// skillCatalog is the YAML wrapper for deserialization.
type skillCatalog struct {
	Skills []SkillEntry `yaml:"skills"`
}

// Name returns the plugin name.
func (p *SkillPlugin) Name() string {
	return PluginName
}

// Version returns the plugin API version.
func (p *SkillPlugin) Version() string {
	return PluginVersion
}

// Description returns a human-readable description.
func (p *SkillPlugin) Description() string {
	return "Skill catalog for tools, operations, and executable actions"
}

// BasePath returns the API base path for this plugin.
func (p *SkillPlugin) BasePath() string {
	return "/api/skills_catalog/v1alpha1"
}

// Healthy returns true if the plugin is functioning correctly.
func (p *SkillPlugin) Healthy() bool {
	return p.healthy.Load()
}

// Init initializes the plugin with configuration.
func (p *SkillPlugin) Init(ctx context.Context, cfg plugin.Config) error {
	p.cfg = cfg
	p.logger = cfg.Logger
	if p.logger == nil {
		p.logger = slog.Default()
	}
	p.db = cfg.DB
	p.sources = make(map[string][]SkillEntry)

	p.logger.Info("initializing skills plugin")

	// Load data from configured YAML sources.
	for _, src := range cfg.Section.Sources {
		if !src.IsEnabled() {
			continue
		}
		if src.Type == "yaml" {
			entries, err := loadYAMLSource(src)
			if err != nil {
				p.logger.Error("failed to load skill source", "source", src.ID, "error", err)
				continue
			}
			// Set sourceId on each entry.
			for i := range entries {
				entries[i].SourceId = src.ID
			}
			p.sources[src.ID] = entries
			p.logger.Info("loaded skill source", "source", src.ID, "entries", len(entries))
		}
	}

	p.healthy.Store(true)
	p.logger.Info("skills plugin initialized", "sources", len(p.sources))
	return nil
}

// Start begins background operations.
func (p *SkillPlugin) Start(ctx context.Context) error {
	p.logger.Info("starting skills plugin")
	p.started.Store(true)
	p.logger.Info("skills plugin started")
	return nil
}

// Stop gracefully shuts down the plugin.
func (p *SkillPlugin) Stop(ctx context.Context) error {
	p.logger.Info("stopping skills plugin")
	p.started.Store(false)
	p.healthy.Store(false)
	return nil
}

// RegisterRoutes mounts the plugin's HTTP routes on the provided router.
func (p *SkillPlugin) RegisterRoutes(router chi.Router) error {
	p.logger.Info("registering skills routes")
	router.Get("/skills", p.listHandler)
	router.Get("/skills/{name}", p.getHandler)
	return nil
}

// Migrations returns database migrations for this plugin.
func (p *SkillPlugin) Migrations() []plugin.Migration {
	// In-memory only, no DB persistence needed.
	return nil
}

// allEntries returns all entries from all loaded sources.
func (p *SkillPlugin) allEntries() []SkillEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var all []SkillEntry
	for _, entries := range p.sources {
		all = append(all, entries...)
	}
	return all
}

// listHandler returns all skills, optionally filtered by filterQuery.
func (p *SkillPlugin) listHandler(w http.ResponseWriter, r *http.Request) {
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

// getHandler returns a single skill by name.
func (p *SkillPlugin) getHandler(w http.ResponseWriter, r *http.Request) {
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
		"error": fmt.Sprintf("skill %q not found", name),
	})
}

// loadYAMLSource reads and parses a YAML file with skill entries.
func loadYAMLSource(src plugin.SourceConfig) ([]SkillEntry, error) {
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

	var catalog skillCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", yamlPath, err)
	}

	return catalog.Skills, nil
}

// applyFilter applies a simple filterQuery to the entries.
// Supports basic equality filters: field='value' and AND combinations.
func applyFilter(entries []SkillEntry, query string) []SkillEntry {
	conditions := parseFilterConditions(query)
	if len(conditions) == 0 {
		return entries
	}

	var filtered []SkillEntry
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

func matchesAllConditions(entry SkillEntry, conditions []filterCondition) bool {
	for _, cond := range conditions {
		if !matchesCondition(entry, cond) {
			return false
		}
	}
	return true
}

func matchesCondition(entry SkillEntry, cond filterCondition) bool {
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

func getFieldValue(entry SkillEntry, field string) string {
	switch field {
	case "name":
		return entry.Name
	case "skillType":
		return ptrStr(entry.SkillType)
	case "version":
		return ptrStr(entry.Version)
	case "author":
		return ptrStr(entry.Author)
	case "license":
		return ptrStr(entry.License)
	case "sourceId":
		return entry.SourceId
	case "riskLevel":
		if s, ok := entry.Safety["riskLevel"].(string); ok {
			return s
		}
		return ""
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

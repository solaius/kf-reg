// Package plugin provides a plugin-based architecture for catalog services.
// Catalog types (models, datasets, etc.) register as plugins via init() and
// are mounted under a unified HTTP server.
package plugin

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// CatalogPlugin defines the interface that all catalog plugins must implement.
// Plugins register themselves via init() using the Register function.
type CatalogPlugin interface {
	// Identity returns the plugin name (e.g., "models", "datasets").
	// This name is used for routing and configuration lookup.
	Name() string

	// Version returns the API version (e.g., "v1alpha1").
	Version() string

	// Description returns a human-readable description of the plugin.
	Description() string

	// Init initializes the plugin with its configuration.
	// Called once during server startup before Start.
	Init(ctx context.Context, cfg Config) error

	// Start begins background operations (hot-reload, watchers, etc.).
	// Called after Init and after database migrations.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the plugin.
	// Called during server shutdown.
	Stop(ctx context.Context) error

	// Healthy returns true if the plugin is functioning correctly.
	// Used for health check endpoints.
	Healthy() bool

	// RegisterRoutes mounts the plugin's HTTP routes on the provided router.
	// The router is already scoped to the plugin's base path.
	RegisterRoutes(router chi.Router) error

	// Migrations returns database migrations for this plugin.
	// Migrations are applied in order during server initialization.
	Migrations() []Migration
}

// BasePathProvider is an optional interface that plugins can implement
// to specify their own API base path. If not implemented, the server
// computes it as /api/{name}_catalog/{version}.
type BasePathProvider interface {
	BasePath() string
}

// SourceKeyProvider is an optional interface that plugins can implement
// to specify which key in the sources.yaml "catalogs" map they respond to.
// If not implemented, the plugin name is used as the config key.
// This allows the plugin name and config key to differ (e.g., plugin "model"
// can read from the "models" config section).
type SourceKeyProvider interface {
	SourceKey() string
}

// PluginCapabilities describes what a plugin supports.
type PluginCapabilities struct {
	EntityKinds  []string `json:"entityKinds"`
	ListEntities bool     `json:"listEntities"`
	GetEntity    bool     `json:"getEntity"`
	ListSources  bool     `json:"listSources"`
	Artifacts    bool     `json:"artifacts"`
}

// CapabilitiesProvider is an optional interface that plugins can implement
// to advertise their capabilities for generic UI/CLI discovery.
type CapabilitiesProvider interface {
	Capabilities() PluginCapabilities
}

// PluginStatus provides detailed status information about a plugin.
type PluginStatus struct {
	Enabled     bool   `json:"enabled"`
	Initialized bool   `json:"initialized"`
	Serving     bool   `json:"serving"`
	LastError   string `json:"lastError,omitempty"`
}

// StatusProvider is an optional interface that plugins can implement
// to provide detailed status beyond the boolean Healthy() check.
type StatusProvider interface {
	Status() PluginStatus
}

// SourceManager is an optional interface that plugins can implement
// to support runtime management of data sources (add/edit/delete/enable).
type SourceManager interface {
	// ListSources returns information about all configured sources.
	ListSources(ctx context.Context) ([]SourceInfo, error)

	// ValidateSource validates a source configuration without applying it.
	ValidateSource(ctx context.Context, src SourceConfigInput) (*ValidationResult, error)

	// ApplySource adds or updates a source configuration.
	ApplySource(ctx context.Context, src SourceConfigInput) error

	// EnableSource enables or disables a source.
	EnableSource(ctx context.Context, id string, enabled bool) error

	// DeleteSource removes a source and its associated entities.
	DeleteSource(ctx context.Context, id string) error
}

// RefreshProvider is an optional interface that plugins can implement
// to support on-demand refresh of data from sources.
type RefreshProvider interface {
	// Refresh triggers a reload of a specific source.
	Refresh(ctx context.Context, sourceID string) (*RefreshResult, error)

	// RefreshAll triggers a reload of all sources.
	RefreshAll(ctx context.Context) (*RefreshResult, error)
}

// DiagnosticsProvider is an optional interface that plugins can implement
// to provide diagnostic information about plugin health and source status.
type DiagnosticsProvider interface {
	// Diagnostics returns diagnostic information about the plugin.
	Diagnostics(ctx context.Context) (*PluginDiagnostics, error)
}

// UIHintsProvider is an optional interface that plugins can implement
// to provide display hints for rendering entities in the UI.
type UIHintsProvider interface {
	// UIHints returns display hints for the UI.
	UIHints() UIHints
}

// CLIHintsProvider is an optional interface that plugins can implement
// to provide display hints for CLI table rendering.
type CLIHintsProvider interface {
	// CLIHints returns display hints for the CLI.
	CLIHints() CLIHints
}

// EntityGetter is an optional interface that plugins can implement to support
// retrieving a single entity by name through the management API. This is useful
// when the plugin's native Get endpoint requires multiple path parameters
// (e.g., /sources/{source_id}/models/{name}) and cannot be used by generic
// clients that only know the entity name.
type EntityGetter interface {
	// GetEntityByName retrieves a single entity by kind and name.
	// Returns nil, nil if the entity is not found.
	GetEntityByName(ctx context.Context, entityKind string, name string) (map[string]any, error)
}

// CapabilitiesV2Provider is an optional interface that plugins can implement
// to provide the full V2 capabilities discovery document directly.
// If not implemented, BuildCapabilitiesV2 assembles a V2 document from
// the V1 CapabilitiesProvider and other optional interfaces.
type CapabilitiesV2Provider interface {
	GetCapabilitiesV2() PluginCapabilitiesV2
}

// AssetMapperProvider is defined in asset_mapper.go.
// Plugins that implement it can project their native entities into the
// universal AssetResource envelope for generic UI/CLI consumption.

// Migration represents a database migration for a plugin.
type Migration struct {
	// Version is a unique identifier for this migration (e.g., "001", "20240101_initial").
	Version string

	// Description provides a human-readable description of what this migration does.
	Description string

	// Up applies the migration.
	Up func(db *gorm.DB) error

	// Down reverts the migration.
	Down func(db *gorm.DB) error
}

// Config is passed to each plugin during Init.
type Config struct {
	// Section contains the plugin-specific configuration from sources.yaml.
	Section CatalogSection

	// DB is the shared database connection.
	DB *gorm.DB

	// Logger is a namespaced logger for this plugin.
	Logger *slog.Logger

	// BasePath is the API base path for this plugin (e.g., "/api/models_catalog/v1alpha1").
	BasePath string

	// ConfigPaths are the paths to all sources.yaml files being used.
	ConfigPaths []string
}

package plugin

import (
	"context"
	"errors"
	"time"
)

// ErrVersionConflict is returned when a Save detects that the underlying config
// has been modified since the caller last loaded it (optimistic concurrency).
var ErrVersionConflict = errors.New("config version conflict: file was modified since last load")

// ConfigRevision represents a historical snapshot of a configuration save.
type ConfigRevision struct {
	// Version is the content hash at the time of the snapshot.
	Version string `json:"version"`

	// Timestamp is when the snapshot was created.
	Timestamp time.Time `json:"timestamp"`

	// Size is the byte size of the configuration file at that revision.
	Size int64 `json:"size"`
}

// ConfigChangeEvent is emitted when the config store detects an external change.
type ConfigChangeEvent struct {
	// Version is the new content hash after the change.
	Version string

	// Error is set if the watcher encountered an error reading the config.
	Error error
}

// ConfigStore abstracts persistent storage for catalog source configuration.
// Implementations must be safe for concurrent use by multiple goroutines.
type ConfigStore interface {
	// Load reads the current configuration and returns it along with a version
	// string (content hash). The version is used for optimistic concurrency on Save.
	Load(ctx context.Context) (*CatalogSourcesConfig, string, error)

	// Save writes the configuration back to storage. The provided version must
	// match the current stored version; otherwise ErrVersionConflict is returned
	// (HTTP 409). On success the new version hash is returned.
	Save(ctx context.Context, cfg *CatalogSourcesConfig, version string) (string, error)

	// Watch returns a channel that emits events when the underlying config
	// changes externally (e.g., file edited on disk). The channel is closed
	// when the context is cancelled. Implementations that do not support
	// watching may return a nil channel and nil error.
	Watch(ctx context.Context) (<-chan ConfigChangeEvent, error)

	// ListRevisions returns the revision history, sorted newest first.
	// Implementations that do not support revision history may return an
	// empty slice and nil error.
	ListRevisions(ctx context.Context) ([]ConfigRevision, error)

	// Rollback restores the configuration to a previous revision identified
	// by its version hash. It returns the restored config, the new version
	// hash (after re-saving), and any error. The restore is performed via
	// Save with the current version for concurrency safety.
	Rollback(ctx context.Context, version string) (*CatalogSourcesConfig, string, error)
}

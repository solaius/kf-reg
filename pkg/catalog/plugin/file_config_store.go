package plugin

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// FileConfigStore implements ConfigStore backed by a YAML file on disk.
// It uses SHA-256 content hashing for optimistic concurrency control and
// atomic writes (write-to-temp + rename) to avoid partial writes.
type FileConfigStore struct {
	path    string
	mu      sync.Mutex
	version string // cached SHA-256 hex digest of file content
}

// NewFileConfigStore creates a FileConfigStore for the given file path.
// The file does not need to exist yet; Load will return an error if it is missing.
func NewFileConfigStore(path string) *FileConfigStore {
	return &FileConfigStore{path: path}
}

// Path returns the file path managed by this store.
func (s *FileConfigStore) Path() string {
	return s.path
}

// Load reads the YAML config file, parses it, and returns the config together
// with a version string (SHA-256 hex digest of the raw file bytes).
func (s *FileConfigStore) Load(_ context.Context) (*CatalogSourcesConfig, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, "", fmt.Errorf("config store: failed to read %s: %w", s.path, err)
	}

	version := hashBytes(data)
	s.version = version

	cfg, err := ParseConfig(data, s.path)
	if err != nil {
		return nil, "", fmt.Errorf("config store: failed to parse %s: %w", s.path, err)
	}

	return cfg, version, nil
}

// Save marshals the config to YAML and writes it atomically to the file.
// The provided version must match the current file hash; otherwise
// ErrVersionConflict is returned. On success the new version hash is returned.
func (s *FileConfigStore) Save(_ context.Context, cfg *CatalogSourcesConfig, version string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Re-read the current file to get the latest hash for comparison.
	currentData, err := os.ReadFile(s.path)
	if err != nil {
		return "", fmt.Errorf("config store: failed to read current file for version check: %w", err)
	}

	currentVersion := hashBytes(currentData)
	if currentVersion != version {
		return "", ErrVersionConflict
	}

	// Marshal the config to YAML.
	data, err := marshalConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("config store: failed to marshal config: %w", err)
	}

	// Atomic write: write to temp file in the same directory, fsync, then rename.
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".sources-*.yaml.tmp")
	if err != nil {
		return "", fmt.Errorf("config store: failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Clean up the temp file on any error path.
	defer func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("config store: failed to write temp file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("config store: failed to sync temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("config store: failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, s.path); err != nil {
		return "", fmt.Errorf("config store: failed to rename temp file: %w", err)
	}
	tmpName = "" // prevent deferred Remove

	newVersion := hashBytes(data)
	s.version = newVersion
	return newVersion, nil
}

// Watch is not implemented for FileConfigStore. Returns nil channel and nil error.
// The server's reconcile loop polls Load() periodically instead.
func (s *FileConfigStore) Watch(_ context.Context) (<-chan ConfigChangeEvent, error) {
	return nil, nil
}

// marshalConfig serializes a CatalogSourcesConfig to YAML bytes.
// Origin fields are excluded via the yaml:"-" tag on SourceConfig.
func marshalConfig(cfg *CatalogSourcesConfig) ([]byte, error) {
	return yaml.Marshal(cfg)
}

// hashBytes returns the SHA-256 hex digest of the given byte slice.
func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// Compile-time interface check.
var _ ConfigStore = (*FileConfigStore)(nil)

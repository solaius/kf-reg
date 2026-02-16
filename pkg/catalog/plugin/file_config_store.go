package plugin

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// maxConfigFileSize is the maximum allowed config file size (1 MiB).
	maxConfigFileSize = 1 << 20

	// maxRevisionHistory is the maximum number of revision snapshots to keep.
	maxRevisionHistory = 20

	// historyDirName is the name of the directory used for revision snapshots.
	historyDirName = ".history"
)

// ErrFileTooLarge is returned when a config file exceeds maxConfigFileSize.
var ErrFileTooLarge = errors.New("config file exceeds maximum allowed size (1 MiB)")

// ErrPathTraversal is returned when a config file path contains path traversal.
var ErrPathTraversal = errors.New("config file path contains path traversal")

// ErrRevisionNotFound is returned when a rollback target version is not found.
var ErrRevisionNotFound = errors.New("revision not found")

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
// Returns an error if the path contains path traversal sequences.
func NewFileConfigStore(path string) (*FileConfigStore, error) {
	if err := validateConfigPath(path); err != nil {
		return nil, err
	}
	return &FileConfigStore{path: path}, nil
}

// validateConfigPath checks that the path does not contain ".." traversal components.
func validateConfigPath(path string) error {
	// Clean and split the path into components; reject any ".." segment.
	cleaned := filepath.Clean(path)
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == ".." {
			return ErrPathTraversal
		}
	}
	return nil
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

	if int64(len(data)) > maxConfigFileSize {
		return nil, "", fmt.Errorf("config store: %s: %w", s.path, ErrFileTooLarge)
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
// Before writing, the current file is snapshotted to .history/ for revision tracking.
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

	if int64(len(data)) > maxConfigFileSize {
		return "", fmt.Errorf("config store: marshaled config: %w", ErrFileTooLarge)
	}

	// Snapshot current file to .history/ before overwriting.
	if err := s.snapshotCurrent(currentData, currentVersion); err != nil {
		// Log but do not fail the save; history is best-effort.
		// The caller's logger isn't available here, so we just swallow the error.
		_ = err
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

	// Prune old history entries (best-effort).
	_ = s.pruneHistory()

	return newVersion, nil
}

// Watch is not implemented for FileConfigStore. Returns nil channel and nil error.
// The server's reconcile loop polls Load() periodically instead.
func (s *FileConfigStore) Watch(_ context.Context) (<-chan ConfigChangeEvent, error) {
	return nil, nil
}

// ListRevisions returns the revision history by scanning the .history/ directory.
// Revisions are returned sorted newest first.
func (s *FileConfigStore) ListRevisions(_ context.Context) ([]ConfigRevision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	histDir := s.historyDir()
	entries, err := os.ReadDir(histDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ConfigRevision{}, nil
		}
		return nil, fmt.Errorf("config store: failed to read history dir: %w", err)
	}

	revisions := make([]ConfigRevision, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}

		rev, err := parseRevisionFilename(e)
		if err != nil {
			continue // skip unparseable filenames
		}
		revisions = append(revisions, rev)
	}

	// Sort newest first.
	sort.Slice(revisions, func(i, j int) bool {
		return revisions[i].Timestamp.After(revisions[j].Timestamp)
	})

	return revisions, nil
}

// Rollback restores the configuration to a previous revision identified by
// its version hash. The revision file is read from .history/, validated,
// and then saved via the normal Save path for concurrency safety.
func (s *FileConfigStore) Rollback(ctx context.Context, version string) (*CatalogSourcesConfig, string, error) {
	// First, find the matching revision file (without holding the lock
	// for the file read, since Save will re-acquire the lock).
	s.mu.Lock()

	histDir := s.historyDir()
	entries, err := os.ReadDir(histDir)
	if err != nil {
		s.mu.Unlock()
		return nil, "", fmt.Errorf("config store: failed to read history dir: %w", err)
	}

	var revisionFile string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		// Match the version prefix in the filename: {timestamp}_{version}.yaml
		parts := strings.SplitN(strings.TrimSuffix(e.Name(), ".yaml"), "_", 2)
		if len(parts) == 2 && parts[1] == version[:min(len(version), 8)] {
			revisionFile = filepath.Join(histDir, e.Name())
			break
		}
	}

	if revisionFile == "" {
		s.mu.Unlock()
		return nil, "", ErrRevisionNotFound
	}

	// Read and validate the revision file.
	revData, err := os.ReadFile(revisionFile)
	if err != nil {
		s.mu.Unlock()
		return nil, "", fmt.Errorf("config store: failed to read revision file: %w", err)
	}

	cfg, err := ParseConfig(revData, s.path)
	if err != nil {
		s.mu.Unlock()
		return nil, "", fmt.Errorf("config store: revision file is invalid: %w", err)
	}

	// Get current version for concurrency check, then release lock
	// so Save can re-acquire it.
	currentData, err := os.ReadFile(s.path)
	if err != nil {
		s.mu.Unlock()
		return nil, "", fmt.Errorf("config store: failed to read current file: %w", err)
	}
	currentVersion := hashBytes(currentData)
	s.mu.Unlock()

	// Save the restored config through the normal Save path.
	newVersion, err := s.Save(ctx, cfg, currentVersion)
	if err != nil {
		return nil, "", fmt.Errorf("config store: failed to save rolled-back config: %w", err)
	}

	return cfg, newVersion, nil
}

// historyDir returns the path to the .history/ directory next to the config file.
func (s *FileConfigStore) historyDir() string {
	return filepath.Join(filepath.Dir(s.path), historyDirName)
}

// snapshotCurrent copies the current file content to .history/{timestamp}_{version_short}.yaml.
// Must be called with s.mu held.
func (s *FileConfigStore) snapshotCurrent(data []byte, version string) error {
	histDir := s.historyDir()
	if err := os.MkdirAll(histDir, 0o755); err != nil {
		return fmt.Errorf("config store: failed to create history dir: %w", err)
	}

	versionShort := version
	if len(versionShort) > 8 {
		versionShort = versionShort[:8]
	}

	filename := fmt.Sprintf("%d_%s.yaml", time.Now().Unix(), versionShort)
	histPath := filepath.Join(histDir, filename)

	if err := os.WriteFile(histPath, data, 0o644); err != nil {
		return fmt.Errorf("config store: failed to write history snapshot: %w", err)
	}

	return nil
}

// pruneHistory removes old revision files, keeping only the most recent maxRevisionHistory.
// Must be called with s.mu held.
func (s *FileConfigStore) pruneHistory() error {
	histDir := s.historyDir()
	entries, err := os.ReadDir(histDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Filter to only .yaml files.
	var yamlFiles []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			yamlFiles = append(yamlFiles, e)
		}
	}

	if len(yamlFiles) <= maxRevisionHistory {
		return nil
	}

	// Sort by name (which starts with unix timestamp) ascending.
	sort.Slice(yamlFiles, func(i, j int) bool {
		return yamlFiles[i].Name() < yamlFiles[j].Name()
	})

	// Remove oldest entries.
	toRemove := len(yamlFiles) - maxRevisionHistory
	for i := 0; i < toRemove; i++ {
		_ = os.Remove(filepath.Join(histDir, yamlFiles[i].Name()))
	}

	return nil
}

// parseRevisionFilename extracts a ConfigRevision from a history directory entry.
// Expected filename format: {unix_timestamp}_{version_short}.yaml
func parseRevisionFilename(e os.DirEntry) (ConfigRevision, error) {
	name := strings.TrimSuffix(e.Name(), ".yaml")
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return ConfigRevision{}, fmt.Errorf("unexpected filename format: %s", e.Name())
	}

	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return ConfigRevision{}, fmt.Errorf("invalid timestamp in filename: %s", e.Name())
	}

	info, err := e.Info()
	if err != nil {
		return ConfigRevision{}, fmt.Errorf("failed to stat history file: %w", err)
	}

	return ConfigRevision{
		Version:   parts[1],
		Timestamp: time.Unix(ts, 0),
		Size:      info.Size(),
	}, nil
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

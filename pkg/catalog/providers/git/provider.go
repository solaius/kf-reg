// Package git provides a Git repository-based data provider for catalog data.
// It clones/fetches a Git repo, walks path glob patterns to discover YAML files,
// parses each file using a caller-supplied Parse callback, and tracks commit SHA
// for provenance. It supports shallow clones and periodic sync.
package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gogithttp "github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/kubeflow/model-registry/pkg/catalog"
)

// Config configures a Git provider.
type Config[E any, A any] struct {
	// RepoURLKey is the property key for the Git repository URL.
	// Defaults to "repoUrl" if empty.
	RepoURLKey string

	// BranchKey is the property key for the branch name.
	// Defaults to "branch" if empty.
	BranchKey string

	// PathKey is the property key for the file glob pattern.
	// Defaults to "path" if empty.
	PathKey string

	// AuthTokenKey is the property key for the auth token.
	// Defaults to "authToken" if empty.
	AuthTokenKey string

	// SyncIntervalKey is the property key for the sync interval.
	// Defaults to "syncInterval" if empty.
	SyncIntervalKey string

	// Parse parses raw YAML bytes into a slice of entity records.
	Parse func(data []byte) ([]catalog.Record[E, A], error)

	// Filter optionally filters records before emitting them.
	Filter func(record catalog.Record[E, A]) bool

	// Logger for logging messages (optional).
	Logger Logger

	// DefaultBranch is the default branch if none is specified.
	// Defaults to "main".
	DefaultBranch string

	// DefaultSyncInterval is the default sync interval.
	// Defaults to 1 hour.
	DefaultSyncInterval time.Duration

	// ShallowClone controls whether to use depth=1 clones.
	// Defaults to true.
	ShallowClone *bool
}

// Logger is an interface for logging.
type Logger interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Infof(format string, args ...any)  {}
func (noopLogger) Errorf(format string, args ...any) {}

// Provider is a Git repository-based data provider.
type Provider[E any, A any] struct {
	config       Config[E, A]
	repoURL      string
	branch       string
	pathPattern  string
	authToken    string
	syncInterval time.Duration
	shallowClone bool
	logger       Logger
	cloneDir     string
	lastCommit   string
}

// NewProvider creates a new Git provider.
func NewProvider[E any, A any](config Config[E, A], source *catalog.Source, reldir string) (*Provider[E, A], error) {
	repoURLKey := config.RepoURLKey
	if repoURLKey == "" {
		repoURLKey = "repoUrl"
	}
	repoURL, ok := source.Properties[repoURLKey].(string)
	if !ok || repoURL == "" {
		return nil, fmt.Errorf("missing %s string property", repoURLKey)
	}

	branchKey := config.BranchKey
	if branchKey == "" {
		branchKey = "branch"
	}
	branch := config.DefaultBranch
	if branch == "" {
		branch = "main"
	}
	if b, ok := source.Properties[branchKey].(string); ok && b != "" {
		branch = b
	}

	pathKey := config.PathKey
	if pathKey == "" {
		pathKey = "path"
	}
	pathPattern, _ := source.Properties[pathKey].(string)
	if pathPattern == "" {
		pathPattern = "**/*.yaml"
	}

	authTokenKey := config.AuthTokenKey
	if authTokenKey == "" {
		authTokenKey = "authToken"
	}
	authToken, _ := source.Properties[authTokenKey].(string)

	syncIntervalKey := config.SyncIntervalKey
	if syncIntervalKey == "" {
		syncIntervalKey = "syncInterval"
	}
	syncInterval := config.DefaultSyncInterval
	if syncInterval == 0 {
		syncInterval = 1 * time.Hour
	}
	if s, ok := source.Properties[syncIntervalKey].(string); ok && s != "" {
		if parsed, err := time.ParseDuration(s); err == nil {
			syncInterval = parsed
		}
	}

	shallowClone := true
	if config.ShallowClone != nil {
		shallowClone = *config.ShallowClone
	}
	if v, ok := source.Properties["shallowClone"].(bool); ok {
		shallowClone = v
	}

	logger := config.Logger
	if logger == nil {
		logger = noopLogger{}
	}

	return &Provider[E, A]{
		config:       config,
		repoURL:      repoURL,
		branch:       branch,
		pathPattern:  pathPattern,
		authToken:    authToken,
		syncInterval: syncInterval,
		shallowClone: shallowClone,
		logger:       logger,
	}, nil
}

// Records starts reading the Git repo and returns a channel of records.
// The channel is closed when the context is canceled.
func (p *Provider[E, A]) Records(ctx context.Context) (<-chan catalog.Record[E, A], error) {
	// Clone the repo initially.
	records, err := p.cloneAndRead()
	if err != nil {
		return nil, fmt.Errorf("initial clone failed: %w", err)
	}

	ch := make(chan catalog.Record[E, A])
	go func() {
		defer close(ch)
		defer p.cleanup()

		// Send initial records.
		p.emit(ctx, records, ch)

		// Periodically fetch and re-emit if HEAD changed.
		p.watchAndReload(ctx, ch)
	}()

	return ch, nil
}

// LastCommit returns the SHA of the last fetched commit.
func (p *Provider[E, A]) LastCommit() string {
	return p.lastCommit
}

// RepoURL returns the Git repository URL configured for this provider.
func (p *Provider[E, A]) RepoURL() string {
	return p.repoURL
}

// Branch returns the branch name configured for this provider.
func (p *Provider[E, A]) Branch() string {
	return p.branch
}

func (p *Provider[E, A]) cloneAndRead() ([]catalog.Record[E, A], error) {
	// Create temp directory for the clone.
	dir, err := os.MkdirTemp("", "catalog-git-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	p.cloneDir = dir

	cloneOpts := &gogit.CloneOptions{
		URL:           p.repoURL,
		ReferenceName: plumbing.NewBranchReferenceName(p.branch),
		SingleBranch:  true,
	}
	if p.shallowClone {
		cloneOpts.Depth = 1
	}
	if p.authToken != "" {
		cloneOpts.Auth = &gogithttp.BasicAuth{
			Username: "git", // Username is ignored for token auth.
			Password: p.authToken,
		}
	}

	p.logger.Infof("Cloning %s (branch: %s) into %s", p.repoURL, p.branch, dir)
	repo, err := gogit.PlainClone(dir, false, cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("git clone failed for %s: %w", p.repoURL, err)
	}

	// Track HEAD commit SHA.
	if err := p.updateLastCommit(repo); err != nil {
		p.logger.Errorf("Failed to get HEAD commit: %v", err)
	}

	return p.readFiles()
}

func (p *Provider[E, A]) fetchAndRead() ([]catalog.Record[E, A], bool, error) {
	repo, err := gogit.PlainOpen(p.cloneDir)
	if err != nil {
		return nil, false, fmt.Errorf("failed to open repo: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get worktree: %w", err)
	}

	pullOpts := &gogit.PullOptions{
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(p.branch),
		SingleBranch:  true,
	}
	if p.authToken != "" {
		pullOpts.Auth = &gogithttp.BasicAuth{
			Username: "git",
			Password: p.authToken,
		}
	}

	err = w.Pull(pullOpts)
	if err == gogit.NoErrAlreadyUpToDate {
		return nil, false, nil // No changes.
	}
	if err != nil {
		return nil, false, fmt.Errorf("git pull failed: %w", err)
	}

	// Check if commit changed.
	oldCommit := p.lastCommit
	if err := p.updateLastCommit(repo); err != nil {
		p.logger.Errorf("Failed to get HEAD commit: %v", err)
	}

	if p.lastCommit == oldCommit {
		return nil, false, nil
	}

	records, err := p.readFiles()
	if err != nil {
		return nil, false, err
	}
	return records, true, nil
}

func (p *Provider[E, A]) readFiles() ([]catalog.Record[E, A], error) {
	files, err := p.globFiles()
	if err != nil {
		return nil, err
	}

	var allRecords []catalog.Record[E, A]
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			p.logger.Errorf("Failed to read %s: %v", file, err)
			continue
		}

		records, err := p.config.Parse(data)
		if err != nil {
			p.logger.Errorf("Failed to parse %s: %v", file, err)
			continue
		}

		allRecords = append(allRecords, records...)
	}

	p.logger.Infof("Read %d records from %d files in %s", len(allRecords), len(files), p.repoURL)
	return allRecords, nil
}

func (p *Provider[E, A]) globFiles() ([]string, error) {
	var matches []string

	err := filepath.Walk(p.cloneDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip .git directory.
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path from clone dir.
		relPath, err := filepath.Rel(p.cloneDir, path)
		if err != nil {
			return nil
		}
		// Normalize to forward slashes for pattern matching.
		relPath = filepath.ToSlash(relPath)

		if matchGlob(p.pathPattern, relPath) {
			matches = append(matches, path)
		}
		return nil
	})

	return matches, err
}

func (p *Provider[E, A]) updateLastCommit(repo *gogit.Repository) error {
	ref, err := repo.Head()
	if err != nil {
		return err
	}
	p.lastCommit = ref.Hash().String()
	return nil
}

func (p *Provider[E, A]) emit(ctx context.Context, records []catalog.Record[E, A], out chan<- catalog.Record[E, A]) {
	done := ctx.Done()
	for _, record := range records {
		if p.config.Filter != nil && !p.config.Filter(record) {
			continue
		}
		select {
		case out <- record:
		case <-done:
			return
		}
	}

	// Send an empty record to indicate batch completion.
	var zero catalog.Record[E, A]
	select {
	case out <- zero:
	case <-done:
	}
}

func (p *Provider[E, A]) watchAndReload(ctx context.Context, ch chan<- catalog.Record[E, A]) {
	ticker := time.NewTicker(p.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.logger.Infof("Checking for updates in %s", p.repoURL)
			records, changed, err := p.fetchAndRead()
			if err != nil {
				p.logger.Errorf("Failed to fetch updates: %v", err)
				continue
			}
			if !changed {
				continue
			}
			p.logger.Infof("New commits detected (HEAD: %s), reloading", p.lastCommit)
			p.emit(ctx, records, ch)
		}
	}
}

func (p *Provider[E, A]) cleanup() {
	if p.cloneDir != "" {
		os.RemoveAll(p.cloneDir)
	}
}

// NewProviderFunc creates a ProviderFunc for the Git provider.
func NewProviderFunc[E any, A any](config Config[E, A]) catalog.ProviderFunc[E, A] {
	return func(ctx context.Context, source *catalog.Source, reldir string) (<-chan catalog.Record[E, A], error) {
		provider, err := NewProvider(config, source, reldir)
		if err != nil {
			return nil, err
		}
		return provider.Records(ctx)
	}
}

// matchGlob matches a path against a glob pattern.
// Supports *, **, and ? wildcards.
func matchGlob(pattern, path string) bool {
	// Handle ** (match any number of directories).
	if strings.Contains(pattern, "**") {
		parts := strings.SplitN(pattern, "**", 2)
		prefix := parts[0]
		suffix := strings.TrimLeft(parts[1], "/")

		if prefix != "" && !strings.HasPrefix(path, prefix) {
			return false
		}

		// If no suffix, match everything under prefix.
		if suffix == "" {
			return true
		}

		// Try matching suffix against all possible subpaths.
		trimmed := path
		if prefix != "" {
			trimmed = strings.TrimPrefix(path, prefix)
		}
		pathParts := strings.Split(trimmed, "/")
		for i := range pathParts {
			subpath := strings.Join(pathParts[i:], "/")
			matched, _ := filepath.Match(suffix, subpath)
			if matched {
				return true
			}
		}
		return false
	}

	// Simple glob without **.
	matched, _ := filepath.Match(pattern, path)
	return matched
}

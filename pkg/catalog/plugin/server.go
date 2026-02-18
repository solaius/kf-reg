package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/gorm"

	"github.com/kubeflow/model-registry/pkg/audit"
	"github.com/kubeflow/model-registry/pkg/authz"
	"github.com/kubeflow/model-registry/pkg/cache"
	"github.com/kubeflow/model-registry/pkg/catalog/governance"
	"github.com/kubeflow/model-registry/pkg/ha"
	"github.com/kubeflow/model-registry/pkg/jobs"
	"github.com/kubeflow/model-registry/pkg/tenancy"
)

// failedPlugin records a plugin that failed during initialization.
type failedPlugin struct {
	plugin CatalogPlugin
	err    error
}

// Server manages the lifecycle of catalog plugins and provides a unified HTTP server.
type Server struct {
	router          chi.Router
	db              *gorm.DB
	config          *CatalogSourcesConfig
	configPaths     []string
	logger          *slog.Logger
	plugins         []CatalogPlugin
	failedPlugins   []failedPlugin
	roleExtractor   RoleExtractor
	configStore     ConfigStore
	configVersion   string // hash from last ConfigStore.Load
	rateLimiter     *RefreshRateLimiter
	secretResolver  SecretResolver
	overlayStore     *OverlayStore
	governanceStore  *governance.GovernanceStore
	auditStore       *governance.AuditStore
	governanceConfig *governance.GovernanceConfig
	auditConfig      *audit.AuditConfig
	jobConfig        *jobs.JobConfig
	jobStore         *jobs.JobStore
	jobWorker        *jobs.WorkerPool
	tenancyMode      tenancy.TenancyMode
	authorizer       authz.Authorizer
	cacheManager     *cache.CacheManager
	migrationLocker  ha.MigrationLocker
	leaderElector    *ha.LeaderElector
	startedAt        time.Time
	initialLoadDone  bool
	mu               sync.RWMutex
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// WithRoleExtractor sets a custom role extractor for RBAC middleware.
func WithRoleExtractor(extractor RoleExtractor) ServerOption {
	return func(s *Server) {
		s.roleExtractor = extractor
	}
}

// WithConfigStore sets a ConfigStore for persistent config management.
// When set, the server will load config from the store during Init and
// management handlers will persist mutations back to the store.
func WithConfigStore(store ConfigStore) ServerOption {
	return func(s *Server) {
		s.configStore = store
	}
}

// WithSecretResolver sets a SecretResolver for resolving SecretRef values
// in source properties. When set, management handlers resolve SecretRef
// objects before passing properties to plugin operations.
func WithSecretResolver(resolver SecretResolver) ServerOption {
	return func(s *Server) {
		s.secretResolver = resolver
	}
}

// WithTenancyMode sets the tenancy mode for the server.
// Defaults to ModeSingle if not set.
func WithTenancyMode(mode tenancy.TenancyMode) ServerOption {
	return func(s *Server) {
		s.tenancyMode = mode
	}
}

// WithAuthorizer sets the Authorizer for SAR-based authorization.
// When set, management routes use fine-grained permission checks instead of
// the simple RoleExtractor-based model.
func WithAuthorizer(a authz.Authorizer) ServerOption {
	return func(s *Server) {
		s.authorizer = a
	}
}

// WithCacheConfig sets up the CacheManager for caching discovery and
// capabilities endpoints. If the config is nil or disabled, no caching is applied.
func WithCacheConfig(cfg *cache.CacheConfig) ServerOption {
	return func(s *Server) {
		s.cacheManager = cache.NewCacheManager(cfg)
	}
}

// WithMigrationLocker sets the MigrationLocker used to serialize database
// migrations across multiple replicas. If not set, migrations run without
// locking (safe for single-replica deployments).
func WithMigrationLocker(locker ha.MigrationLocker) ServerOption {
	return func(s *Server) {
		s.migrationLocker = locker
	}
}

// WithLeaderElector sets the LeaderElector for gating singleton background
// workers. Only the leader replica runs reconcile loops, audit cleanup, and
// other periodic tasks.
func WithLeaderElector(le *ha.LeaderElector) ServerOption {
	return func(s *Server) {
		s.leaderElector = le
	}
}

// IsLeader returns true if this server instance is the current leader.
// Returns true when leader election is not configured (single-replica mode).
func (s *Server) IsLeader() bool {
	if s.leaderElector == nil {
		return true
	}
	return s.leaderElector.IsLeader()
}

// WithGovernanceConfig sets the governance configuration for the server.
func WithGovernanceConfig(cfg *governance.GovernanceConfig) ServerOption {
	return func(s *Server) {
		s.governanceConfig = cfg
	}
}

// WithAuditConfig sets the audit V2 configuration for the server.
func WithAuditConfig(cfg *audit.AuditConfig) ServerOption {
	return func(s *Server) {
		s.auditConfig = cfg
	}
}

// WithJobConfig sets the async job queue configuration for the server.
func WithJobConfig(cfg *jobs.JobConfig) ServerOption {
	return func(s *Server) {
		s.jobConfig = cfg
	}
}

// NewServer creates a new plugin server.
func NewServer(cfg *CatalogSourcesConfig, configPaths []string, db *gorm.DB, logger *slog.Logger, opts ...ServerOption) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	s := &Server{
		db:            db,
		config:        cfg,
		configPaths:   configPaths,
		logger:        logger,
		plugins:       make([]CatalogPlugin, 0),
		failedPlugins: make([]failedPlugin, 0),
		roleExtractor: DefaultRoleExtractor,
		rateLimiter:   NewRefreshRateLimiter(30 * time.Second),
		startedAt:     time.Now(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Init initializes all registered plugins that have configuration.
// If a ConfigStore is set, the config is loaded from it (overriding the
// config passed to NewServer).
func (s *Server) Init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load config from store if available.
	if s.configStore != nil {
		cfg, version, err := s.configStore.Load(ctx)
		if err != nil {
			s.logger.Error("failed to load config from store, using initial config", "error", err)
		} else {
			s.config = cfg
			s.configVersion = version
			s.logger.Info("loaded config from store", "version", version)
		}
	}

	// Auto-migrate all tables. When a MigrationLocker is configured, all
	// migrations run under the lock to prevent concurrent schema changes
	// from multiple replicas.
	if s.db != nil {
		migrateFn := func() error {
			if err := s.db.AutoMigrate(&RefreshStatusRecord{}); err != nil {
				s.logger.Error("failed to auto-migrate refresh status table", "error", err)
			}

			s.overlayStore = NewOverlayStore(s.db)
			if err := s.overlayStore.AutoMigrate(); err != nil {
				s.logger.Error("failed to auto-migrate overlay store table", "error", err)
			}

			s.governanceStore = governance.NewGovernanceStore(s.db)
			s.auditStore = governance.NewAuditStore(s.db)
			if err := s.governanceStore.AutoMigrate(); err != nil {
				s.logger.Error("failed to auto-migrate governance tables", "error", err)
			}

			vStore := governance.NewVersionStore(s.db)
			if err := vStore.AutoMigrate(); err != nil {
				s.logger.Error("failed to auto-migrate version tables", "error", err)
			}
			bStore := governance.NewBindingStore(s.db)
			if err := bStore.AutoMigrate(); err != nil {
				s.logger.Error("failed to auto-migrate binding tables", "error", err)
			}

			if s.jobConfig != nil && s.jobConfig.Enabled {
				s.jobStore = jobs.NewJobStore(s.db)
				if err := s.jobStore.AutoMigrate(); err != nil {
					s.logger.Error("failed to auto-migrate job tables", "error", err)
				}
			}

			return nil
		}

		if s.migrationLocker != nil {
			s.logger.Info("running migrations with lock")
			if err := s.migrationLocker.WithLock(ctx, migrateFn); err != nil {
				s.logger.Error("migration lock error", "error", err)
				return fmt.Errorf("migration lock error: %w", err)
			}
		} else {
			_ = migrateFn()
		}
	}

	for _, p := range All() {
		// Use SourceKey if the plugin provides one, otherwise fall back to plugin name
		configKey := p.Name()
		if skp, ok := p.(SourceKeyProvider); ok {
			configKey = skp.SourceKey()
		}

		section, ok := s.config.Catalogs[configKey]
		if !ok {
			s.logger.Info("plugin has no sources configured", "plugin", p.Name(), "configKey", configKey)
			section = CatalogSection{}
		}

		// Use plugin's BasePath if it implements BasePathProvider, otherwise compute it.
		var basePath string
		if bp, ok := p.(BasePathProvider); ok {
			basePath = bp.BasePath()
		} else {
			basePath = fmt.Sprintf("/api/%s_catalog/%s", p.Name(), p.Version())
		}

		// Only pass config paths to plugins that have sources configured.
		// Unconfigured plugins should not try to parse the server config file.
		var configPaths []string
		if ok {
			configPaths = s.configPaths
		}

		pluginCfg := Config{
			Section:     section,
			DB:          s.db,
			Logger:      s.logger.With("plugin", p.Name()),
			BasePath:    basePath,
			ConfigPaths: configPaths,
		}

		s.logger.Info("initializing plugin", "plugin", p.Name(), "version", p.Version(), "basePath", basePath)

		if err := p.Init(ctx, pluginCfg); err != nil {
			s.logger.Error("plugin init failed, continuing with remaining plugins", "plugin", p.Name(), "error", err)
			s.failedPlugins = append(s.failedPlugins, failedPlugin{plugin: p, err: err})
			continue
		}

		s.plugins = append(s.plugins, p)
	}

	// Mark initial load as done so /readyz reports ready.
	s.initialLoadDone = true

	return nil
}

// MountRoutes creates the HTTP router with all plugin routes mounted.
func (s *Server) MountRoutes() chi.Router {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.router = chi.NewRouter()

	// Add common middleware
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Recoverer)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-PINGOTHER", tenancy.NamespaceHeader},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Add tenancy middleware (resolves tenant context per request).
	s.router.Use(tenancy.NewMiddleware(s.tenancyMode))

	// Add identity middleware (extracts X-Remote-User/X-Remote-Group into context).
	s.router.Use(authz.IdentityMiddleware())

	// Add audit middleware (captures management actions as audit events).
	if s.auditStore != nil && s.auditConfig != nil && s.auditConfig.Enabled {
		s.router.Use(audit.AuditMiddleware(s.auditStore, s.auditConfig, s.logger))
		s.logger.Info("audit middleware enabled",
			"logDenied", s.auditConfig.LogDenied,
			"retentionDays", s.auditConfig.RetentionDays)
	}

	// Mount plugin routes
	for _, p := range s.plugins {
		var basePath string
		if bp, ok := p.(BasePathProvider); ok {
			basePath = bp.BasePath()
		} else {
			basePath = fmt.Sprintf("/api/%s_catalog/%s", p.Name(), p.Version())
		}
		s.logger.Info("mounting plugin routes", "plugin", p.Name(), "basePath", basePath)

		s.router.Route(basePath, func(r chi.Router) {
			if err := p.RegisterRoutes(r); err != nil {
				s.logger.Error("failed to register routes", "plugin", p.Name(), "error", err)
			}

			// Mount management routes if the plugin implements any management interfaces.
			// Management routes are mounted under /management to avoid collisions with
			// plugin-native routes (e.g., model plugin's own GET /sources).
			_, hasSM := p.(SourceManager)
			_, hasRP := p.(RefreshProvider)
			_, hasDP := p.(DiagnosticsProvider)
			_, hasAP := p.(ActionProvider)
			if hasSM || hasRP || hasDP || hasAP {
				mgmtRouter := managementRouter(p, s.roleExtractor, s, s.authorizer)
				r.Mount("/management", mgmtRouter)
				s.logger.Info("mounted management routes", "plugin", p.Name(),
					"sourceManager", hasSM, "refresh", hasRP, "diagnostics", hasDP, "actions", hasAP)
			}
		})
	}

	// Add health endpoints
	s.router.Get("/healthz", s.healthHandler)
	s.router.Get("/livez", s.healthHandler)
	s.router.Get("/readyz", s.readyHandler)

	// Add plugin info endpoint, optionally wrapped with cache middleware.
	if s.cacheManager != nil {
		s.router.With(s.cacheManager.DiscoveryMiddleware()).Get("/api/plugins", s.pluginsHandler)
		s.router.With(s.cacheManager.CapabilitiesMiddleware()).Get("/api/plugins/{pluginName}/capabilities", s.capabilitiesHandler)
		s.logger.Info("discovery/capabilities caching enabled")
	} else {
		s.router.Get("/api/plugins", s.pluginsHandler)
		s.router.Get("/api/plugins/{pluginName}/capabilities", s.capabilitiesHandler)
	}

	// Add tenancy API endpoint
	s.router.Get("/api/tenancy/v1alpha1/namespaces", s.namespacesHandler)

	// Mount governance routes.
	if s.governanceStore != nil {
		var approvalStore *governance.ApprovalStore
		var evaluator *governance.ApprovalEvaluator
		var versionStore *governance.VersionStore
		var bindingStore *governance.BindingStore

		if s.db != nil {
			approvalStore = governance.NewApprovalStore(s.db)
			if err := approvalStore.AutoMigrate(); err != nil {
				s.logger.Error("failed to auto-migrate approval tables", "error", err)
			}
			versionStore = governance.NewVersionStore(s.db)
			bindingStore = governance.NewBindingStore(s.db)
		}

		// Load approval policies from YAML file.
		approvalPoliciesPath := os.Getenv("CATALOG_APPROVAL_POLICIES")
		if approvalPoliciesPath == "" {
			approvalPoliciesPath = "/config/approval-policies.yaml"
		}
		var err error
		evaluator, err = governance.LoadApprovalPolicies(approvalPoliciesPath)
		if err != nil {
			s.logger.Warn("failed to load approval policies, using empty defaults",
				"path", approvalPoliciesPath, "error", err)
			evaluator = governance.NewApprovalEvaluator(nil)
		}

		govRouter := governance.NewRouterFull(
			s.governanceStore, s.auditStore, approvalStore, evaluator,
			versionStore, bindingStore,
		)
		s.router.Mount("/api/governance/v1alpha1", govRouter)
		s.logger.Info("mounted governance routes",
			"approvals", approvalStore != nil,
			"versioning", versionStore != nil,
		)
	}

	// Mount audit API routes.
	if s.auditStore != nil {
		auditRouter := audit.Router(s.auditStore, s.authorizer)
		s.router.Mount("/api/audit/v1alpha1", auditRouter)
		s.logger.Info("mounted audit API routes")
	}

	// Mount job status API routes.
	if s.jobStore != nil {
		jobRouter := jobs.Router(s.jobStore, s.authorizer)
		s.router.Mount("/api/jobs/v1alpha1", jobRouter)
		s.logger.Info("mounted job API routes")
	}

	return s.router
}

// Start starts all plugins' background operations.
func (s *Server) Start(ctx context.Context) error {
	s.mu.RLock()
	plugins := make([]CatalogPlugin, len(s.plugins))
	copy(plugins, s.plugins)
	s.mu.RUnlock()

	for _, p := range plugins {
		s.logger.Info("starting plugin", "plugin", p.Name())
		if err := p.Start(ctx); err != nil {
			return fmt.Errorf("plugin %s start failed: %w", p.Name(), err)
		}
	}

	s.mu.Lock()
	s.initialLoadDone = true
	s.mu.Unlock()

	return nil
}

// Stop gracefully shuts down all plugins.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var lastErr error
	for _, p := range s.plugins {
		s.logger.Info("stopping plugin", "plugin", p.Name())
		if err := p.Stop(ctx); err != nil {
			s.logger.Error("plugin stop failed", "plugin", p.Name(), "error", err)
			lastErr = err
		}
	}

	return lastErr
}

// Router returns the underlying chi.Router.
func (s *Server) Router() chi.Router {
	return s.router
}

// Plugins returns the list of initialized plugins.
func (s *Server) Plugins() []CatalogPlugin {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]CatalogPlugin, len(s.plugins))
	copy(result, s.plugins)
	return result
}

// healthHandler returns the liveness status of the server.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	uptime := time.Since(s.startedAt).Round(time.Second).String()

	response := map[string]string{
		"status": "alive",
		"uptime": uptime,
	}

	_ = json.NewEncoder(w).Encode(response)
}

// readyHandler checks if all components are ready to serve traffic.
// It verifies DB connectivity, initial load completion, and plugin health.
func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	initialLoadDone := s.initialLoadDone
	plugins := make([]CatalogPlugin, len(s.plugins))
	copy(plugins, s.plugins)
	failedPlugins := make([]failedPlugin, len(s.failedPlugins))
	copy(failedPlugins, s.failedPlugins)
	s.mu.RUnlock()

	allReady := true

	// Check DB connectivity.
	dbStatus := map[string]string{"status": "up"}
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err != nil {
			dbStatus["status"] = "down"
			dbStatus["error"] = err.Error()
			allReady = false
		} else if err := sqlDB.Ping(); err != nil {
			dbStatus["status"] = "down"
			dbStatus["error"] = err.Error()
			allReady = false
		}
	} else {
		dbStatus["status"] = "not_configured"
	}

	// Check initial load completion.
	initialLoadStatus := map[string]string{"status": "complete"}
	if !initialLoadDone {
		initialLoadStatus["status"] = "pending"
		allReady = false
	}

	// Check plugin health.
	healthyCount := 0
	totalCount := len(plugins) + len(failedPlugins)
	for _, p := range plugins {
		if p.Healthy() {
			healthyCount++
		}
	}

	pluginsStatus := map[string]string{
		"status":  "healthy",
		"details": fmt.Sprintf("all %d plugins healthy", totalCount),
	}
	if healthyCount < totalCount {
		pluginsStatus["status"] = "degraded"
		pluginsStatus["details"] = fmt.Sprintf("%d of %d plugins healthy", healthyCount, totalCount)
		allReady = false
	}

	// Report leader election status (informational, does not gate readiness).
	leaderStatus := map[string]string{"status": "not_configured"}
	if s.leaderElector != nil {
		if s.leaderElector.IsLeader() {
			leaderStatus["status"] = "leader"
		} else {
			leaderStatus["status"] = "follower"
		}
	}

	components := map[string]any{
		"database":        dbStatus,
		"initial_load":    initialLoadStatus,
		"plugins":         pluginsStatus,
		"leader_election": leaderStatus,
	}

	status := "ready"
	if !allReady {
		status = "not_ready"
	}

	w.Header().Set("Content-Type", "application/json")

	if allReady {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	response := map[string]any{
		"status":     status,
		"components": components,
	}

	_ = json.NewEncoder(w).Encode(response)
}

// ConfigVersion returns the current config version hash. Empty if no ConfigStore is set.
func (s *Server) ConfigVersion() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configVersion
}

// Config returns a copy of the current config.
func (s *Server) Config() *CatalogSourcesConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// ConfigStore returns the server's ConfigStore, or nil if not set.
func (s *Server) GetConfigStore() ConfigStore {
	return s.configStore
}

// GetSecretResolver returns the server's SecretResolver, or nil if not set.
func (s *Server) GetSecretResolver() SecretResolver {
	return s.secretResolver
}

// GetOverlayStore returns the server's OverlayStore, or nil if no DB is configured.
func (s *Server) GetOverlayStore() *OverlayStore {
	return s.overlayStore
}

// GetGovernanceStore returns the server's GovernanceStore, or nil if no DB is configured.
func (s *Server) GetGovernanceStore() *governance.GovernanceStore {
	return s.governanceStore
}

// GetAuditStore returns the server's AuditStore, or nil if no DB is configured.
func (s *Server) GetAuditStore() *governance.AuditStore {
	return s.auditStore
}

// GetGovernanceConfig returns the server's governance configuration.
func (s *Server) GetGovernanceConfig() *governance.GovernanceConfig {
	return s.governanceConfig
}

// GetAuditConfig returns the server's audit configuration.
func (s *Server) GetAuditConfig() *audit.AuditConfig {
	return s.auditConfig
}

// GetJobStore returns the server's job store (nil if jobs are not enabled).
func (s *Server) GetJobStore() *jobs.JobStore {
	return s.jobStore
}

// GetJobConfig returns the server's job configuration.
func (s *Server) GetJobConfig() *jobs.JobConfig {
	return s.jobConfig
}

// GetCacheManager returns the server's CacheManager, or nil if caching is disabled.
func (s *Server) GetCacheManager() *cache.CacheManager {
	return s.cacheManager
}

// NewJobWorkerPool creates a worker pool for async refresh jobs.
// Call Run(ctx) on the returned pool to start processing.
// Returns nil if jobs are not enabled.
func (s *Server) NewJobWorkerPool() *jobs.WorkerPool {
	if s.jobStore == nil || s.jobConfig == nil || !s.jobConfig.Enabled {
		return nil
	}

	lookup := func(pluginName string) (jobs.PluginRefresher, bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, p := range s.plugins {
			if p.Name() == pluginName {
				rp, ok := p.(RefreshProvider)
				if !ok {
					return nil, false
				}
				return &pluginRefreshAdapter{rp: rp, srv: s, pluginName: pluginName}, true
			}
		}
		return nil, false
	}

	return jobs.NewWorkerPool(s.jobStore, lookup, s.jobConfig, s.logger)
}

// persistConfig saves the current in-memory config to the ConfigStore using
// optimistic concurrency. It updates the in-memory configVersion on success.
// Returns the new version on success, or an error (including ErrVersionConflict).
func (s *Server) persistConfig(ctx context.Context) (string, error) {
	if s.configStore == nil {
		return "", nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	newVersion, err := s.configStore.Save(ctx, s.config, s.configVersion)
	if err != nil {
		return "", err
	}
	s.configVersion = newVersion
	return newVersion, nil
}

// updateConfigSource updates a single source in the in-memory config for a
// given plugin config key. If the source ID exists, it is replaced; otherwise
// it is appended.
func (s *Server) updateConfigSource(configKey string, src SourceConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	section := s.config.Catalogs[configKey]
	found := false
	for i, existing := range section.Sources {
		if existing.ID == src.ID {
			section.Sources[i] = src
			found = true
			break
		}
	}
	if !found {
		section.Sources = append(section.Sources, src)
	}
	s.config.Catalogs[configKey] = section
}

// enableConfigSource updates the enabled state of a source in the in-memory config.
func (s *Server) enableConfigSource(configKey, sourceID string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	section := s.config.Catalogs[configKey]
	for i, src := range section.Sources {
		if src.ID == sourceID {
			section.Sources[i].Enabled = &enabled
			break
		}
	}
	s.config.Catalogs[configKey] = section
}

// CleanupPluginData removes all persisted data associated with a plugin.
// This should be called when a plugin is unregistered or permanently removed.
// Currently it deletes all refresh status records for the plugin. Future
// cleanup steps (e.g., removing config sections) can be added here.
func (s *Server) CleanupPluginData(pluginName string) {
	s.deleteAllRefreshStatuses(pluginName)
	s.logger.Info("cleaned up plugin data", "plugin", pluginName)
}

// deleteConfigSource removes a source from the in-memory config.
func (s *Server) deleteConfigSource(configKey, sourceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	section := s.config.Catalogs[configKey]
	filtered := make([]SourceConfig, 0, len(section.Sources))
	for _, src := range section.Sources {
		if src.ID != sourceID {
			filtered = append(filtered, src)
		}
	}
	section.Sources = filtered
	s.config.Catalogs[configKey] = section
}

// ReconcileLoop periodically checks the ConfigStore for external changes and
// re-initializes affected plugins. It runs until the context is cancelled.
func (s *Server) ReconcileLoop(ctx context.Context) {
	if s.configStore == nil {
		s.logger.Info("no config store set, reconcile loop disabled")
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.logger.Info("config reconcile loop started", "interval", "30s")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("config reconcile loop stopped")
			return
		case <-ticker.C:
			s.reconcileOnce(ctx)
		}
	}
}

// reconcileOnce performs a single reconciliation check.
func (s *Server) reconcileOnce(ctx context.Context) {
	cfg, version, err := s.configStore.Load(ctx)
	if err != nil {
		s.logger.Error("reconcile: failed to load config", "error", err)
		return
	}

	s.mu.RLock()
	currentVersion := s.configVersion
	s.mu.RUnlock()

	if version == currentVersion {
		return
	}

	s.logger.Info("reconcile: config changed externally, updating",
		"oldVersion", currentVersion, "newVersion", version)

	s.mu.Lock()
	s.config = cfg
	s.configVersion = version
	s.mu.Unlock()

	// Re-initialize plugins with the new config.
	// We use a best-effort approach: re-init each plugin individually.
	for _, p := range s.Plugins() {
		configKey := p.Name()
		if skp, ok := p.(SourceKeyProvider); ok {
			configKey = skp.SourceKey()
		}

		section, ok := cfg.Catalogs[configKey]
		if !ok {
			section = CatalogSection{}
		}

		var basePath string
		if bp, ok := p.(BasePathProvider); ok {
			basePath = bp.BasePath()
		} else {
			basePath = fmt.Sprintf("/api/%s_catalog/%s", p.Name(), p.Version())
		}

		var configPaths []string
		if ok {
			s.mu.RLock()
			configPaths = s.configPaths
			s.mu.RUnlock()
		}

		pluginCfg := Config{
			Section:     section,
			DB:          s.db,
			Logger:      s.logger.With("plugin", p.Name()),
			BasePath:    basePath,
			ConfigPaths: configPaths,
		}

		s.logger.Info("reconcile: re-initializing plugin", "plugin", p.Name())
		if err := p.Init(ctx, pluginCfg); err != nil {
			s.logger.Error("reconcile: plugin re-init failed", "plugin", p.Name(), "error", err)
		}
	}

	// Invalidate all caches after reconciliation since plugin state may have changed.
	s.cacheManager.InvalidateAll()
}

// AuditCleanupLoop periodically deletes audit events older than the configured
// retention period. It runs daily until the context is cancelled.
func (s *Server) AuditCleanupLoop(ctx context.Context) {
	if s.auditStore == nil || s.governanceConfig == nil || s.governanceConfig.AuditRetention.Days <= 0 {
		s.logger.Info("audit cleanup loop disabled")
		return
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	s.logger.Info("audit cleanup loop started", "retentionDays", s.governanceConfig.AuditRetention.Days)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("audit cleanup loop stopped")
			return
		case <-ticker.C:
			cutoff := time.Now().AddDate(0, 0, -s.governanceConfig.AuditRetention.Days)
			deleted, err := s.auditStore.DeleteOlderThan(cutoff)
			if err != nil {
				s.logger.Error("audit cleanup failed", "error", err)
			} else if deleted > 0 {
				s.logger.Info("audit cleanup completed", "deleted", deleted, "cutoff", cutoff.Format(time.RFC3339))
			}
		}
	}
}

// pluginConfigKey returns the config key for a given plugin.
func pluginConfigKey(p CatalogPlugin) string {
	if skp, ok := p.(SourceKeyProvider); ok {
		return skp.SourceKey()
	}
	return p.Name()
}

// capabilitiesHandler returns V2 capabilities for a specific plugin.
func (s *Server) capabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "pluginName")

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find the plugin among initialized plugins.
	var found CatalogPlugin
	var basePath string
	for _, p := range s.plugins {
		if p.Name() == pluginName {
			found = p
			if bp, ok := p.(BasePathProvider); ok {
				basePath = bp.BasePath()
			} else {
				basePath = fmt.Sprintf("/api/%s_catalog/%s", p.Name(), p.Version())
			}
			break
		}
	}

	if found == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("plugin %q not found", pluginName),
		})
		return
	}

	v2caps := BuildCapabilitiesV2(found, basePath)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v2caps)
}

// namespacesHandler returns the list of namespaces accessible to the user.
// In single-tenant mode it always returns ["default"].
// In namespace mode it returns a list from the CATALOG_NAMESPACES env var or ["default"].
func (s *Server) namespacesHandler(w http.ResponseWriter, r *http.Request) {
	namespaces := []string{"default"}
	if s.tenancyMode == tenancy.ModeNamespace {
		if ns := os.Getenv("CATALOG_NAMESPACES"); ns != "" {
			parts := strings.Split(ns, ",")
			trimmed := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					trimmed = append(trimmed, p)
				}
			}
			if len(trimmed) > 0 {
				namespaces = trimmed
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"namespaces": namespaces,
		"mode":       string(s.tenancyMode),
	})
}

// pluginsHandler returns information about registered plugins.
func (s *Server) pluginsHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type managementCaps struct {
		SourceManager bool `json:"sourceManager"`
		Refresh       bool `json:"refresh"`
		Diagnostics   bool `json:"diagnostics"`
		Actions       bool `json:"actions"`
	}

	type pluginInfo struct {
		Name           string                `json:"name"`
		Version        string                `json:"version"`
		Description    string                `json:"description"`
		BasePath       string                `json:"basePath"`
		Healthy        bool                  `json:"healthy"`
		EntityKinds    []string              `json:"entityKinds,omitempty"`
		Capabilities   *PluginCapabilities   `json:"capabilities,omitempty"`
		CapabilitiesV2 *PluginCapabilitiesV2 `json:"capabilitiesV2,omitempty"`
		Status         *PluginStatus         `json:"status,omitempty"`
		Management     *managementCaps       `json:"management,omitempty"`
		UIHintsData    *UIHints              `json:"uiHints,omitempty"`
		CLIHintsData   *CLIHints             `json:"cliHints,omitempty"`
	}

	plugins := make([]pluginInfo, 0, len(s.plugins)+len(s.failedPlugins))
	for _, p := range s.plugins {
		var basePath string
		if bp, ok := p.(BasePathProvider); ok {
			basePath = bp.BasePath()
		} else {
			basePath = fmt.Sprintf("/api/%s_catalog/%s", p.Name(), p.Version())
		}
		info := pluginInfo{
			Name:        p.Name(),
			Version:     p.Version(),
			Description: p.Description(),
			BasePath:    basePath,
			Healthy:     p.Healthy(),
		}

		if cp, ok := p.(CapabilitiesProvider); ok {
			caps := cp.Capabilities()
			info.Capabilities = &caps
			info.EntityKinds = caps.EntityKinds
		}
		if sp, ok := p.(StatusProvider); ok {
			status := sp.Status()
			info.Status = &status
		}

		// Report management capabilities
		_, hasSM := p.(SourceManager)
		_, hasRP := p.(RefreshProvider)
		_, hasDP := p.(DiagnosticsProvider)
		_, hasAP := p.(ActionProvider)
		if hasSM || hasRP || hasDP || hasAP {
			info.Management = &managementCaps{
				SourceManager: hasSM,
				Refresh:       hasRP,
				Diagnostics:   hasDP,
				Actions:       hasAP,
			}
		}

		if hp, ok := p.(UIHintsProvider); ok {
			hints := hp.UIHints()
			info.UIHintsData = &hints
		}
		if cp, ok := p.(CLIHintsProvider); ok {
			hints := cp.CLIHints()
			info.CLIHintsData = &hints
		}

		// Build and include V2 capabilities inline.
		v2caps := BuildCapabilitiesV2(p, basePath)
		info.CapabilitiesV2 = &v2caps

		plugins = append(plugins, info)
	}

	for _, fp := range s.failedPlugins {
		var basePath string
		if bp, ok := fp.plugin.(BasePathProvider); ok {
			basePath = bp.BasePath()
		} else {
			basePath = fmt.Sprintf("/api/%s_catalog/%s", fp.plugin.Name(), fp.plugin.Version())
		}
		status := PluginStatus{
			Enabled:     true,
			Initialized: false,
			Serving:     false,
			LastError:   fp.err.Error(),
		}
		plugins = append(plugins, pluginInfo{
			Name:        fp.plugin.Name(),
			Version:     fp.plugin.Version(),
			Description: fp.plugin.Description(),
			BasePath:    basePath,
			Healthy:     false,
			Status:      &status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]any{
		"plugins": plugins,
		"count":   len(plugins),
	}

	_ = json.NewEncoder(w).Encode(response)
}

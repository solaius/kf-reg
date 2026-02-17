package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/gorm"
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
	overlayStore    *OverlayStore
	startedAt       time.Time
	initialLoadDone bool
	mu              sync.RWMutex
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

	// Auto-migrate the refresh status table.
	if s.db != nil {
		if err := s.db.AutoMigrate(&RefreshStatusRecord{}); err != nil {
			s.logger.Error("failed to auto-migrate refresh status table", "error", err)
		}

		// Auto-migrate the overlay store table.
		s.overlayStore = NewOverlayStore(s.db)
		if err := s.overlayStore.AutoMigrate(); err != nil {
			s.logger.Error("failed to auto-migrate overlay store table", "error", err)
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
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-PINGOTHER"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

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

			// Mount management routes if the plugin implements any management interfaces
			_, hasSM := p.(SourceManager)
			_, hasRP := p.(RefreshProvider)
			_, hasDP := p.(DiagnosticsProvider)
			_, hasAP := p.(ActionProvider)
			if hasSM || hasRP || hasDP || hasAP {
				mgmtRouter := managementRouter(p, s.roleExtractor, s)
				r.Mount("/", mgmtRouter)
				s.logger.Info("mounted management routes", "plugin", p.Name(),
					"sourceManager", hasSM, "refresh", hasRP, "diagnostics", hasDP, "actions", hasAP)
			}
		})
	}

	// Add health endpoints
	s.router.Get("/healthz", s.healthHandler)
	s.router.Get("/livez", s.healthHandler)
	s.router.Get("/readyz", s.readyHandler)

	// Add plugin info endpoint
	s.router.Get("/api/plugins", s.pluginsHandler)
	s.router.Get("/api/plugins/{pluginName}/capabilities", s.capabilitiesHandler)

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

	components := map[string]any{
		"database":     dbStatus,
		"initial_load": initialLoadStatus,
		"plugins":      pluginsStatus,
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

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

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
	router        chi.Router
	db            *gorm.DB
	config        *CatalogSourcesConfig
	configPaths   []string
	logger        *slog.Logger
	plugins       []CatalogPlugin
	failedPlugins []failedPlugin
	mu            sync.RWMutex
}

// NewServer creates a new plugin server.
func NewServer(cfg *CatalogSourcesConfig, configPaths []string, db *gorm.DB, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		db:            db,
		config:        cfg,
		configPaths:   configPaths,
		logger:        logger,
		plugins:       make([]CatalogPlugin, 0),
		failedPlugins: make([]failedPlugin, 0),
	}
}

// Init initializes all registered plugins that have configuration.
func (s *Server) Init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
		})
	}

	// Add health endpoint
	s.router.Get("/healthz", s.healthHandler)
	s.router.Get("/readyz", s.readyHandler)

	// Add plugin info endpoint
	s.router.Get("/api/plugins", s.pluginsHandler)

	return s.router
}

// Start starts all plugins' background operations.
func (s *Server) Start(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.plugins {
		s.logger.Info("starting plugin", "plugin", p.Name())
		if err := p.Start(ctx); err != nil {
			return fmt.Errorf("plugin %s start failed: %w", p.Name(), err)
		}
	}

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

// healthHandler returns the health status of the server.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]string{
		"status": "ok",
	}

	_ = json.NewEncoder(w).Encode(response)
}

// readyHandler checks if all plugins are healthy.
func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	allHealthy := true
	pluginStatus := make(map[string]bool)

	for _, p := range s.plugins {
		healthy := p.Healthy()
		pluginStatus[p.Name()] = healthy
		if !healthy {
			allHealthy = false
		}
	}

	for _, fp := range s.failedPlugins {
		pluginStatus[fp.plugin.Name()] = false
		allHealthy = false
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]any{
		"plugins": pluginStatus,
	}

	if allHealthy {
		response["status"] = "ready"
		w.WriteHeader(http.StatusOK)
	} else {
		response["status"] = "not_ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	_ = json.NewEncoder(w).Encode(response)
}

// pluginsHandler returns information about registered plugins.
func (s *Server) pluginsHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type pluginInfo struct {
		Name         string              `json:"name"`
		Version      string              `json:"version"`
		Description  string              `json:"description"`
		BasePath     string              `json:"basePath"`
		Healthy      bool                `json:"healthy"`
		EntityKinds  []string            `json:"entityKinds,omitempty"`
		Capabilities *PluginCapabilities `json:"capabilities,omitempty"`
		Status       *PluginStatus       `json:"status,omitempty"`
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

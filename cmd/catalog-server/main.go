// Package main provides the unified catalog server entry point.
// This server hosts all registered catalog plugins under a single process.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeflow/model-registry/internal/datastore"
	"github.com/kubeflow/model-registry/internal/datastore/embedmd"
	"github.com/kubeflow/model-registry/internal/db"
	"github.com/kubeflow/model-registry/pkg/audit"
	"github.com/kubeflow/model-registry/pkg/authz"
	"github.com/kubeflow/model-registry/pkg/cache"
	"github.com/kubeflow/model-registry/pkg/catalog/governance"
	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
	"github.com/kubeflow/model-registry/pkg/ha"
	"github.com/kubeflow/model-registry/pkg/jobs"
	"github.com/kubeflow/model-registry/pkg/tenancy"

	// Import plugins - their init() registers them
	_ "github.com/kubeflow/model-registry/catalog/plugins/model"
	_ "github.com/kubeflow/model-registry/catalog/plugins/mcp"        // generated via catalog-gen
	_ "github.com/kubeflow/model-registry/catalog/plugins/knowledge"  // knowledge sources plugin
	_ "github.com/kubeflow/model-registry/catalog/plugins/prompts"    // prompt templates plugin
	_ "github.com/kubeflow/model-registry/catalog/plugins/agents"     // agents catalog plugin
	_ "github.com/kubeflow/model-registry/catalog/plugins/guardrails" // guardrails plugin
	_ "github.com/kubeflow/model-registry/catalog/plugins/policies"   // policies plugin
	_ "github.com/kubeflow/model-registry/catalog/plugins/skills"     // skills plugin
)

func main() {
	var (
		listenAddr     string
		sourcesPath    string
		databaseType   string
		databaseDSN    string
		configStoreStr string
	)

	flag.StringVar(&listenAddr, "listen", ":8080", "Address to listen on")
	flag.StringVar(&sourcesPath, "sources", "/config/sources.yaml", "Path to catalog sources config")
	flag.StringVar(&databaseType, "db-type", "postgres", "Database type (postgres or mysql)")
	flag.StringVar(&databaseDSN, "db-dsn", "", "Database connection string")
	flag.StringVar(&configStoreStr, "config-store", "", "Config store backend (file, k8s, or none)")
	flag.Parse()

	// Allow env var override for config store mode.
	if configStoreStr == "" {
		configStoreStr = os.Getenv("CATALOG_CONFIG_STORE_MODE")
	}
	if configStoreStr == "" {
		configStoreStr = "file" // default
	}

	// Initialize glog for backwards compatibility
	_ = flag.Set("logtostderr", "true")

	// Set up structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting catalog server",
		"listen", listenAddr,
		"sources", sourcesPath,
		"plugins", plugin.Names(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	// Load config
	cfg, err := plugin.LoadConfig(sourcesPath)
	if err != nil {
		glog.Fatalf("Failed to load config: %v", err)
	}

	logger.Info("loaded config",
		"apiVersion", cfg.APIVersion,
		"kind", cfg.Kind,
		"catalogs", len(cfg.Catalogs),
	)

	// Setup database
	gormDB, err := setupDatabase(databaseType, databaseDSN)
	if err != nil {
		glog.Fatalf("Failed to connect to database: %v", err)
	}

	// Set up config store based on mode (file, k8s, none).
	var serverOpts []plugin.ServerOption
	switch configStoreStr {
	case "file":
		configStore, err := plugin.NewFileConfigStore(sourcesPath)
		if err != nil {
			glog.Fatalf("Failed to create file config store: %v", err)
		}
		serverOpts = append(serverOpts, plugin.WithConfigStore(configStore))
		logger.Info("using file config store", "path", sourcesPath)
	case "k8s":
		k8sNamespace := envOrDefault("CATALOG_CONFIG_NAMESPACE", "default")
		k8sConfigMap := envOrDefault("CATALOG_CONFIG_CONFIGMAP_NAME", "catalog-sources")
		k8sDataKey := envOrDefault("CATALOG_CONFIG_CONFIGMAP_KEY", "sources.yaml")

		// Create in-cluster K8s client. This requires the catalog-server to run
		// inside a K8s pod with an appropriate ServiceAccount and RBAC granting
		// get/update on the target ConfigMap (see deploy/catalog-server/rbac.yaml).
		k8sCfg, err := rest.InClusterConfig()
		if err != nil {
			glog.Fatalf("Failed to create in-cluster K8s config (is the server running in a pod?): %v", err)
		}
		clientset, err := kubernetes.NewForConfig(k8sCfg)
		if err != nil {
			glog.Fatalf("Failed to create K8s clientset: %v", err)
		}
		configStore := plugin.NewK8sSourceConfigStore(clientset, k8sNamespace, k8sConfigMap, k8sDataKey)
		serverOpts = append(serverOpts, plugin.WithConfigStore(configStore))

		// Wire SecretResolver so that source properties containing SecretRef
		// objects (e.g. {"name":"my-secret","key":"token"}) are resolved from
		// Kubernetes Secrets at runtime. The resolver defaults to k8sNamespace
		// when a SecretRef omits its namespace field.
		secretResolver := plugin.NewK8sSecretResolver(clientset, k8sNamespace)
		serverOpts = append(serverOpts, plugin.WithSecretResolver(secretResolver))

		logger.Info("using k8s config store",
			"namespace", k8sNamespace, "configMap", k8sConfigMap, "dataKey", k8sDataKey)
	case "none":
		logger.Info("config store disabled, mutations will not be persisted")
	default:
		glog.Fatalf("Unknown config store mode: %q (expected file, k8s, or none)", configStoreStr)
	}

	// Set up auth based on CATALOG_AUTH_MODE.
	authMode := os.Getenv("CATALOG_AUTH_MODE")
	switch authMode {
	case "jwt":
		jwtCfg := plugin.JWTRoleExtractorConfig{
			RoleClaim:         envOrDefault("CATALOG_JWT_ROLE_CLAIM", "role"),
			OperatorRoleValue: envOrDefault("CATALOG_JWT_OPERATOR_VALUE", "operator"),
			PublicKeyPath:     os.Getenv("CATALOG_JWT_PUBLIC_KEY_PATH"),
			Issuer:            os.Getenv("CATALOG_JWT_ISSUER"),
			Audience:          os.Getenv("CATALOG_JWT_AUDIENCE"),
			Logger:            logger,
		}
		serverOpts = append(serverOpts, plugin.WithJWTRoleExtractor(jwtCfg))
		logger.Info("using JWT auth",
			"roleClaim", jwtCfg.RoleClaim,
			"operatorValue", jwtCfg.OperatorRoleValue,
			"hasPublicKey", jwtCfg.PublicKeyPath != "")
	case "header", "":
		// Default: use X-User-Role header (development mode)
		if authMode == "" {
			logger.Info("using default header-based auth (X-User-Role)")
		}
	default:
		glog.Fatalf("Unknown auth mode: %q (expected jwt, header, or empty)", authMode)
	}

	// Load governance config if available.
	govConfigPath := envOrDefault("CATALOG_GOVERNANCE_CONFIG", "/config/governance.yaml")
	govConfig, err := governance.LoadGovernanceConfig(govConfigPath)
	if err != nil {
		logger.Warn("failed to load governance config, using defaults", "path", govConfigPath, "error", err)
		govConfig = governance.DefaultGovernanceConfig()
	}
	serverOpts = append(serverOpts, plugin.WithGovernanceConfig(govConfig))

	// Load audit V2 config from environment.
	auditCfg := audit.AuditConfigFromEnv()
	serverOpts = append(serverOpts, plugin.WithAuditConfig(auditCfg))
	logger.Info("audit config loaded",
		"enabled", auditCfg.Enabled,
		"retentionDays", auditCfg.RetentionDays,
		"logDenied", auditCfg.LogDenied)

	// Load async job config from environment.
	jobCfg := jobs.JobConfigFromEnv()
	serverOpts = append(serverOpts, plugin.WithJobConfig(jobCfg))
	logger.Info("job config loaded",
		"enabled", jobCfg.Enabled,
		"concurrency", jobCfg.Concurrency,
		"maxRetries", jobCfg.MaxRetries)

	// Set up tenancy mode.
	tenancyModeStr := envOrDefault("CATALOG_TENANCY_MODE", "single")
	var tenancyMode tenancy.TenancyMode
	switch tenancyModeStr {
	case "single":
		tenancyMode = tenancy.ModeSingle
	case "namespace":
		tenancyMode = tenancy.ModeNamespace
	default:
		glog.Fatalf("Unknown tenancy mode: %q (expected single or namespace)", tenancyModeStr)
	}
	serverOpts = append(serverOpts, plugin.WithTenancyMode(tenancyMode))
	logger.Info("tenancy mode configured", "mode", tenancyModeStr)

	// Set up authorization based on CATALOG_AUTHZ_MODE.
	authzModeStr := envOrDefault("CATALOG_AUTHZ_MODE", "none")
	switch authz.AuthzMode(authzModeStr) {
	case authz.AuthzModeSAR:
		// Create in-cluster K8s client for SAR checks (reuse existing
		// clientset if k8s config store is already configured).
		k8sCfg, err := rest.InClusterConfig()
		if err != nil {
			glog.Fatalf("Failed to create in-cluster K8s config for SAR authz: %v", err)
		}
		authzClient, err := kubernetes.NewForConfig(k8sCfg)
		if err != nil {
			glog.Fatalf("Failed to create K8s clientset for SAR authz: %v", err)
		}
		sarAuthorizer := authz.NewSARAuthorizer(authzClient)
		cachedAuthorizer := authz.NewCachedAuthorizer(sarAuthorizer, authz.DefaultCacheTTL)
		serverOpts = append(serverOpts, plugin.WithAuthorizer(cachedAuthorizer))
		logger.Info("using SAR-based authorization", "cacheTTL", authz.DefaultCacheTTL)
	case authz.AuthzModeNone:
		logger.Info("authorization disabled (CATALOG_AUTHZ_MODE=none)")
	default:
		glog.Fatalf("Unknown authz mode: %q (expected none or sar)", authzModeStr)
	}

	// Set up discovery/capabilities caching.
	cacheCfg := cache.CacheConfigFromEnv()
	serverOpts = append(serverOpts, plugin.WithCacheConfig(cacheCfg))
	logger.Info("cache config loaded",
		"enabled", cacheCfg.Enabled,
		"discoveryTTL", cacheCfg.DiscoveryTTL,
		"capabilitiesTTL", cacheCfg.CapabilitiesTTL,
		"maxSize", cacheCfg.MaxSize)

	// Set up HA features (migration locking, leader election).
	haCfg := ha.HAConfigFromEnv()
	if haCfg.MigrationLockEnabled && gormDB != nil {
		locker := ha.NewMigrationLocker(gormDB)
		serverOpts = append(serverOpts, plugin.WithMigrationLocker(locker))
		logger.Info("migration locking enabled")
	}
	var leaderElector *ha.LeaderElector
	if haCfg.LeaderElectionEnabled {
		k8sCfg, err := rest.InClusterConfig()
		if err != nil {
			glog.Fatalf("Failed to create in-cluster K8s config for leader election: %v", err)
		}
		leClient, err := kubernetes.NewForConfig(k8sCfg)
		if err != nil {
			glog.Fatalf("Failed to create K8s clientset for leader election: %v", err)
		}
		leaderElector = ha.NewLeaderElector(haCfg, leClient, haCfg.Identity, logger)
		serverOpts = append(serverOpts, plugin.WithLeaderElector(leaderElector))
		logger.Info("leader election enabled",
			"identity", haCfg.Identity,
			"lease", haCfg.LeaseName,
			"namespace", haCfg.LeaseNamespace)
	}

	// Create and initialize server
	server := plugin.NewServer(cfg, []string{sourcesPath}, gormDB, logger, serverOpts...)
	if err := server.Init(ctx); err != nil {
		glog.Fatalf("Failed to initialize plugins: %v", err)
	}

	// Mount routes and start
	router := server.MountRoutes()

	if err := server.Start(ctx); err != nil {
		glog.Fatalf("Failed to start plugins: %v", err)
	}

	// startSingletonLoops starts background workers that should only run on
	// one replica at a time (the leader). These are started immediately in
	// single-replica mode or by the leader election callback in HA mode.
	startSingletonLoops := func(loopCtx context.Context) {
		go server.ReconcileLoop(loopCtx)

		retentionDays := auditCfg.RetentionDays
		if govConfig != nil && govConfig.AuditRetention.Days > 0 {
			retentionDays = govConfig.AuditRetention.Days
		}
		retentionWorker := audit.NewRetentionWorker(server.GetAuditStore(), retentionDays, logger)
		go retentionWorker.Run(loopCtx)

		if workerPool := server.NewJobWorkerPool(); workerPool != nil {
			go workerPool.Run(loopCtx)
			logger.Info("async job worker pool started",
				"concurrency", jobCfg.Concurrency,
				"maxRetries", jobCfg.MaxRetries)
		}

		logger.Info("singleton background loops started")
	}

	if haCfg.LeaderElectionEnabled && leaderElector != nil {
		// In HA mode, singleton loops are started/stopped by leader election
		// callbacks. The leader election goroutine blocks until ctx is cancelled.
		var loopCancel context.CancelFunc
		leaderElector.OnStartLeading(func(leaderCtx context.Context) {
			var loopCtx context.Context
			loopCtx, loopCancel = context.WithCancel(leaderCtx)
			startSingletonLoops(loopCtx)
		})
		leaderElector.OnStopLeading(func() {
			if loopCancel != nil {
				loopCancel()
			}
			logger.Info("singleton background loops stopped (lost leadership)")
		})

		go leaderElector.Run(ctx)
		logger.Info("leader election active, singleton loops will start when elected")
	} else {
		// Single-replica mode: start all loops immediately.
		startSingletonLoops(ctx)
	}

	logger.Info("catalog server ready",
		"listen", listenAddr,
		"plugins", plugin.Names(),
	)

	// Create HTTP server with graceful shutdown
	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: router,
	}

	// Start HTTP server in goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			glog.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	logger.Info("shutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	if err := server.Stop(shutdownCtx); err != nil {
		logger.Error("plugin shutdown error", "error", err)
	}

	logger.Info("catalog server stopped")
}

func setupDatabase(dbType, dsn string) (*gorm.DB, error) {
	if dsn == "" {
		// Try to get from environment
		dsn = os.Getenv("DATABASE_DSN")
		if dsn == "" {
			return nil, fmt.Errorf("database DSN is required (use -db-dsn flag or DATABASE_DSN environment variable)")
		}
	}

	if dbType == "" {
		dbType = os.Getenv("DATABASE_TYPE")
		if dbType == "" {
			dbType = "postgres"
		}
	}

	// Create embedmd connector
	cfg := &embedmd.EmbedMDConfig{
		DatabaseType: dbType,
		DatabaseDSN:  dsn,
	}

	connector, err := datastore.NewConnector("embedmd", cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connector: %w", err)
	}

	// Connect to initialize the database
	// We need a minimal spec just to establish the connection
	_, err = connector.Connect(datastore.NewSpec())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get the GORM DB from the db package
	dbConnector, ok := db.GetConnector()
	if !ok {
		return nil, fmt.Errorf("database connector not available")
	}

	gormDB, err := dbConnector.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to get GORM connection: %w", err)
	}

	return gormDB, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

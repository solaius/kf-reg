# Configuration Management

This document covers the CLI framework, configuration loading, and environment variable handling.

## Overview

**Location:**
- `cmd/root.go` - Root command and Viper setup
- `cmd/config.go` - Configuration structures
- `cmd/proxy.go` - Proxy command and flags

## CLI Framework: Cobra

### Command Structure

```
model-registry
├── proxy          # Main API server
├── catalog        # Catalog service
└── controller     # Kubernetes controller (separate)
```

### Root Command

```go
// cmd/root.go
var rootCmd = &cobra.Command{
    Use:   "model-registry",
    Short: "Kubeflow Model Registry",
    Long:  `Model Registry provides a central repository for ML model metadata`,
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        return initConfig()
    },
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func init() {
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
        "config file (default is $HOME/.model-registry.yaml)")
    rootCmd.AddCommand(proxyCmd)
    rootCmd.AddCommand(catalogCmd)
}
```

### Proxy Command

```go
// cmd/proxy.go
var proxyCmd = &cobra.Command{
    Use:   "proxy",
    Short: "Start the OpenAPI proxy server",
    Long:  `Starts the Model Registry REST API server`,
    RunE: func(cmd *cobra.Command, args []string) error {
        return runProxy()
    },
}

func init() {
    // Server configuration
    proxyCmd.Flags().StringVarP(&proxyCfg.Hostname, "hostname", "n",
        "localhost", "Server hostname")
    proxyCmd.Flags().IntVarP(&proxyCfg.Port, "port", "p",
        8080, "Server port")

    // Database configuration
    proxyCmd.Flags().StringVar(&proxyCfg.EmbedMD.DatabaseType, "embedmd-database-type",
        "mysql", "Database type (mysql or postgres)")
    proxyCmd.Flags().StringVar(&proxyCfg.EmbedMD.DatabaseDSN, "embedmd-database-dsn",
        "", "Database connection string")

    // TLS configuration
    proxyCmd.Flags().StringVar(&proxyCfg.EmbedMD.TLS.CertFile, "tls-cert",
        "", "Path to TLS certificate file")
    proxyCmd.Flags().StringVar(&proxyCfg.EmbedMD.TLS.KeyFile, "tls-key",
        "", "Path to TLS key file")
    proxyCmd.Flags().StringVar(&proxyCfg.EmbedMD.TLS.CAFile, "tls-ca",
        "", "Path to CA certificate file")
}
```

## Configuration Structures

### Base Config

```go
// cmd/config.go
type Config struct {
    DbFile      string   `mapstructure:"db-file"`
    Hostname    string   `mapstructure:"hostname"`
    Port        int      `mapstructure:"port"`
    LibraryDirs []string `mapstructure:"library-dirs"`
}
```

### Proxy Config

```go
type ProxyConfig struct {
    Config
    EmbedMD EmbedMDConfig `mapstructure:"embedmd"`
}

type EmbedMDConfig struct {
    DatabaseType string     `mapstructure:"database-type"`
    DatabaseDSN  string     `mapstructure:"database-dsn"`
    TLS          TLSConfig  `mapstructure:"tls"`
}

type TLSConfig struct {
    CertFile           string `mapstructure:"cert-file"`
    KeyFile            string `mapstructure:"key-file"`
    CAFile             string `mapstructure:"ca-file"`
    ServerName         string `mapstructure:"server-name"`
    InsecureSkipVerify bool   `mapstructure:"insecure-skip-verify"`
}
```

## Viper Configuration

### Initialization

```go
// cmd/root.go
func initConfig() error {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        home, err := os.UserHomeDir()
        if err != nil {
            return err
        }

        viper.AddConfigPath(home)
        viper.SetConfigType("yaml")
        viper.SetConfigName(".model-registry")
    }

    // Environment variable configuration
    viper.SetEnvPrefix("MR")
    viper.AutomaticEnv()
    viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

    // Read config file
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return err
        }
    }

    return nil
}
```

### Environment Variables

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--hostname` | `MR_HOSTNAME` | Server hostname |
| `--port` | `MR_PORT` | Server port |
| `--embedmd-database-type` | `MR_EMBEDMD_DATABASE_TYPE` | Database type |
| `--embedmd-database-dsn` | `MR_EMBEDMD_DATABASE_DSN` | Database DSN |
| `--tls-cert` | `MR_TLS_CERT` | TLS certificate |
| `--tls-key` | `MR_TLS_KEY` | TLS key |
| `--tls-ca` | `MR_TLS_CA` | TLS CA certificate |

### Config File Format

```yaml
# ~/.model-registry.yaml
hostname: 0.0.0.0
port: 8080

embedmd:
  database-type: mysql
  database-dsn: "user:password@tcp(localhost:3306)/model_registry?parseTime=true"
  tls:
    cert-file: /path/to/cert.pem
    key-file: /path/to/key.pem
    ca-file: /path/to/ca.pem
```

## Configuration Priority

Configuration values are loaded with the following priority (highest to lowest):

1. **Command-line flags**
2. **Environment variables**
3. **Config file**
4. **Default values**

## Catalog Service Configuration

### Flags

```go
// catalog/cmd/catalog.go
var catalogCmd = &cobra.Command{
    Use:   "catalog",
    Short: "Start the Catalog API server",
    RunE:  runCatalog,
}

func init() {
    catalogCmd.Flags().StringVar(&catalogCfg.ListenAddress, "listen",
        "0.0.0.0:8080", "Server listen address")
    catalogCmd.Flags().StringSliceVar(&catalogCfg.CatalogsPaths, "catalogs-path",
        []string{}, "Path(s) to catalog sources.yaml files")
    catalogCmd.Flags().StringVar(&catalogCfg.PerformanceMetricsPath, "performance-metrics",
        "", "Path to performance metrics directory")
}
```

### Environment Variables

| Flag | Environment Variable |
|------|---------------------|
| `--listen` | `CATALOG_LISTEN` |
| `--catalogs-path` | `CATALOG_CATALOGS_PATH` |
| `--performance-metrics` | `CATALOG_PERFORMANCE_METRICS` |

## BFF Configuration

### Environment Variables

```go
// clients/ui/bff/internal/config/config.go
type Config struct {
    Port                    int    `envconfig:"PORT" default:"8080"`
    DeploymentMode          string `envconfig:"DEPLOYMENT_MODE" default:"standalone"`
    AuthMethod              string `envconfig:"AUTH_METHOD" default:"none"`
    MockK8sClient           bool   `envconfig:"MOCK_K8S_CLIENT" default:"false"`
    MockMRClient            bool   `envconfig:"MOCK_MR_CLIENT" default:"false"`
    ModelRegistryURL        string `envconfig:"MODEL_REGISTRY_URL"`
    CatalogURL              string `envconfig:"CATALOG_URL"`
    TLSCertFile             string `envconfig:"TLS_CERT_FILE"`
    TLSKeyFile              string `envconfig:"TLS_KEY_FILE"`
    CABundlePath            string `envconfig:"CA_BUNDLE_PATH"`
}
```

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DEPLOYMENT_MODE` | `standalone` | Deployment mode |
| `AUTH_METHOD` | `none` | Authentication method |
| `MOCK_K8S_CLIENT` | `false` | Use mock K8s client |
| `MOCK_MR_CLIENT` | `false` | Use mock MR client |
| `MODEL_REGISTRY_URL` | - | Backend URL |
| `CATALOG_URL` | - | Catalog service URL |

## Frontend Configuration

### Environment Variables

```typescript
// clients/ui/frontend/src/app/utilities/const.ts
export const URL_PREFIX = process.env.URL_PREFIX || '/model-registry';
export const BFF_API_VERSION = process.env.BFF_API_VERSION || 'v1';
export const DEPLOYMENT_MODE = process.env.DEPLOYMENT_MODE || 'standalone';
export const STYLE_THEME = process.env.STYLE_THEME || 'Patternfly';
export const POLL_INTERVAL = parseInt(process.env.POLL_INTERVAL || '30000');
export const KUBEFLOW_USERNAME = process.env.KUBEFLOW_USERNAME || 'user@example.com';
```

## Docker Compose Configuration

### MySQL Profile

```yaml
# docker-compose.yaml
services:
  model-registry:
    environment:
      - MR_EMBEDMD_DATABASE_TYPE=mysql
      - MR_EMBEDMD_DATABASE_DSN=root:password@tcp(mysql:3306)/model_registry?parseTime=true
    depends_on:
      mysql:
        condition: service_healthy

  mysql:
    image: mysql:8.3.0
    environment:
      - MYSQL_ROOT_PASSWORD=password
      - MYSQL_DATABASE=model_registry
```

### PostgreSQL Profile

```yaml
services:
  model-registry:
    environment:
      - MR_EMBEDMD_DATABASE_TYPE=postgres
      - MR_EMBEDMD_DATABASE_DSN=host=postgres port=5432 user=postgres password=password dbname=model_registry sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=model_registry
```

## Kubernetes Configuration

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: model-registry-config
data:
  MR_EMBEDMD_DATABASE_TYPE: mysql
  MR_HOSTNAME: "0.0.0.0"
  MR_PORT: "8080"
```

### Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: model-registry-secrets
type: Opaque
data:
  MR_EMBEDMD_DATABASE_DSN: <base64-encoded-dsn>
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: model-registry
spec:
  template:
    spec:
      containers:
      - name: model-registry
        envFrom:
        - configMapRef:
            name: model-registry-config
        - secretRef:
            name: model-registry-secrets
```

---

[Back to Backend Index](./README.md) | [Previous: Middleware](./middleware.md)

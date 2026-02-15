# M6: Docker Compose with catalog-server and MCP Plugin

**Date**: 2026-02-15
**Status**: Complete
**Phase**: Phase 1: Platform Architecture

## Summary

Switched the Docker Compose local development stack from the legacy `catalog` command to the new plugin-based `catalog-server` binary, enabling both the model catalog and MCP servers plugins to run together in Docker. This required building the `catalog-server` binary in the Dockerfile, creating a combined sources config in the new `CatalogSourcesConfig` format, and fixing a config format incompatibility in the model plugin wrapper.

## Motivation

- The Docker Compose stack was using the old `catalog` command, which only supports the model catalog. The `/api/plugins` endpoint returned 404.
- With the MCP plugin implemented (M2) and the plugin framework hardened (M1), the compose stack needed to use `catalog-server` to validate the full multi-plugin architecture end-to-end in a containerized environment.
- Developers and reviewers need a one-command way to bring up the complete stack and verify both plugins work together.

## What Changed

### Files Created
| File | Purpose |
|------|---------|
| `catalog/internal/catalog/testdata/test-catalog-server-sources.yaml` | Combined sources config in `CatalogSourcesConfig` format with `models` and `mcp` sections |
| `catalog/internal/catalog/testdata/mcp-servers.yaml` | Copy of MCP test data into shared testdata directory so single volume mount serves both plugins |

### Files Modified
| File | Change |
|------|--------|
| `Dockerfile` | Added `catalog-server` build step and `COPY` into final image |
| `docker-compose-local.yaml` | Changed `model-catalog` service from `catalog` command to `/catalog-server` entrypoint with DB DSN passed directly |
| `catalog/plugins/model/plugin.go` | Fixed `Init()` to populate `SourceCollection` directly from plugin config instead of re-reading the config file with the old parser |

## How It Works

### Dockerfile Changes

The Dockerfile now builds two binaries from the same source tree. The existing `model-registry` binary (which includes the old `catalog` subcommand) is built via `make build/compile`. The new `catalog-server` binary is built separately:

```dockerfile
# Build catalog-server binary (plugin-based catalog)
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -buildvcs=false -o catalog-server ./cmd/catalog-server

# Final image includes both binaries
COPY --from=builder /workspace/model-registry .
COPY --from=builder /workspace/catalog-server .
```

The default `ENTRYPOINT` remains `/model-registry` for backward compatibility. The compose file overrides it with `/catalog-server` for the catalog service.

### Docker Compose Configuration

The `model-catalog` service now uses `catalog-server` with explicit flags:

```yaml
model-catalog:
  entrypoint: ["/catalog-server"]
  command:
    - --listen=0.0.0.0:8081
    - --sources=/testdata/test-catalog-server-sources.yaml
    - --db-type=postgres
    - --db-dsn=host=postgres port=5432 user=postgres password=demo dbname=model_catalog sslmode=disable
```

The database connection string is passed directly via `--db-dsn` instead of relying on `PGHOST`/`PGUSER`/`PGPASSWORD` environment variables, which were specific to the old catalog code.

### Combined Sources Config

The new `test-catalog-server-sources.yaml` uses the `CatalogSourcesConfig` format that the plugin server expects:

```yaml
apiVersion: catalog/v1alpha1
kind: CatalogSources
catalogs:
  models:
    sources:
      - id: catalog1
        name: "Catalog 1"
        type: yaml
        properties:
          yamlCatalogPath: test-yaml-catalog.yaml
  mcp:
    sources:
      - id: mcp1
        name: "MCP Servers"
        type: yaml
        properties:
          yamlCatalogPath: mcp-servers.yaml
```

The `catalogs` map keys (`models`, `mcp`) match each plugin's source key. The plugin server parses this file, extracts each plugin's section, and passes it during `Init()`.

### Model Plugin Config Compatibility Fix

The model plugin wraps the legacy catalog loader, which re-reads config files during `Start()` using `yaml.UnmarshalStrict`. The legacy parser expects the old flat format (`catalogs: [...]` list) and rejects unknown fields like `apiVersion` and `kind`.

The fix changes `Init()` to populate the loader's `SourceCollection` directly from the already-parsed plugin config, bypassing the file re-read entirely:

```go
// Create the loader with no paths — we populate sources directly
p.loader = catalog.NewLoader(services, nil)

// Convert plugin config sources to the old loader's Source format
sources := make(map[string]catalog.Source, len(cfg.Section.Sources))
for _, src := range cfg.Section.Sources {
    sources[src.ID] = catalog.Source{
        CatalogSource: apimodels.CatalogSource{
            Id:      src.ID,
            Name:    src.Name,
            Enabled: src.Enabled,
            Labels:  src.Labels,
        },
        Type:       src.Type,
        Properties: src.Properties,
        Origin:     origin,
    }
}
p.loader.Sources.Merge(origin, sources)
```

This mirrors the approach used by the MCP plugin (which was designed natively for the plugin system). The old loader receives an empty paths list, skips file parsing in `Start()`, and loads models directly from the pre-populated `SourceCollection`.

## Key Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Build both binaries in one Dockerfile | Keeps a single image for all services; compose selects the entrypoint | Separate Dockerfiles for registry vs catalog-server (more maintenance) |
| Pass DB DSN via `--db-dsn` flag | Explicit, matches catalog-server's flag interface | Use `PGHOST`/`PGUSER` env vars (only works with old catalog code) |
| Fix model plugin to populate sources directly | Eliminates config format incompatibility at the root cause | Create old-format wrapper config files (fragile, duplicates data) |
| Copy MCP test data into shared testdata dir | Single volume mount serves both plugins | Separate volume mounts per plugin (more compose complexity) |

## Testing

- `go build ./catalog/plugins/model/...` — model plugin compiles with the new import and Init() changes
- `go build ./cmd/catalog-server/...` — catalog-server links both plugins correctly
- `go test ./pkg/catalog/plugin/...` — all 15 plugin framework tests pass
- Docker Compose end-to-end: all 4 services start, all endpoints respond correctly

## Verification

```bash
# Bring up the full stack
DB_TYPE=postgres docker compose -f docker-compose-local.yaml --profile postgres up --build -d

# Wait for services to initialize
sleep 10

# Verify all containers are running
DB_TYPE=postgres docker compose -f docker-compose-local.yaml --profile postgres ps

# Test plugin discovery
curl -s http://localhost:8081/api/plugins
# → {"count":2,"plugins":[{"name":"mcp",...,"healthy":true},{"name":"model",...,"healthy":true}]}

# Test MCP servers endpoint
curl -s http://localhost:8081/api/mcp_catalog/v1alpha1/mcpservers
# → {"items":[{"name":"filesystem-server",...},{"name":"database-query-server",...},{"name":"web-search-server",...}],"size":3}

# Test model catalog (backward compatibility)
curl -s http://localhost:8081/api/model_catalog/v1alpha1/models | head -c 200

# Test readiness
curl -s http://localhost:8081/readyz
# → {"plugins":{"mcp":true,"model":true},"status":"ready"}

# Test UI
curl -s -o /dev/null -w "%{http_code}" http://localhost:9000
# → 200
```

## Dependencies & Impact

- **Depends on**: M1 (plugin framework), M2 (MCP plugin), catalog-server binary (`cmd/catalog-server/main.go`)
- **Enables**: Developers can now test the full multi-plugin architecture with a single `docker compose up` command
- **Backward compatibility**: The default `ENTRYPOINT` in the Dockerfile remains `/model-registry`, so existing deployments that don't override the entrypoint are unaffected. The old `catalog` subcommand still works if invoked directly.

## Open Items

- Hot-reload is not active when using the pre-populated source approach (no file paths to watch). For production, the model plugin could be enhanced to watch the original config file and re-populate on change.
- The `docker-compose.yaml` (non-local) file has not been updated to use `catalog-server` — only the local development compose file was changed.
- The MCP test data file is duplicated in two locations (`catalog/plugins/mcp/testdata/` and `catalog/internal/catalog/testdata/`). A symlink or shared directory could eliminate the duplication.

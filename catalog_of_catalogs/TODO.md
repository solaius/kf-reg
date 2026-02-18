# Catalog of Catalogs — Open Items & TODOs

Aggregated from all Phase 1–9 implementation reports. Organized by category with source milestone references.

---

## Security & Auth

- [x] **Production authentication/authorization**: JWT-based `RoleExtractor` implemented in `jwt_role_extractor.go`. Supports RS256 verification, nested claim paths (Keycloak `realm_access.roles`), and trusted-proxy mode. Wired in `main.go` via `CATALOG_AUTH_MODE=jwt`. `X-User-Role` header remains available for dev. *(M2.1, M5.9 — resolved in M5.9 gate fixes)*
- [x] **FilterQuery injection safety**: Client-side `sanitizeFilterValue()` added to escape single quotes in filter values. Server-side parser confirmed safe — uses parameterized queries (`?` placeholders) via GORM. Injection attempt tests added to `parser_test.go` and `query_builder_test.go`. *(M5.9 — resolved in M5.9 gate fixes)*
- [x] **SecretRef verification**: Comprehensive integration tests added (`TestResolveSecretRefsComprehensive`, `TestIsSecretRefEdgeCases`, `TestSecretRefResolution_E2E_FullFlow`) covering namespace defaulting, missing secrets/keys, multi-property resolution, cross-namespace access. Real-cluster verification steps documented. *(M4.6 — resolved in M5.9 gate fixes)*
- [ ] **TLS in Docker Compose**: No TLS configuration in `docker-compose.catalog.yaml`. Acceptable for local dev, but production needs TLS termination (ingress or sidecar proxy). Governance endpoints are not hardened beyond existing stack posture. *(M3.1, M7)*
- [ ] **Fail-closed identity in SAR mode**: Identity middleware defaults to `"anonymous"` when `X-Remote-User` header is missing. In `CATALOG_AUTHZ_MODE=sar`, a missing identity header likely indicates auth proxy misconfiguration and should reject the request (fail-closed) rather than silently proceeding as anonymous. *(M8.3, Phase 8 review)*

## API & Backend

- [ ] **Source config persistence to file/ConfigMap**: Management mutations currently modify in-memory config only. File-backed and K8s ConfigMap persistence are implemented (Phase 4) but write-back of YAML source files requires writable volume mounts. *(M2.1, M3.4)*
- [x] **Async refresh for large sources**: Refresh endpoints now enqueue `RefreshJob` and return `202 Accepted` with `jobId` for status polling via `/api/jobs/v1alpha1/refresh/{id}`. Synchronous fallback when no job store configured. *(M4.3, resolved M8.5)*
- [x] **Rate limiting on refresh endpoints**: `RefreshRateLimiter` implemented per-source with configurable cooldown. Applies at job enqueue time in async mode. *(M2.1, resolved M8.5)*
- [ ] **Per-namespace refresh rate limits / quotas**: Global rate limiter exists but no per-namespace quotas to prevent "refresh spam" in multi-tenant deployments. A noisy tenant can monopolize refresh capacity. *(M8.5, Phase 8 review)*
- [ ] **Batch entity counting**: `CountBySource` executes one DB query per source in `ListSources`. For many sources, use a `GROUP BY source_id` batch query. *(M3.3)*
- [ ] **`ResolvePluginBasePath` caching**: BFF makes a full `GET /api/plugins` call for every management request. Cache the plugin list to reduce round-trips. *(M3.2)*
- [ ] **BFF health check to catalog-server**: Fail-fast validates config only, not connectivity. Add a startup health check. *(M3.2)*
- [ ] **BFF response shape validation**: BFF passes through JSON as-is with no schema validation between BFF and catalog-server. *(M3.2)*
- [ ] **`RetryOnConflict` in management handlers**: Available for K8s ConfigMap store but not yet called from handlers for concurrent writes. *(M4.6)*
- [ ] **`CleanupPluginData` lifecycle hook**: Defined but not called from any plugin lifecycle (shutdown, unregistration). *(M4.6)*
- [ ] **`LastRefreshStatus`/`LastRefreshSummary` fields**: Defined in management types but not populated by MCP plugin's `ListSources()`. *(M4.3)*
- [ ] **Property extraction code generation**: The converter switch in `api_mcpserver_service_impl.go` is handwritten. Could be generated from `catalog.yaml` to prevent drift. *(M3.3)*
- [ ] **OpenAPI/Go model/converter sync validation**: No automated linter to catch drift between OpenAPI spec, Go model, and converter. *(M3.3)*
- [ ] **K8s Watch() implementation**: Currently a stub (returns nil). Could use informers for push-based reconciliation. *(M4.1)*
- [ ] **K8s annotation size limits**: Revision data stored in annotations is limited by etcd value size. Very large configs may not have full snapshot history. *(M4.1)*
- [ ] **Plugin-specific strict validation schemas**: `strict_fields` layer validates against generic `SourceConfig` struct. Plugins could provide custom strict structs. *(M4.2)*
- [ ] **Validation warnings**: `DetailedValidationResult` supports warnings but no layers currently emit them. The `WarningOnly` mechanism could extend beyond security to deprecation warnings, etc. *(M4.2, M4.6)*
- [ ] **Git provider webhook-triggered sync**: Currently uses periodic polling only. Webhook-triggered sync for instant updates on push. *(M6.1)*
- [ ] **Git provider integration tests with remote repos**: Unit tests use local bare repos. Need integration tests against real HTTP/HTTPS remote repositories (e.g., GitHub). *(M6.1, M6.3.1)*
- [ ] **Git provider auth token end-to-end validation**: Auth token mechanism exists in git provider but not yet tested end-to-end with private repos. *(M6.3.1)*
- [ ] **Wire Git provider into additional plugins**: Only Agents plugin currently wires Git provider at runtime. Prompts, Skills, and other plugins could follow the same pattern. *(M6.3.1)*
- [ ] **MCP server reference resolution for Skills**: `execution.mcpServerRef` in Skills could link to MCP plugin entities but cross-plugin resolution is not implemented. *(M6.5)*
- [ ] **OpenAPI spec fetching for `openapi_operation` skills**: Skills with `skillType: openapi_operation` reference external OpenAPI specs but fetching/resolution is deferred. *(M6.5)*
- [ ] **OCI bundle resolution for Guardrails/Policies `bundleRef`**: Requires OCI provider (deferred to Phase 6.5+). *(M6.4)*
- [ ] **Cross-asset link target resolution**: Agent cross-links (`skillRef`, `guardrailRef`, etc.) populate `AssetLinks.Related` with `{Kind, Name}` but don't verify targets exist. *(M6.3)*
- [ ] **`catalog-gen build-server --compile` flag**: Server builder generates `main.go` + `go.mod` + `Dockerfile` but does not actually compile the binary. The `--compile` flag is accepted but prints "not yet implemented". Needs `go build` invocation with proper module resolution. *(M9.5)*
- [ ] **OCI provider support in server builder**: Server builder assumes Go module imports for plugins. Does not support packaging plugins as OCI artifacts or pulling plugin code from OCI registries. *(M9.5)*
- [ ] **Cosign/Sigstore image signing hooks**: Server builder generates Dockerfiles but has no integration with cosign or Sigstore for signing built images. Hooks are designed in the manifest schema but not implemented. *(M9.5)*
- [ ] **Wire `VerifyingProvenanceExtractor` as default-on**: The decorator exists and persists integrity fields to DB, but is not wired into the default server pipeline. Callers must explicitly wrap their extractor. Should be activated by default so provenance integrity verification happens automatically for all plugins. *(M7, M7.1)*
- [ ] **OPA/Rego approval policy adapter**: Current YAML-based policy engine covers basic use cases. Enterprise governance typically requires pluggable policy backends (OPA/Rego) for complex conditional logic, external data sources, and audit-grade policy evaluation. *(M7, Phase 8)*
- [x] **RBAC layering for governance actions**: SAR-based authorization implemented via `pkg/authz`. Governance endpoints mapped to `approvals` resource with per-verb permissions. `RequirePermission` middleware enforces role-based access when `CATALOG_AUTHZ_MODE=sar`. *(M7, resolved M8.3)*
- [ ] **Dynamic namespace discovery from Kubernetes**: `/api/tenancy/v1alpha1/namespaces` returns a static list from `CATALOG_NAMESPACES` env var or `["default"]`. Not dynamically derived from Kubernetes RBAC (SAR namespace enumeration) or Kubeflow Profile access. Should list only namespaces the authenticated user has SAR access to. *(M8.1, M8.8, Phase 8 review)*
- [ ] **Phase 5 entity action endpoint format**: `TestConformance` action tests (tag/annotate/deprecate) fail across all plugins due to entity action endpoint format mismatch. Pre-existing Phase 5 issue, not caused by Phase 7. *(M5, M7.1)*

## UI Frontend

- [ ] **Frontend consumption of extended UI hints (Phase 9)**: Phase 9 formalized 11 `FieldDisplayType` values (text, markdown, badge, tags, link, repoRef, imageRef, dateTime, code, json, secretRef) and added `ListViewHints`, `DetailViewHints`, `SearchHints`, and `ActionDisplayHints` to `EntityUIHints`. The backend serves these in the capabilities document, but the React frontend (`GenericListView`, `GenericDetailView`, etc.) does not yet read or render based on the extended hints. Until consumed, richer layouts (e.g., markdown rendering, badge styling, faceted search) are not realized. *(M9.2, M9.7)*
- [ ] **GenericActionDialog tags input UX**: Currently a simple comma-separated `TextInput`. A proper `LabelGroup` chip input would improve UX. *(M5.5)*
- [ ] **YAML syntax validation**: No frontend validation of YAML content before save on the Manage Source page. *(M3.4)*
- [ ] **Save confirmation dialog**: No confirmation before overwriting server-side YAML file. *(M3.4)*
- [ ] **Auto-refresh after save**: Frontend does not auto-refresh after saving source config to show updated entity counts. *(M3.4)*
- [ ] **Large YAML files**: Content returned inline in API response may impact response size. *(M3.4)*
- [ ] **Source create/edit form**: Only table view with toggle/delete exists. Full source creation form is only available for MCP plugin (model plugin redirects to existing Settings pages). *(M2.3)*
- [ ] **Refresh progress indicator**: Currently shows success/error after completion, no in-progress feedback. *(M2.3)*
- [ ] **Error boundary per API call**: Generic alert on failure; individual component error boundaries would improve UX. *(M2.3)*
- [ ] **MCP detail page tabs**: Only Overview tab implemented. Tools/Resources/Prompts tabs are placeholders. *(M2.5)*
- [ ] **MCP server logos**: Cards use CubesIcon placeholder. Real logos need data URIs or hosted image URLs. *(M2.5)*
- [ ] **Search debouncing**: Currently instant. May need debounce for very large datasets. *(M2.5)*
- [ ] **Per-layer warning counts in validation UI**: Quick triage without expanding the validation panel. *(M4.6)*
- [ ] **Component unit tests for generic catalog components**: No component-level unit tests yet for GenericListView, GenericActionDialog, etc. *(M5.5)*
- [ ] **Enhanced UI for asset details**: Generic catalog detail view should align with the MCP Catalog detail page pattern (structured sections, tabs, rich rendering). Current generic detail view renders raw field values without asset-type-specific formatting. *(Phase 6+)*
- [ ] **Plugin/Entity creation, import, and management from UI**: Full schema-driven CRUD for creating, importing, editing, and deleting entities and plugins directly from the frontend UI. Currently only source-level management (add/enable/disable/delete) is available. *(Phase 6+)*

## CLI

- [ ] **Shell completion**: Cobra supports it natively but not yet wired. Dynamic commands require a running server for completion suggestions. *(M2.2, M5.6)*
- [ ] **`--params-file` flag for actions**: Only inline `key=val` is supported. Add JSON/YAML file loading for action parameters. *(M5.6)*
- [ ] **Interactive confirmation prompts**: No confirmation for destructive operations (delete, disable). *(M2.2)*
- [x] **`--namespace` flag**: `catalogctl --namespace` / `-n` persistent flag implemented. Resolution: flag > `CATALOG_NAMESPACE` env > empty. `catalogctl namespaces` command lists available namespaces. *(M2.2, resolved M8.8)*
- [ ] **Source management subcommands in catalogctl**: `sources validate`, `sources apply`, `sources enable`, `sources disable` are not yet implemented (only `list` and `refresh`). *(M5.6)*
- [ ] **CLI plugin discovery caching**: Each invocation makes a fresh `GET /api/plugins` call. *(M5.6)*

## Testing

- [ ] **Conformance harness CI integration**: The importable conformance harness (`pkg/catalog/conformance/`) requires a live catalog-server with fixtures loaded to fully execute. CI needs a job that boots the server (e.g., Docker Compose), loads fixture data, and runs the harness. Currently only compile-time tests pass in CI; the 6 runtime categories (A-F) are not exercised. *(M9.3, M9.7)*
- [ ] **Governance checks CI enforcement**: `catalog-gen validate --governance` can verify all 8 plugins pass governance checks, but this is not wired into CI as a gate. PRs that break governance (e.g., removing plugin.yaml, dropping license) are not caught automatically. *(M9.6)*
- [ ] **Plugin index schema validation in CI**: `deploy/plugin-index/schema.yaml` defines a JSON Schema for index entries, but no CI step validates that all entries in `deploy/plugin-index/plugins/*.yaml` conform to the schema. *(M9.6)*
- [ ] **BFF handler unit tests**: Currently only compile-verified. Need mock HTTP client tests for isolation. *(M2.4, M4.5)*
- [ ] **Playwright/Cypress automated UI test suite**: Tests defined but require full stack for execution. *(M2.4, M4.5)*
- [ ] **Test coverage enforcement in CI**: `go test -cover` available but not enforced. *(M4.5)*
- [ ] **Extended field unit test coverage**: `TestConvertToOpenAPIModel` does not cover the 10 extended MCP fields (verified, certified, deploymentMode, etc.). *(M3.3)*
- [ ] **Integration tests against running catalog-server**: Docker Compose setup needed. *(M2.2, M2.4)*
- [ ] **Full end-to-end test suite (unit and integration)**: Comprehensive test coverage across all 8 plugins, providers, management endpoints, BFF handlers, and UI components. Includes conformance suite execution against full stack, per-plugin entity parsing tests, filter/action tests, and Docker-based integration tests. *(M6.6, Phase 6+)*
- [ ] **Conformance suite execution against full Phase 6 stack**: Conformance suite auto-discovers plugins and should pass without code changes, but has not been run against the full 8-plugin stack. *(M6.6)*

## Documentation

- [ ] **Automated doc link-checking**: No harness to verify code snippets in docs stay in sync with source. *(M1.M5)*
- [ ] **React component example in UI/CLI guide**: Conceptual extension points described but no working React component example. *(M1.M5)*
- [ ] **`catalog-gen` error messages and troubleshooting guide**: Common failures (missing `catalog.yaml`, invalid types) not documented. *(M1.M5)*
- [ ] **catalog-gen generated docs review**: `gen_docs.go` generates a documentation kit (README, provider guide, schema guide, testing guide, publishing guide), but the generated content uses template placeholders. A review pass should ensure the generated docs are accurate for real-world plugin development. *(M9.4)*

## Observability & Telemetry

- [ ] **Prometheus metrics**: No metrics exposition endpoint. Key metrics needed: request latency histograms, SAR decision cache hit/miss rates, job queue depth, refresh duration, active workers, leader election status. *(Phase 8 review)*
- [ ] **OpenTelemetry tracing**: No distributed tracing instrumentation. Traces across tenancy middleware -> authz -> handler -> DB would aid debugging in multi-tenant deployments. *(Phase 8 review)*

## Infrastructure & DevOps

- [ ] **PostgreSQL persistent volume**: Data is ephemeral (`docker compose down -v` destroys all). Named volume for persistent dev data. *(M3.1)*
- [ ] **Hot-reload for model plugin**: Not active when using pre-populated source approach (no file paths to watch). *(M1.M6)*
- [ ] **Non-local Docker Compose file**: `docker-compose.yaml` (non-local) not updated to use `catalog-server`. *(M1.M6)*
- [ ] **MCP test data duplication**: Data duplicated in `catalog/plugins/mcp/testdata/` and `catalog/internal/catalog/testdata/`. Symlink or shared directory. *(M1.M6)*
- [ ] **Health check endpoint**: Catalog-server Docker health check uses `--help` flag. Proper `/healthz` or `/readyz` HTTP probes were added in Phase 4 (M4.4) but Docker Compose may need updating. *(M3.1, M4.4)*
- [ ] **Healthcheck binary vs gRPC**: Current binary could be replaced by `grpc_health_probe` if gRPC is added. *(M4.4)*
- [ ] **Versioned DB migrations**: Phase 8 uses GORM `AutoMigrate` (with advisory locking for multi-replica safety). For enterprise ops, versioned migrations (e.g., golang-migrate, atlas) would provide explicit rollback, migration history, and safer schema evolution. Current rollback story is limited to "add column with default" idempotency. *(M8.2, M8.7, Phase 8 review)*
- [ ] **Dockerfile updates for production builds**: Sample data `COPY` instructions needed for Phase 6 plugins (prompts, agents, guardrails, policies, skills). Current Dockerfile only copies data for model and MCP plugins. *(M6.6)*
- [ ] **CI pipeline integration for Phase 6**: Phase 6 builds should run conformance suite as part of PR checks. *(M6.6)*

## Data & Asset Quality

- [ ] **Confirm all models from original model-registry default installation are configured and available in catalog**: Verify that the model plugin sources include all models that were present in the model-registry prior to catalog_of_catalogs updates, and that they are fully functional (list, detail, filters, source tabs). *(Phase 6+)*
- [ ] **Audit of loaded assets to only have real assets in the system**: Review all sample/seed data across all 8 plugins. Replace placeholder or synthetic entries with real, production-representative assets. Create new real assets where necessary. *(Phase 6+)*
- [ ] **AI asset source enhancements**: Expand beyond YAML and Git providers to include additional real-world source types (OCI registries, HTTP/REST APIs, Hugging Face Hub, S3/object storage, database-backed sources). Improve source configuration validation, discovery, and auto-refresh capabilities. *(Phase 6+)*

## Future Enhancements (Low Priority)

- [ ] **Pagination total accuracy**: `totalSize` in "Showing X of Y" reflects server's `size` field which may shift between page fetches. Cosmetic. *(M5.9)*
- [ ] **EntityGetter for future multi-param plugins**: Only model plugin implements `EntityGetter`. MCP/Knowledge don't need it (single-param get), but future plugins with multi-param get should. *(M5.9)*
- [ ] **Plugin discovery endpoint filtering**: No support for filtering by health status or name. *(M1.M4)*
- [ ] **Plugin metadata caching in BFF**: Every request proxies to catalog-server. May need caching for high-traffic deployments. *(M1.M4)*
- [ ] **DB connectivity health check depth**: Uses `Ping()` (connection check) not `SELECT 1` (query capability). Likely unnecessary but noted. *(M4.4)*
- [ ] **OCI artifact provider**: Deferred from Phase 6. Would enable loading catalog data from OCI registries (e.g., guardrail bundles, policy bundles). *(M6.1)*
- [ ] **Remaining asset-type plugins**: Datasets, Evaluators, Benchmarking, and Notebooks plugins deferred to Phase 6.5/7. Same plugin template pattern applies. *(Phase 6 plan)*
- [ ] **A2A protocol integration**: Agent-to-agent communication and multi-agent handoffs for the Agents plugin. *(M6.3)*
- [ ] **Guardrail runtime integration testing**: Testing with actual NeMo Guardrails or Guardrails AI runtimes (catalog is discovery-only, but validation of config format against real runtimes would be valuable). *(M6.4)*
- [ ] **Policy evaluation engine integration**: Policy plugin is discovery-only. Future integration with OPA/Rego evaluation for policy enforcement. *(M6.4)*
- [ ] **Prompt rendering/execution endpoint**: Prompt Templates plugin is discovery-only. Future endpoint for rendering templates with parameters. *(M6.2)*
- [ ] **Skill execution/invocation endpoint**: Skills plugin is discovery-only. Future endpoint for invoking skills with input. *(M6.5)*
- [ ] **Agent execution/invocation endpoint**: Agents plugin is discovery-only. Future endpoint for invoking agents. *(M6.3)*
- [ ] **Running job cancellation**: `Cancel()` only transitions queued jobs to canceled. Running jobs rely on cooperative cancellation (worker checks context). A forceful cancel mechanism (e.g., context cancellation propagated to the worker goroutine) would improve operational control. Known limitation, acceptable for current use. *(M8.5, Phase 8 review)*
- [ ] **Registry/deployment integration bridge**: Connect catalog governance (lifecycle states, promotion bindings) to actual deployment systems (Model Registry, K8s, serving infrastructure). Currently governance is catalog-layer only. *(Phase 8)*
- [ ] **External ecosystem alignment**: Integrate with external AI asset standards and registries (MLflow, OCI artifacts, SLSA/in-toto supply chain metadata, Sigstore signing). *(Phase 8)*
- [ ] **Provenance signing with Sigstore/cosign**: `VerifyingProvenanceExtractor` uses content hashing. Production supply chain security requires cryptographic signing (cosign, in-toto attestations). *(Phase 8)*
- [ ] **Server builder OCI artifact distribution**: Compose custom catalog-servers and distribute as OCI artifacts. Currently only generates source code. *(M9.5)*
- [ ] **Plugin hot-reload from index**: Server builder generates static binaries. Dynamic plugin loading from the index at runtime (without recompilation) is not supported. *(M9.5, M9.6)*
- [ ] **Community plugin contribution workflow**: Plugin index has 3 tiers (built-in, supported, community) but no documented contribution process, review checklist, or automated PR template for community plugin submissions. *(M9.6)*

---

*Last updated: 2026-02-17. Generated from implementation reports M1.1–M1.6, M2.1–M2.6, M3.1–M3.4, M4.1–M4.6, M5.1–M5.9, M6.1–M6.6, M7, M7.1, M8.1–M8.10, M9.1–M9.7.*

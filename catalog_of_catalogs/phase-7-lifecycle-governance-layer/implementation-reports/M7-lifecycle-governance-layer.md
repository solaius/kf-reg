# M7: Lifecycle Governance Layer

**Date**: 2026-02-17
**Updated**: 2026-02-17 (reconciled with M7.1 gap-closing fixes)
**Status**: Complete
**Phase**: Phase 7 - Lifecycle Governance Layer

## Summary
Phase 7 adds enterprise lifecycle governance to the catalog-of-catalogs: a state machine (draft/approved/deprecated/archived), versioning with environment promotion bindings, governance metadata (owner, team, SLA, risk, compliance), approval workflows with YAML policy engine, provider provenance tracking with integrity verification, immutable audit trails with background retention cleanup, and full UI/CLI/BFF integration. Governance is a centralized, plugin-agnostic service at `pkg/catalog/governance/` with routes at `/api/governance/v1alpha1/`.

All governance UI panels are wired into the generic catalog detail page and render conditionally based on capabilities. Conformance tests cover both `mcp` and `agents` plugins. See [M7.1 Gap-Closing Fixes](M7.1_gap-closing-fixes.md) for the follow-up fixes that closed remaining gaps.

## Motivation
- Without lifecycle governance, the catalog is discovery-only. Production AI deployments need ownership, approval gates, promotion controls, and audit trails before assets can be trusted in production environments.
- Satisfies Phase 7 spec requirements: FR1-FR12, AC1-AC6 across 15 specification documents.

## What Changed

### Files Created (47 new files)

#### Governance Core (`pkg/catalog/governance/`)
| File | Purpose |
|------|---------|
| `types.go` | API types: GovernanceOverlay, OwnerInfo, TeamInfo, SLAInfo, RiskInfo, ComplianceInfo, LifecycleInfo, AuditMetadata, enums |
| `models.go` | GORM models: AssetGovernanceRecord, AuditEventRecord, custom JSON types |
| `store.go` | GovernanceStore: Get/Upsert/Delete/List/EnsureExists |
| `audit.go` | AuditStore: Append (immutable), ListByAsset, ListAll (paginated), DeleteOlderThan (M7.1) |
| `config.go` | LoadGovernanceConfig from YAML, DefaultGovernanceConfig |
| `handlers.go` | HTTP handlers: GET/PATCH governance, GET history, combined action dispatch |
| `router.go` | Chi router with NewRouterFull supporting all route groups |
| `lifecycle.go` | LifecycleMachine: ValidateTransition, RequiresApproval, AllowedTransitions |
| `lifecycle_actions.go` | Actions: lifecycle.setState/deprecate/archive/restore with approval gates |
| `approval_models.go` | GORM: ApprovalRequestRecord, ApprovalDecisionRecord |
| `approval_store.go` | ApprovalStore: Create/Get/List/AddDecision/UpdateStatus |
| `approval_policy.go` | ApprovalEvaluator: YAML rule matching, gate evaluation |
| `approval_handlers.go` | HTTP handlers: list/get/submit-decision/cancel approvals |
| `version_models.go` | GORM: AssetVersionRecord, EnvBindingRecord |
| `version_store.go` | VersionStore: CreateVersion/GetVersion/ListVersions |
| `binding_store.go` | BindingStore: GetBinding/SetBinding/ListBindings |
| `version_handlers.go` | HTTP handlers: list/create versions, list/set bindings |
| `promotion_actions.go` | Actions: version.create, promotion.bind/promote/rollback |
| `provenance.go` | ProvenanceExtractor interface, StaticProvenanceExtractor, ContentHashProvenanceExtractor, VerifyingProvenanceExtractor decorator (M7.1), `applyProvenance()` writes integrity fields (M7.1) |
| `capabilities.go` | GovernanceCapabilities, LifecycleCapabilities, VersionCapabilities types |
| `store_test.go` | GovernanceStore CRUD tests |
| `audit_test.go` | AuditStore tests, DeleteOlderThan tests (M7.1) |
| `lifecycle_test.go` | Table-driven lifecycle transition tests |
| `approval_policy_test.go` | Policy matching + gate evaluation tests |
| `version_store_test.go` | Version and binding store tests |
| `promotion_actions_test.go` | Promotion action tests |
| `provenance_test.go` | Provenance extraction tests, VerifyingProvenanceExtractor tests (M7.1) |

#### Config
| File | Purpose |
|------|---------|
| `catalog/config/governance.yaml` | Default governance config: environments, trusted sources, audit retention |
| `catalog/config/approval-policies.yaml` | Default approval policies: agent approval gate, high-risk 2-approver gate |

#### BFF
| File | Purpose |
|------|---------|
| `clients/ui/bff/internal/api/catalog_governance_handler.go` | BFF proxy for all governance API endpoints |

#### CLI
| File | Purpose |
|------|---------|
| `cmd/catalogctl/governance.go` | CLI: governance get/set, versions, promote, rollback, bindings, history |
| `cmd/catalogctl/approvals.go` | CLI: approvals list/get/approve/reject/cancel |

#### UI Frontend
| File | Purpose |
|------|---------|
| `clients/ui/frontend/src/app/types/governance.ts` | TypeScript types for governance API |
| `clients/ui/frontend/src/app/api/governance/service.ts` | API service functions |
| `.../components/GovernancePanel.tsx` | Owner, team, SLA, risk, compliance editing panel |
| `.../components/LifecycleBadge.tsx` | State badge (draft=yellow, approved=green, deprecated=orange, archived=gray) |
| `.../components/VersionsPanel.tsx` | Version list + create |
| `.../components/PromotionPanel.tsx` | Environment bindings + promote/rollback |
| `.../components/ApprovalsPanel.tsx` | Pending approvals + approve/reject |
| `.../components/AuditHistoryPanel.tsx` | Audit event timeline |
| `.../components/ProvenancePanel.tsx` | Source + revision info |

#### Conformance Tests
| File | Purpose |
|------|---------|
| `tests/conformance/governance_test.go` | Governance CRUD conformance |
| `tests/conformance/governance_helpers_test.go` | Shared test helpers |
| `tests/conformance/governance_e2e_test.go` | Full lifecycle E2E flow test |
| `tests/conformance/lifecycle_test.go` | Lifecycle transition validation |
| `tests/conformance/approvals_test.go` | Approval flow conformance |
| `tests/conformance/promotion_test.go` | Promotion + rollback conformance |
| `tests/conformance/provenance_test.go` | Provenance field presence |
| `tests/conformance/audit_test.go` | Audit event emission + pagination |
| `tests/conformance/backward_compat_test.go` | Model catalog API unchanged |

### Files Modified
| File | Change |
|------|--------|
| `pkg/catalog/plugin/server.go` | Added governance/audit/version/binding stores, auto-migration, NewRouterFull wiring, WithGovernanceConfig option |
| `pkg/catalog/plugin/capabilities_types.go` | Added `Governance` field to EntityCapabilities |
| `pkg/catalog/plugin/capabilities_builder.go` | Populated governance capabilities for all entities via `applyGovernanceCaps()` in both V2Provider and fallback paths (M7.1 fix: original code only applied in fallback path, bypassing all V2 plugins) |
| `cmd/catalog-server/main.go` | Load governance config, pass to server, launch `AuditCleanupLoop` (M7.1) |
| `clients/ui/bff/internal/api/app.go` | Register governance BFF routes |
| `clients/ui/frontend/.../PluginEntityDetailPage.tsx` | Wired all 6 governance panels conditionally based on capabilities (M7.1) |

## How It Works

### Lifecycle State Machine
Four states with configurable approval gates:
```
draft ──(approval)──> approved ──> deprecated ──> archived
                         │                          │
                         └──────(approval)──────────┘
                                                    │
                         archived ──(approval)──> deprecated
                         archived ──(approval)──> draft
```

Denied transitions: draft→deprecated, draft→archived, archived→approved.

### Approval Policy Engine
YAML-based rules with selectors and gates:
```yaml
policies:
  - id: high-risk-gate
    selector:
      risk_levels: [high, critical]
    gates:
      - action: "lifecycle.setState"
        approvalsRequired: 2
```

When a gated action is requested, the system creates a pending `ApprovalRequest` (HTTP 202) instead of executing immediately. When the approval threshold is met, the action auto-executes.

### Versioning + Promotion
Immutable version snapshots capture governance state at a point in time. Environment bindings (dev/stage/prod) map to specific versions. Lifecycle enforcement: archived cannot bind; draft cannot bind to stage/prod.

### Action Dispatch
Combined handler dispatches lifecycle actions (`lifecycle.*`) and promotion actions (`version.*`, `promotion.*`) through a single endpoint:
```
POST /api/governance/v1alpha1/assets/{plugin}/{kind}/{name}/actions/{action}
```

## Key Design Decisions
| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Centralized governance (not per-plugin) | All asset types get identical governance; no plugin code changes needed | Per-plugin governance hooks (rejected: too much duplication) |
| YAML-only approval policies | Covers 90% of use cases, simple to configure | OPA/Rego (deferred to Phase 8) |
| Combined action handler | Single endpoint for all governance actions | Separate lifecycle and promotion endpoints (rejected: inconsistent UX) |
| Freeform version labels | Different asset types don't all fit SemVer | Strict SemVer enforcement (rejected: too restrictive) |
| Configurable environments | Organizations have different promotion paths | Fixed dev/stage/prod (rejected: too rigid) |
| 90-day audit retention default with daily cleanup | Balances audit trail with storage; `AuditCleanupLoop` runs daily in background, config-gated by `AuditRetention.Days` | Unlimited (rejected: unbounded growth); event-driven cleanup (rejected: more complex, same outcome) |

## Testing
- **Unit tests**: 35+ tests across 7 test files in `pkg/catalog/governance/` (includes M7.1 additions for provenance integrity and audit deletion)
- **Conformance tests**: 9 test files covering governance CRUD, lifecycle, approvals, promotion, provenance, audit, backward compatibility, and full E2E flow
- **Multi-plugin coverage** (M7.1): All governance conformance tests iterate over both `mcp` and `agents` plugins via `governanceTestPlugins()`
- **Live stack proof run** (M7.1): All 8 governance test suites pass across both plugins:
  - `TestGovernanceCRUD` — PASS (mcp + agents, 8 subtests)
  - `TestGovernanceLifecycle` — PASS (mcp + agents, 18 subtests)
  - `TestGovernancePromotion` — PASS (mcp + agents, 18 subtests)
  - `TestGovernanceApprovals` — PASS (mcp + agents)
  - `TestGovernancePolicies` — PASS
  - `TestGovernanceAudit` — PASS (mcp + agents, 10 subtests)
  - `TestGovernanceProvenance` — PASS (mcp + agents, 6 subtests)
  - `TestGovernanceE2EFullLifecycle` — PASS (mcp + agents)
- **Backward compatibility**: All existing tests pass (8 plugin endpoints, health endpoints, pagination)

```bash
# Unit tests
go test ./pkg/catalog/governance/... -v -count=1

# Conformance (requires running stack)
CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/conformance/... -v -count=1 -run "TestGovernance"
```

## Verification
```bash
# Unit tests
go test ./pkg/catalog/governance/... -v -count=1

# Start fresh stack
docker compose -f docker-compose.catalog.yaml down -v
docker compose -f docker-compose.catalog.yaml up --build -d
curl -s http://localhost:8080/readyz

# Verify governance capabilities are advertised (should show governance.supported=true)
curl -s http://localhost:8080/api/plugins/mcp/capabilities | python3 -c "
import json,sys; d=json.load(sys.stdin)
for e in d.get('entities',[]):
    print(f'{e[\"kind\"]}: governance.supported={e.get(\"governance\",{}).get(\"supported\")}')"

# Governance CRUD
curl -s -X PATCH http://localhost:8080/api/governance/v1alpha1/assets/agents/Agent/my-agent \
  -H 'Content-Type: application/json' \
  -d '{"owner":{"principal":"alice","email":"alice@example.com"},"risk":{"level":"medium"}}' | python3 -m json.tool

# Lifecycle transition
curl -s -X POST http://localhost:8080/api/governance/v1alpha1/assets/agents/Agent/my-agent/actions/lifecycle.setState \
  -H 'Content-Type: application/json' -H 'X-User-Principal: operator' \
  -d '{"params":{"state":"approved","reason":"reviewed"}}' | python3 -m json.tool

# Full governance conformance (both mcp + agents plugins)
CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/conformance/... -v -count=1 -run "TestGovernance"

# Full suite excluding pre-existing Phase 5 action failures
CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/conformance/... -v -count=1 \
  -run "TestGovernance|TestBackwardCompat|TestCapabilities|TestPluginNames|TestBasePaths|TestPagination"

# UI verification: navigate to http://localhost:9000/catalog/mcp/mcpservers/filesystem
# Governance panels (Lifecycle, Versions, Provenance, Promotion, Approvals, Audit History)
# should render below the entity detail fields.
```

## Dependencies & Impact
- **Enables**: Production-grade asset lifecycle management, multi-environment promotion, audit compliance
- **Depends on**: Phase 5 universal asset framework, Phase 6 plugin ecosystem
- **Backward compatibility**: No changes to existing model/MCP/knowledge catalog APIs; governance is additive
- **Follow-up**: [M7.1 Gap-Closing Fixes](M7.1_gap-closing-fixes.md) resolved 5 gaps identified in code review (provenance integrity, audit cleanup, UI wiring, multi-plugin conformance, test stability)

## Open Items

### Resolved in M7.1
The following items were listed as open in the original M7 report and have since been resolved:

- [x] ~~Signature verification for provenance~~ — `VerifyingProvenanceExtractor` decorator implemented, `applyProvenance()` writes integrity fields to DB. **Note**: the decorator is available as a composable hook but is not wired into the default server pipeline; callers must explicitly wrap their extractor. The mechanism and persistence are complete; default-on activation is deferred.
- [x] ~~Audit retention background cleanup job~~ — `AuditCleanupLoop` runs daily in background (`main.go:203`), config-gated by `governanceConfig.AuditRetention.Days` (default: 90). Disabled automatically if days <= 0.
- [x] ~~Governance panels integration into generic detail page~~ — All 6 panels (GovernancePanel, VersionsPanel, ProvenancePanel, PromotionPanel, ApprovalsPanel, AuditHistoryPanel) wired into `PluginEntityDetailPage.tsx`, conditionally rendered based on capabilities. Fully generic (no plugin-specific code).
- [x] ~~Capabilities builder bypass for V2 plugins~~ — `BuildCapabilitiesV2()` now calls `applyGovernanceCaps()` in both the V2Provider early-return path and the fallback builder path, ensuring governance capabilities are advertised for all plugins.

### Still Open
- [ ] TLS in Docker Compose for governance endpoints (same as existing TODO)
- [ ] OPA/Rego adapter for approval policies (Phase 8)
- [ ] Wire `VerifyingProvenanceExtractor` as default-on in the server pipeline (mechanism exists, not yet activated)
- [ ] `TestConformance` action tests (tag/annotate/deprecate) are pre-existing Phase 5 failures — entity action endpoint format may need updating in a future phase

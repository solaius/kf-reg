# Delivery Plan and Milestones

_Last updated: 2026-02-16_

## Milestone breakdown

### M3.1 Persistent config and reconciliation
Deliver:
- ConfigStore interface
- FileConfigStore implementation
- KubernetesConfigMapStore implementation (if target runtime includes cluster mode)
- Concurrency and reconcile loop
- Source-level refresh for YAML provider
DoD and AC in 03_persistent_config_spec.md

### M3.2 Real wiring, no mocks in default path
Deliver:
- BFF real mode default
- remove placeholder responses in real mode
- UI uses real data in E2E
DoD and AC in 05_bff_real_wiring_no_mocks.md and 06_ui_operational_hardening.md

### M3.3 MCP real data and provider validation
Deliver:
- YAML source with 6 entries
- refresh updates list
- diagnostics on parse errors
DoD and AC in 08_mcp_catalog_real_data_and_providers.md

### M3.4 UI operational hardening
Deliver:
- sources status fields live
- diagnostics tab
- role gating
DoD and AC in 06_ui_operational_hardening.md

### M3.5 CLI v2
Deliver:
- full command set
- role header support
- json output matches OpenAPI
DoD and AC in 07_cli_v2_management_and_mcp.md

### M3.6 Tests, CI, docs
Deliver:
- test layers
- `make e2e` and smoke tests
DoD and AC in 09_e2e_runtime_and_dev_workflow.md and 10_tests_quality_gates.md

## Release readiness checklist
- All acceptance criteria satisfied
- Demo script recorded
- Known limitations documented
- No open critical bugs

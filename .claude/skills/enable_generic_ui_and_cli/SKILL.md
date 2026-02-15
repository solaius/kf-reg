# Skill: enable_generic_ui_and_cli

## When to use
Use this skill when wiring UI and CLI to support new plugins with minimal per-type code.

## Repo rules to obey
- Use existing UI and BFF patterns and routing conventions
- Keep new UI behavior generic where possible
- Prefer additive API surface and do not break existing endpoints
- Keep tests and CI green

## Steps
1. Plugin discovery contract
   - Ensure /api/plugins exposes enough metadata for discovery (name, base path, version, enabled, health)
2. Shared schema contract
   - Ensure shared schemas provide consistent fields for generic rendering (id, name, description, timestamps, customProperties)
3. BFF integration
   - Follow BFF patterns (Go, chi router) and keep handlers thin
   - Add a generic proxy or typed client wrapper that can route to plugin base paths
4. Frontend integration
   - Implement generic list and detail views that can render any plugin that uses BaseResource and BaseResourceList
   - Allow per-plugin column overrides only via metadata, not hardcoded UI paths
5. CLI integration
   - Add commands that work for any plugin by name (list, get, sources, health)
   - Keep output stable and script-friendly (JSON by default, optional table output)
6. Tests
   - Add e2e tests for basic list and get paths that can run against at least one non-model plugin

## Validation
- make lint
- make test
- Any UI build/test targets required by the repo

## Output
- The generic UI and CLI contract you implemented
- A checklist for adding a new plugin without UI rewrites
- Commands run and results

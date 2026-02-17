# Skill: enable_generic_ui_and_cli

## When to use
Use this skill when wiring UI and CLI to support new plugins with minimal per-type code, or when verifying that a new plugin appears in UI/CLI without code changes.

## Repo rules to obey
- Use existing UI and BFF patterns and routing conventions
- Keep new UI behavior generic where possible
- Prefer additive API surface and do not break existing endpoints
- Keep tests and CI green
- NO plugin-specific branching in generic components (`if pluginName === 'mcp'` is forbidden)

## Steps
1. Plugin discovery contract
   - Ensure /api/plugins exposes V2 capabilities with entities, sources, actions, field metadata
   - Ensure /api/plugins/{name}/capabilities returns full `PluginCapabilitiesV2`
2. BFF capabilities proxy
   - `GET /api/v1/catalog/:plugin/capabilities` - proxy to catalog server
   - Generic entity proxy: `GET/POST /api/v1/catalog/:plugin/:entity/*` - route to plugin endpoints
   - Keep handlers thin; use capabilities to build target URLs dynamically
3. Frontend generic components (capabilities-driven)
   - `CatalogContext` loads all plugin capabilities on mount
   - `GenericListView` renders table columns from capabilities `columns` field
   - `GenericFilterBar` renders filters from capabilities `filterFields`
   - `GenericDetailView` renders detail from capabilities `detailFields`
   - `GenericActionBar` renders action buttons from capabilities `actions`
   - `GenericActionDialog` renders action parameter forms
   - Navigation built dynamically from discovered plugins + entities
   - Routes: `/catalog/:plugin/:entity` (list), `/catalog/:plugin/:entity/:name` (detail)
4. CLI v2 (catalogctl) capabilities-driven
   - `catalogctl plugins list` - list all plugins
   - `catalogctl <plugin> <entity> list` - capabilities-driven table output
   - `catalogctl <plugin> <entity> get <name>` - detail output (json/yaml/table)
   - `catalogctl <plugin> sources list/refresh` - source management
   - `catalogctl <plugin> <entity> action <id>` - execute actions
   - Commands auto-discovered from capabilities, no hardcoded plugin names
5. Tests
   - Playwright tests for generic UI: verify all plugins render list/detail/actions
   - CLI golden tests: verify output format for all plugins
   - Conformance suite: verify all plugins pass universal contract

## Zero-change verification for new plugins
When adding a new plugin, run this checklist to prove zero UI/CLI changes needed:
```bash
# Verify no frontend changes
git diff --name-only HEAD -- clients/ui/frontend/
# Expected: empty

# Verify no CLI changes
git diff --name-only HEAD -- cmd/catalogctl/
# Expected: empty

# Verify plugin appears in API
curl -s http://localhost:8080/api/plugins | python3 -c "import sys,json; plugins=json.load(sys.stdin); print([p['name'] for p in plugins])"

# Verify plugin has capabilities
curl -s http://localhost:8080/api/plugins/<name>/capabilities | python3 -m json.tool

# Verify UI nav shows plugin (Playwright MCP)
# Navigate to http://localhost:9000 and check nav items

# Verify CLI shows plugin
./catalogctl plugins list
./catalogctl <name> <entity> list
```

## Validation
- `npm run build` in frontend directory succeeds
- `go build ./cmd/catalogctl/` succeeds
- All generic components render for all registered plugins
- No plugin-specific imports or conditionals in generic components

## Output
- The generic UI and CLI contract you implemented
- A checklist for adding a new plugin without UI rewrites
- Commands run and results
- Playwright screenshots showing new plugin in UI

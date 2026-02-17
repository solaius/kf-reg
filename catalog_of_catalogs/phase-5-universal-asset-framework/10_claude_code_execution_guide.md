# Claude Code Execution Guide (Phase 5)

## Objective
Implement Phase 5 in a way that is:
- automated (generator-driven)
- repeatable (conformance suite)
- additive (no breaking changes)
- proven (knowledge-sources plugin appears with zero UI/CLI changes)

## Recommended order
1) Capabilities schema + endpoints (server)
2) Universal asset contract (common schemas + projections)
3) Action model endpoints + baseline actions implemented in shared framework
4) Update model + mcp plugins (capabilities + baseline actions)
5) UI generic components library (capabilities-driven)
6) CLI v2 (capabilities-driven)
7) Knowledge-sources plugin (scaffold + sample data)
8) Conformance suite + CI verification

## Hard requirements
- No breaking changes to model catalog API paths
- Avoid UI/CLI plugin-specific branching
- “New plugin appears” behavior must be capabilities-driven

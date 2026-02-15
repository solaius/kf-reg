# Personas and User Journeys

## Persona: AI Engineer

Goals

- Find the right asset quickly (model, MCP server, dataset, prompt template, evaluation, agent)
- Compare options using consistent metadata and filters
- Pull a stable reference for automation (IDs, versions, source, artifact URLs)
- Export or hand off asset references to downstream workflows

Primary journeys

- Discover
  - Pick asset type (plugin)
  - Search and filter
  - Open details and artifacts
  - Copy reference or download metadata

- Validate fit
  - Inspect required inputs and outputs
  - Inspect compatibility metadata (tags, frameworks, runtimes, protocols)
  - Inspect provenance and licensing metadata

- Track changes
  - See source and version
  - See last refresh time and ingestion status
  - Detect deprecations

## Persona: Ops for AI

Goals

- Add and manage catalog sources safely
- Control who can change sources and who can only view assets
- Monitor health and ingestion to keep catalog trustworthy
- Troubleshoot failures quickly

Primary journeys

- Configure
  - Add a new source definition
  - Validate the config
  - Enable the source
  - Trigger refresh

- Operate
  - Check plugin health and ingestion status
  - Inspect errors and last successful refresh
  - Roll back a bad config change

- Govern
  - Grant view access broadly
  - Restrict change access to a small group
  - Ensure auditability of changes

## Success metrics

User facing

- Time to first useful search result for a new user
- Time to add a new catalog source end to end
- Percentage of catalog sources in healthy state

Operational

- Mean time to detect ingestion failures
- Mean time to diagnose and recover from ingestion failures
- Number of breaking UI or CLI changes per release

Developer experience

- Time to scaffold and integrate a new plugin to UI and CLI
- Volume of hand edited generated code required after schema changes

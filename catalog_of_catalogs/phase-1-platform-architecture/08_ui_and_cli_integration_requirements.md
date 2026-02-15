# UI and CLI integration requirements

## UI expectations
The UI should be able to:
- Discover available plugins and their base paths
- Present a navigation pattern per asset type (tabs or left nav)
- Show a list view with:
  - search and filter inputs
  - pagination controls
  - common columns (name, description, source, updated)
- Show a detail view with:
  - core fields
  - metadata properties
  - artifacts where applicable
  - a copyable stable reference

UI should not require hardcoded per-plugin UI logic for basic list and detail views
- It may allow plugin-specific enhancements but must work generically by default

## CLI expectations
The CLI should support:
- Listing plugins
- Listing entities for a given plugin
- Getting an entity
- Searching with filterQuery
- Listing sources and inspecting source status
- Validating sources.yaml structure

CLI should not embed knowledge of each asset type beyond the plugin name and its API base path.

## Contract requirements to enable generic UI and CLI
- Plugins endpoint returns enough metadata to discover routes and entity kinds
- OpenAPI spec is merged and available for client generation
- Shared schemas ensure consistent fields for generic rendering


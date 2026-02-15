# Data model and metadata conventions

## Goals
- Make asset metadata flexible without requiring frequent DB migrations
- Maintain consistent metadata shape across plugins
- Support rich search and filtering

## Required fields for all entities
- id
- name
- description
- create and update timestamps
- custom properties or metadata map for extensibility
- source provenance fields sufficient to trace where data came from

## Custom properties
- Store dynamic attributes in a flexible structure (row-based or JSON) consistent with existing repo patterns
- Ensure types are preserved (string, bool, number, timestamp, list where supported)

## Provenance
Every entity should capture:
- source id
- source type
- last refresh time
- optional origin uri

## References
- Entities should emit a stable reference string for cross-asset linking
- References should be resolvable back to an API path

## Filtering
- Define mappings from queryable fields to underlying DB representation
- Be explicit about which custom properties are filterable by default


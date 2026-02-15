# Open questions and decision log

## Open questions
1. What is the minimum plugin metadata returned by /api/plugins to enable generic UI and CLI
2. How should reference strings be standardized and versioned
3. How should custom loader logic be supported in the generated scaffolding
4. How to handle disabling subtypes or capabilities per plugin without breaking schema generation
5. Which fields should be indexed by default for performance across plugins
6. How to represent source authentication and secrets consistently in sources.yaml

## Decisions to record as they are made
- Plugin naming and config key mapping rules
- Schema naming and collision avoidance strategy in merged OpenAPI
- Default provider set supported by the framework
- Common metadata fields and their canonical types
- Cross-asset reference convention

## Decision log template
- Date
- Decision
- Rationale
- Alternatives considered
- Implications


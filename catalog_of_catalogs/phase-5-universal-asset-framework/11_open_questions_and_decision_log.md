# Open Questions and Decision Log (Phase 5)

## Decisions required early
1) Capability transport
- Catalog-server direct vs BFF aggregate/cache

2) Action endpoint shape
- :action endpoints (action id in body) vs dedicated endpoints

3) Where edits persist
- Overlay store vs mutating source YAML

## Recommendation defaults
- Capabilities cached in BFF (passthrough allowed in dev)
- :action endpoints for uniformity
- Overlay store for user edits (do not mutate source catalogs automatically)

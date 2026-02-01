# Remaining Work

This document outlines the work remaining to complete the MCP Catalog feature.

## Current Status

**Branch:** `feature/mcp-catalog`
**Status:** In development, not yet merged to main
**Last Update:** 4 commits ahead of main

## Outstanding Tasks

### 1. Branch Merge to Main

The feature branch needs to be merged to the main branch:

```bash
# Commits on feature/mcp-catalog not in main:
21a39286 - Main MCP implementation (99 files, +13,931/-2,471)
8d7b5096 - Free-form keyword search
08c62fae - Add MCP schemas to source OpenAPI
933b2bd9 - Regenerate OpenAPI client SDK
```

**Action Required:**
- [ ] Create pull request from `feature/mcp-catalog` to `main`
- [ ] Address any review comments
- [ ] Ensure CI/CD passes
- [ ] Merge after approval

### 2. Integration Testing

Additional integration tests are needed:

- [ ] End-to-end API tests for MCP endpoints
- [ ] BFF integration tests with mock catalog service
- [ ] Frontend E2E tests with Cypress
- [ ] Hot-reload behavior testing
- [ ] Multi-source merge conflict testing

### 3. Documentation Updates

- [ ] Update main README with MCP Catalog information
- [ ] Add MCP configuration to deployment guides
- [ ] Create user-facing documentation for MCP discovery
- [ ] Add API reference documentation

### 4. Performance Optimization

- [ ] Add database indexes for common query patterns
- [ ] Implement caching for filter options
- [ ] Optimize text search queries
- [ ] Add pagination support for large server lists

### 5. UI Enhancements

- [ ] Add server comparison feature
- [ ] Implement favorites/bookmarks
- [ ] Add tool parameter validation display
- [ ] Improve mobile responsiveness
- [ ] Add keyboard navigation

### 6. Additional Provider Support

Future provider types to consider:

- [ ] **GitHub Provider**: Discover MCP servers from GitHub repositories
- [ ] **HuggingFace Provider**: Discover from HuggingFace Hub
- [ ] **Remote Discovery**: Dynamic discovery from MCP server endpoints

### 7. Security Enhancements

- [ ] Add RBAC for MCP source management
- [ ] Implement signature verification for artifacts
- [ ] Add audit logging for server access
- [ ] Support for private OCI registries

## Known Issues

### 1. Text Search Limitations

**Issue:** Text search only matches exact substrings in name and description.

**Workaround:** Use filter queries for more precise matching.

**Future Fix:** Implement full-text search with relevance scoring.

### 2. License Filter Display

**Issue:** When filtering by license, the filter uses SPDX identifiers but displays user-friendly names.

**Status:** Working as designed, but may confuse users.

**Future Improvement:** Add tooltip explaining the mapping.

### 3. Hot-Reload Timing

**Issue:** On Kubernetes, ConfigMap updates may not trigger hot-reload immediately.

**Workaround:** Use a sidecar like `reloader` or restart pods.

**Future Fix:** Implement periodic polling as fallback.

### 4. Large Tool Lists

**Issue:** Servers with many tools may have slow page loads.

**Workaround:** None currently.

**Future Fix:** Lazy-load tool details or paginate tool list.

## Technical Debt

### 1. Property Storage

**Issue:** Tools and other complex objects are stored as JSON strings in property values.

**Impact:** Limited query capability for tool-level filtering.

**Future Improvement:** Consider separate `mcp_tool` table for first-class tool storage.

### 2. Test Coverage

**Current Coverage:**
- Unit tests: ~75%
- Integration tests: ~50%
- E2E tests: ~30%

**Target Coverage:**
- Unit tests: 90%
- Integration tests: 70%
- E2E tests: 60%

### 3. Error Handling

**Issue:** Some error cases return generic "server error" messages.

**Future Improvement:** Implement detailed error types with actionable messages.

## Feature Backlog

### High Priority

| Feature | Description | Effort |
|---------|-------------|--------|
| Search improvements | Full-text search with ranking | Medium |
| Server import | Import from MCP registry | Large |
| Tool execution | Direct tool invocation from UI | Large |

### Medium Priority

| Feature | Description | Effort |
|---------|-------------|--------|
| Categories | Server categorization system | Medium |
| Version history | Track server version changes | Medium |
| Health checks | Monitor remote server availability | Medium |

### Low Priority

| Feature | Description | Effort |
|---------|-------------|--------|
| Server ratings | User ratings and reviews | Small |
| Export | Export server configs | Small |
| Notifications | Alert on server updates | Medium |

## Migration Considerations

When merging to main, consider:

1. **Database Migration:**
   - MCP tables will be created automatically by GORM
   - No manual migration scripts required
   - Existing data is not affected

2. **Configuration:**
   - Existing catalogs (models) continue to work
   - MCP sources require new configuration files
   - No breaking changes to existing APIs

3. **Backward Compatibility:**
   - `/sources` API now returns both model and MCP sources
   - New `assetType` field distinguishes source types
   - Existing clients unaffected

## Timeline Estimate

| Task | Estimated Effort |
|------|------------------|
| Branch merge and review | 1-2 days |
| Integration testing | 3-5 days |
| Documentation | 2-3 days |
| Performance optimization | 2-3 days |
| Bug fixes from testing | 2-3 days |
| **Total** | **10-16 days** |

---

[Back to MCP Catalog Index](./README.md) | [Previous: Data Models](./data-models.md) | [Next: Step-by-Step Creation](./step-by-step-creation.md)

# Action Framework

## Overview

The action framework provides a uniform mechanism for mutating entities and
sources across all catalog plugins. Rather than each plugin exposing bespoke
mutation endpoints, every plugin that implements the `ActionProvider` interface
gains a pair of generic `:action` endpoints (one scoped to sources, one to
assets). The framework handles request parsing, action validation, dry-run
gating, and response formatting so that plugins only need to supply the
business logic for each action.

Three **builtin actions** (tag, annotate, deprecate) are provided by the
framework itself. They persist metadata through the `OverlayStore`, a
shared database table that stores user-applied overlays without mutating
upstream source data. Plugins can expose these builtins, add custom actions,
or combine both.

**Location:** `pkg/catalog/plugin/`

```
                        POST .../management/entities/{name}:action
                        POST .../management/sources/{id}:action
                                       |
                                       v
                             +-------------------+
                             |  actionHandler()  |
                             +-------------------+
                                       |
                       +---------------+---------------+
                       |                               |
                  Plugin implements            Plugin does not
                  ActionProvider?              implement interface
                       |                               |
                       v                               v
                 Parse request body              501 Not Implemented
                       |
                       v
               Lookup action in ListActions()
                       |
               +-------+-------+
               |               |
           Found           Not found
               |               |
               v               v
       Check dry-run       400 Unknown
       support             action
               |
               v
       Call HandleAction()
               |
       +-------+-------+
       |               |
    Success          Error
       |               |
       v               v
    200 OK         500 Internal
    ActionResult   Server Error
```

## ActionProvider Interface

Plugins opt into the action framework by implementing `ActionProvider`:

```go
// pkg/catalog/plugin/action_types.go
type ActionProvider interface {
    // HandleAction executes an action. scope is "source" or "asset",
    // targetID is the source ID or entity name.
    HandleAction(ctx context.Context, scope ActionScope, targetID string, req ActionRequest) (*ActionResult, error)

    // ListActions returns the actions available for the given scope.
    ListActions(scope ActionScope) []ActionDefinition
}
```

`ListActions` is called at request time to validate that the requested action
exists for the given scope and to check dry-run eligibility. It is also
exposed via read-only discovery endpoints (see Action Discovery below).

## ActionScope

Actions are scoped to either a **source** or an **asset**:

```go
// pkg/catalog/plugin/action_types.go
type ActionScope string

const (
    ActionScopeSource ActionScope = "source"
    ActionScopeAsset  ActionScope = "asset"
)
```

The scope determines which URL parameter supplies the target identifier:

| Scope    | URL Parameter   | Example Target ID    |
|----------|-----------------|----------------------|
| `source` | `{sourceId}`    | `my-hf-source`       |
| `asset`  | `{entityName}`  | `filesystem`         |

## ActionRequest and ActionResult

### ActionRequest

The request body for every `:action` endpoint:

```go
// pkg/catalog/plugin/action_types.go
type ActionRequest struct {
    Action string         `json:"action"`           // Required. Action ID to execute.
    DryRun bool           `json:"dryRun,omitempty"`  // Preview without side effects.
    Params map[string]any `json:"params,omitempty"`  // Action-specific parameters.
}
```

### ActionResult

The response returned from every successful action invocation:

```go
// pkg/catalog/plugin/action_types.go
type ActionResult struct {
    Action  string         `json:"action"`            // Echoes the requested action ID.
    Status  string         `json:"status"`            // "completed", "dry-run", or "error"
    Message string         `json:"message,omitempty"` // Human-readable summary.
    Data    map[string]any `json:"data,omitempty"`    // Action-specific output payload.
}
```

### ActionDefinition

Each action advertised by a plugin is described with:

```go
// pkg/catalog/plugin/capabilities_types.go
type ActionDefinition struct {
    ID             string `json:"id"`
    DisplayName    string `json:"displayName"`
    Description    string `json:"description"`
    Scope          string `json:"scope"`          // "source" or "asset"
    SupportsDryRun bool   `json:"supportsDryRun"`
    Idempotent     bool   `json:"idempotent"`
    Destructive    bool   `json:"destructive,omitempty"`
}
```

## Builtin Actions

### BuiltinActionHandler

The framework ships a `BuiltinActionHandler` that implements tag, annotate,
and deprecate using the `OverlayStore`. Plugins instantiate it during `Init`:

```go
// pkg/catalog/plugin/builtin_actions.go
handler := NewBuiltinActionHandler(overlayStore, pluginName, entityKind)
```

The handler exposes three methods that a plugin's `HandleAction` can delegate
to: `HandleTag`, `HandleAnnotate`, and `HandleDeprecate`. The companion
function `BuiltinActionDefinitions()` returns the matching `ActionDefinition`
slice for inclusion in `ListActions`.

### Builtin Action Table

| Action ID   | Scope | Params                                  | Dry-Run | Idempotent | Description                            |
|-------------|-------|-----------------------------------------|---------|------------|----------------------------------------|
| `tag`       | asset | `tags` ([]string)                       | Yes     | Yes        | Add or replace tags on an entity       |
| `annotate`  | asset | `annotations` (map[string]string)       | Yes     | Yes        | Add or update annotations on an entity |
| `deprecate` | asset | `phase` (string, optional, default `"deprecated"`) | Yes     | Yes        | Set lifecycle phase to deprecated      |

### OverlayStore Persistence

Builtin actions never mutate the upstream source data. Instead they write to
the `catalog_overlays` database table via `OverlayStore`:

```
+-------------------------------------------------------------------+
| catalog_overlays                                                  |
+-------------------------------------------------------------------+
| plugin_name  | entity_kind | entity_uid | tags | annotations     |
|              |             |            |      | labels           |
|              |             |            |      | lifecycle_phase  |
|              |             |            |      | updated_at       |
+-------------------------------------------------------------------+
  PK: (plugin_name, entity_kind, entity_uid)
```

Each builtin action follows the same pattern:

1. Look up the existing `OverlayRecord` (or create an empty one)
2. Mutate the relevant field (tags, annotations, or lifecycle)
3. Upsert the record back via `OverlayStore.Upsert`

The overlay is merged with source data at read time so that `GET` responses
reflect both the upstream value and any user-applied overlays.

## Action Routing

The `managementRouter` function in `management_handlers.go` mounts action
endpoints when the plugin implements `ActionProvider`:

```go
// pkg/catalog/plugin/management_handlers.go (simplified)
if _, ok := p.(ActionProvider); ok {
    // Source-scoped actions.
    r.Post("/sources/{sourceId}:action",
        RequireRole(RoleOperator, roleExtractor)(
            http.HandlerFunc(actionHandler(p, ActionScopeSource)),
        ).ServeHTTP)

    // Asset-scoped actions.
    r.Post("/entities/{entityName}:action",
        RequireRole(RoleOperator, roleExtractor)(
            http.HandlerFunc(actionHandler(p, ActionScopeAsset)),
        ).ServeHTTP)

    // Action discovery (read-only, all roles).
    r.Get("/actions/source", actionsListHandler(p, ActionScopeSource))
    r.Get("/actions/asset",  actionsListHandler(p, ActionScopeAsset))
}
```

The management router is mounted at `{basePath}/management`, so the full
paths are:

```
POST {basePath}/management/sources/{sourceId}:action     (source scope)
POST {basePath}/management/entities/{entityName}:action   (asset scope)
GET  {basePath}/management/actions/source                 (discover source actions)
GET  {basePath}/management/actions/asset                  (discover asset actions)
```

### actionHandler Processing Steps

The `actionHandler` function in `action_handler.go` performs the following
sequence for every incoming action request:

```
1. Assert plugin implements ActionProvider         -> 501 if not
2. Extract targetID from URL:
     source scope -> chi.URLParam(r, "sourceId")
     asset scope  -> chi.URLParam(r, "entityName")
3. Decode ActionRequest from JSON body             -> 400 on parse error
4. Verify req.Action is non-empty                  -> 400 if missing
5. Call ListActions(scope) and find matching ID     -> 400 if unknown
6. If req.DryRun && !found.SupportsDryRun          -> 400 if unsupported
7. Call HandleAction(ctx, scope, targetID, req)     -> 500 on error
8. Return ActionResult as JSON with 200 OK
```

## Dry Run Support

Any action whose `ActionDefinition.SupportsDryRun` is `true` can be invoked
with `"dryRun": true` in the request body. The framework validates this at
the routing layer before calling `HandleAction`.

When dry-run is active:

- The action handler returns a preview of what **would** happen
- No state is mutated (no overlay writes, no source changes)
- The response `status` field is set to `"dry-run"` instead of `"completed"`
- The `data` field contains the computed values that would be applied

All three builtin actions support dry-run. Custom plugin actions can opt in
by setting `SupportsDryRun: true` in their `ActionDefinition` and branching
on `req.DryRun` inside `HandleAction`.

## API Endpoint Patterns

| Method | Path | Scope | Auth | Description |
|--------|------|-------|------|-------------|
| `POST` | `{basePath}/management/sources/{sourceId}:action` | source | Operator | Execute a source-scoped action |
| `POST` | `{basePath}/management/entities/{entityName}:action` | asset | Operator | Execute an asset-scoped action |
| `GET`  | `{basePath}/management/actions/source` | source | Any | List available source-scoped actions |
| `GET`  | `{basePath}/management/actions/asset` | asset | Any | List available asset-scoped actions |

The `:action` suffix is a Google-style custom method on the resource URL,
not a separate path segment. Chi routes it as a literal suffix on the
`{sourceId}` or `{entityName}` parameter.

## Example

### Tag an entity (dry-run)

```bash
curl -X POST http://localhost:8080/api/mcp_catalog/v1alpha1/management/entities/filesystem:action \
  -H 'Content-Type: application/json' \
  -d '{"action":"tag","dryRun":true,"params":{"tags":["production","verified"]}}'
```

Response:

```json
{
  "action": "tag",
  "status": "dry-run",
  "message": "would set 2 tags on filesystem",
  "data": {
    "tags": ["production", "verified"]
  }
}
```

### Deprecate an entity

```bash
curl -X POST http://localhost:8080/api/mcp_catalog/v1alpha1/management/entities/old-server:action \
  -H 'Content-Type: application/json' \
  -d '{"action":"deprecate"}'
```

Response:

```json
{
  "action": "deprecate",
  "status": "completed",
  "message": "set lifecycle of old-server to \"deprecated\"",
  "data": {
    "lifecycle": "deprecated"
  }
}
```

### Discover available asset actions

```bash
curl -s http://localhost:8080/api/mcp_catalog/v1alpha1/management/actions/asset
```

Response:

```json
{
  "actions": [
    {
      "id": "tag",
      "displayName": "Tag",
      "description": "Add or replace tags on an entity",
      "scope": "asset",
      "supportsDryRun": true,
      "idempotent": true
    },
    {
      "id": "annotate",
      "displayName": "Annotate",
      "description": "Add or update annotations on an entity",
      "scope": "asset",
      "supportsDryRun": true,
      "idempotent": true
    },
    {
      "id": "deprecate",
      "displayName": "Deprecate",
      "description": "Mark an entity as deprecated",
      "scope": "asset",
      "supportsDryRun": true,
      "idempotent": true
    }
  ],
  "count": 3
}
```

## Key Files

| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/action_types.go` | `ActionScope`, `ActionRequest`, `ActionResult`, `ActionProvider` interface |
| `pkg/catalog/plugin/action_handler.go` | `actionHandler` HTTP handler and `actionsListHandler` discovery handler |
| `pkg/catalog/plugin/builtin_actions.go` | `BuiltinActionHandler` with tag, annotate, deprecate; helper extractors |
| `pkg/catalog/plugin/capabilities_types.go` | `ActionDefinition` struct |
| `pkg/catalog/plugin/overlay_store.go` | `OverlayStore` and `OverlayRecord` for metadata persistence |
| `pkg/catalog/plugin/management_handlers.go` | `managementRouter` that mounts `:action` and discovery routes |

---

[Back to Universal Assets](./README.md) | [Prev: Asset Contract](./asset-contract.md)

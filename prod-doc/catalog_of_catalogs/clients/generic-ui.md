# Generic UI Components

## Overview

The generic UI layer provides capabilities-driven React components that render
any catalog plugin with zero plugin-specific code. Built with PatternFly 6.4,
React 18, and TypeScript, the system reads V2 capabilities from each plugin at
runtime and dynamically constructs navigation, list views, detail views,
filter bars, and action dialogs.

The design principle is simple: the backend plugin declares what its entities
look like (columns, detail fields, filters, actions) through the V2 capabilities
contract, and the frontend renders them generically. Adding a new plugin to the
catalog server causes it to appear in the UI automatically -- no pull request to
the frontend is required.

**Location:** `clients/ui/frontend/src/app/pages/genericCatalog/`

```
+------------------------------------------------------------+
|                    CatalogContextProvider                   |
|   Fetches /api/v1/catalog/plugins at mount time            |
|   Fetches /api/catalog/{plugin}/capabilities per plugin    |
|   Stores plugin list + V2 capabilities in React context    |
+--+---------------------------------------------------------+
   |
   |  React Router (parameterized routes)
   |
   +-->  /catalog                    -->  CatalogHomePage
   +-->  /catalog/:pluginName        -->  (redirect to first entity)
   +-->  /catalog/:plugin/:entity    -->  PluginEntityListPage
   +-->  /catalog/:plugin/:entity/:name  -->  PluginEntityDetailPage
```

---

## CatalogContextProvider

The `CatalogContextProvider` is a React context provider that bootstraps all
plugin metadata needed by the generic components. It wraps the entire
`GenericCatalogRoutes` tree so that every child has access to plugin data
without prop drilling.

```
Mount CatalogContextProvider
         |
         v
  useAPIState(hostPath, createAPI)
         |
         v
  apiState.api.getAllPlugins()
         |  GET /api/v1/model_catalog/plugins
         v
  For each plugin in response:
         |
         v
  getPluginCapabilities(hostPath, plugin.name)
         |  GET /api/catalog/{plugin}/capabilities
         v
  Store in capabilitiesMap: Record<string, PluginCapabilitiesV2>
```

### Context Shape

```typescript
// clients/ui/frontend/src/app/context/catalog/CatalogContext.tsx
type CatalogContextType = {
  plugins: PluginInfo[];                              // List of registered plugins
  pluginsLoaded: boolean;                             // True after initial fetch completes
  pluginsLoadError?: Error;                           // Set if plugin list fetch fails
  capabilitiesMap: Record<string, PluginCapabilitiesV2>;  // V2 capabilities keyed by plugin name
  getPluginCaps: (pluginName: string) => PluginCapabilitiesV2 | undefined;
};
```

### Behavior

| Trigger | Action |
|---------|--------|
| Provider mounts | Fetches plugin list from BFF proxy |
| Plugin list arrives | Iterates over plugins, fetches V2 capabilities for each |
| Capabilities arrive | Stores in `capabilitiesMap` state, triggers re-render of children |
| Capabilities fetch error | Silently skips the plugin (it will not appear in navigation) |
| Plugin list fetch error | Sets `pluginsLoadError`, children render error state |

The hook `useCatalogPlugins()` is the consumer entry point. Every screen and
component in the generic catalog tree calls it to read plugin metadata and
capabilities.

---

## Routing

`GenericCatalogRoutes` defines static, parameterized React Router routes.
Plugin names and entity types are resolved at runtime from URL parameters, not
from build-time code generation.

```typescript
// clients/ui/frontend/src/app/pages/genericCatalog/GenericCatalogRoutes.tsx
const GenericCatalogRoutes: React.FC = () => (
  <CatalogContextProvider>
    <Routes>
      <Route index element={<CatalogHomePage />} />
      <Route path=":pluginName/:entityPlural" element={<PluginEntityListPage />} />
      <Route path=":pluginName/:entityPlural/:entityName" element={<PluginEntityDetailPage />} />
      <Route path="*" element={<Navigate to="." replace />} />
    </Routes>
  </CatalogContextProvider>
);
```

### Route Table

| Pattern | Component | Description |
|---------|-----------|-------------|
| `/catalog` | `CatalogHomePage` | Plugin gallery showing a card per plugin |
| `/catalog/:pluginName/:entityPlural` | `PluginEntityListPage` | Entity list with dynamic columns and filters |
| `/catalog/:pluginName/:entityPlural/:entityName` | `PluginEntityDetailPage` | Entity detail with dynamic sections and actions |
| `/catalog/*` | `Navigate` redirect | Catch-all fallback to the catalog index |

The `:pluginName` parameter maps to a plugin's `name` field (e.g. `mcp`,
`knowledge`, `model`). The `:entityPlural` parameter maps to the `plural` field
from the plugin's `EntityCapabilities` (e.g. `mcpservers`, `knowledgesources`).

When a user clicks a plugin card on the home page, `CatalogHomePage` resolves
the first entity type from the plugin's capabilities and navigates to
`/catalog/{pluginName}/{entityPlural}`.

---

## Component Inventory

| Component | Purpose |
|-----------|---------|
| `CatalogHomePage` | Plugin gallery landing page showing a PatternFly `Gallery` of cards, one per registered plugin, with display name, description, version, and entity count |
| `PluginEntityListPage` | Dynamic list screen driven by V2 capabilities; reads `EntityCapabilities` from context, fetches entity list, applies filters and search, delegates to `GenericListView` |
| `PluginEntityDetailPage` | Dynamic detail screen; fetches a single entity, resolves available actions from capabilities, delegates to `GenericDetailView` and `GenericActionBar` |
| `GenericListView` | Reusable PatternFly `Table` that renders columns from `V2ColumnHint[]` definitions; extracts cell values from entity JSON via dot-path; first column is a clickable link |
| `GenericDetailView` | Reusable detail panel that groups `V2FieldHint[]` into named sections, renders each section as a PatternFly `Card` with a `DescriptionList`; supports type-specific rendering (tags, urls, markdown, booleans) |
| `GenericActionDialog` | Modal dialog for executing actions; dynamically builds a form from `ActionParameter[]`; supports dry-run toggle; handles submit/error/loading states |
| `GenericActionBar` | Horizontal button bar generated from the capabilities `actions` list; renders destructive actions with a danger variant |
| `GenericFilterBar` | Dynamic `Toolbar` with search input and filter controls built from `V2FilterField[]` definitions; supports text, select (multi-value), and boolean filter types |

---

## Data Flow

```
CatalogContextProvider
       |
       | fetches /api/v1/model_catalog/plugins
       | fetches /api/catalog/{plugin}/capabilities (per plugin)
       |
       v
 capabilitiesMap: Record<string, PluginCapabilitiesV2>
       |
       |  React Router resolves :pluginName, :entityPlural
       v
PluginEntityListPage
       |
       | getPluginCaps(pluginName) --> EntityCapabilities
       | usePluginEntities(pluginName, entityPlural, queryParams)
       |   GET /api/catalog/{plugin}/entities/{entityPlural}
       v
GenericFilterBar                         GenericListView
  |  builds filterQuery from             |  renders columns from V2ColumnHint[]
  |  V2FilterField definitions           |  uses getFieldValue(entity, col.path)
  |  and search term                     |  to extract values from entity JSON
  v                                      v
Filter state updates -->  re-fetch  <--  User clicks row
                                              |
                                              v
                                   PluginEntityDetailPage
                                         |
                                         | getEntity(hostPath, plugin, entityPlural, entityName)
                                         |   GET /api/catalog/{plugin}/entities/{plural}/{name}
                                         | resolves actions from caps.actions[]
                                         v
                                   GenericDetailView          GenericActionBar
                                     |                            |
                                     | renders sections from      | renders action buttons
                                     | V2FieldHint[] grouped by   | from ActionDefinition[]
                                     | detailSections             |
                                     v                            v
                                   PatternFly Cards         User clicks action
                                   with DescriptionLists         |
                                                                 v
                                                          GenericActionDialog
                                                            |
                                                            | builds form from ActionParameter[]
                                                            | POST .../entities/{name}/action
                                                            v
                                                          Action result notification
```

### Utility Functions

Two helper functions in `utils.ts` support the generic rendering pipeline:

| Function | Signature | Purpose |
|----------|-----------|---------|
| `getFieldValue` | `(entity: GenericEntity, path: string) => unknown` | Walks a dot-separated path (e.g. `metadata.name`) to extract a value from a JSON entity object |
| `formatFieldValue` | `(value: unknown, type?: string) => string` | Formats a raw value for display; handles tags (join), booleans (Yes/No), dates (toLocaleDateString), objects (JSON.stringify), and null (dash) |

### Type-Specific Rendering in GenericDetailView

The detail view's `renderFieldValue` function enhances display based on the
`V2FieldHint.type` field:

| Type | Rendering |
|------|-----------|
| `tags` | PatternFly `LabelGroup` with a `Label` per tag |
| `url` | Anchor tag opening in a new tab |
| `markdown` | Preformatted `<pre>` block with word wrap |
| `boolean` | "Yes" / "No" text |
| `date` | Locale-formatted date string |
| (default) | Plain string via `formatFieldValue` |

---

## API Service Layer

The generic components do not call backend endpoints directly. Two API service
modules abstract the HTTP calls and response parsing:

### catalogCapabilities/service.ts

| Function | Endpoint | Returns |
|----------|----------|---------|
| `getAllPlugins` | `GET /api/v1/model_catalog/plugins` | `PluginsResponse` |
| `getPluginCapabilities` | `GET /api/catalog/{plugin}/capabilities` | `PluginCapabilitiesV2` |

### catalogEntities/service.ts

| Function | Endpoint | Returns |
|----------|----------|---------|
| `getEntityList` | `GET /api/catalog/{plugin}/entities/{plural}` | `GenericEntityList` |
| `getEntity` | `GET /api/catalog/{plugin}/entities/{plural}/{name}` | `GenericEntity` |
| `executeEntityAction` | `POST /api/catalog/{plugin}/entities/{plural}/{name}/action` | `GenericEntity` |

All functions follow the curried pattern `(hostPath, ...) => (opts) => Promise<T>`
used by the `mod-arch-core` library. Responses are unwrapped via
`isModArchResponse` and typed to the generic entity types.

### usePluginEntities Hook

The `usePluginEntities` custom hook manages the entity list lifecycle:

```
usePluginEntities(pluginName, entityPlural, queryParams)
       |
       |  on mount / on params change:
       |    getEntityList(hostPath, plugin, entity, queryParams)
       |
       |  returns:
       |    entities: GenericEntity[]     -- current page of entities
       |    loaded: boolean               -- fetch complete
       |    error?: Error                 -- fetch error
       |    totalSize: number             -- server-reported total
       |    nextPageToken?: string        -- for token-based pagination
       |    loadMore: () => void          -- fetch next page, append to entities
       |    isLoadingMore: boolean        -- true while loading next page
       |    refresh: () => void           -- re-fetch from first page
```

---

## TypeScript Type Contracts

### GenericEntity

```typescript
// clients/ui/frontend/src/app/types/asset.ts
type GenericEntity = Record<string, unknown>;
```

A generic entity is an opaque JSON object. Fields are accessed exclusively
through dot-path lookups driven by the V2 capabilities column and field
definitions. No plugin-specific type assertions exist in the generic UI code.

### V2 Capabilities Types

```typescript
// clients/ui/frontend/src/app/types/capabilities.ts
type V2ColumnHint = {
  name: string;        // Internal column name
  displayName: string; // Table header label
  path: string;        // Dot-separated path into the entity JSON
  type?: string;       // Rendering type hint (e.g. "tags", "date", "boolean")
  sortable?: boolean;  // Whether the column supports server-side sorting
  width?: string;      // PatternFly column width percentage
};

type V2FilterField = {
  name: string;
  displayName: string;
  type: 'text' | 'select' | 'boolean' | 'number';
  options?: string[];     // For select filters
  operators?: string[];   // For advanced filter expressions
};

type V2FieldHint = {
  name: string;
  displayName: string;
  path: string;         // Dot path into entity JSON
  type?: string;        // Rendering type (tags, url, markdown, etc.)
  section?: string;     // Groups fields into detail view sections
};

type EntityCapabilities = {
  kind: string;
  plural: string;
  displayName: string;
  description?: string;
  endpoints: { list: string; get: string; action?: string };
  fields: {
    columns: V2ColumnHint[];
    filterFields?: V2FilterField[];
    detailFields?: V2FieldHint[];
  };
  uiHints?: {
    icon?: string;
    nameField?: string;
    detailSections?: string[];
  };
  actions?: string[];   // References action IDs from plugin-level actions[]
};

type ActionDefinition = {
  id: string;
  displayName: string;
  description?: string;
  scope?: string;
  supportsDryRun?: boolean;
  idempotent?: boolean;
  destructive?: boolean;
  parameters?: ActionParameter[];
};

type ActionParameter = {
  name: string;
  type: 'string' | 'boolean' | 'number' | 'tags' | 'key-value';
  label: string;
  required?: boolean;
  description?: string;
  defaultValue?: unknown;
};
```

---

## Zero-Code-Change Proof

The following asset-type plugins render fully in the generic UI without any
plugin-specific code in `clients/ui/frontend/`:

- **MCP Servers** -- the original MCP catalog plugin
- **Knowledge Sources** -- Phase 5 proof-of-concept plugin
- **Prompts** -- prompt template catalog
- **Agents** -- agent definition catalog
- **Guardrails** -- guardrail policy catalog
- **Policies** -- governance policy catalog
- **Skills** -- reusable skill catalog

Each of these plugins:

1. Registers with the catalog server at startup
2. Provides V2 capabilities via `CapabilitiesV2Provider`
3. Appears automatically in `CatalogHomePage` as a clickable card
4. Renders a list view with columns, filters, and pagination
5. Renders a detail view with grouped sections and typed fields
6. Exposes actions (tag, annotate, deprecate, plus any custom actions) via
   `GenericActionBar` and `GenericActionDialog`

The only code that knows about specific plugins is in the backend plugin
implementations under `catalog/plugins/`. The frontend and CLI contain no
`if plugin === "mcp"` or `switch(pluginName)` branching.

### Adding a New Plugin

To add a new asset-type plugin that appears in the generic UI:

```
1. Implement CatalogPlugin interface in Go (backend)
2. Implement CapabilitiesV2Provider interface
     - Define entities with columns, detailFields, filterFields
     - Define actions with parameters
3. Register the plugin with the catalog server
4. The generic UI discovers the plugin at runtime
     - No frontend code changes
     - No route changes
     - No component changes
```

---

## Key Files

| File | Path | Purpose |
|------|------|---------|
| `GenericCatalogRoutes.tsx` | `clients/ui/frontend/src/app/pages/genericCatalog/GenericCatalogRoutes.tsx` | Top-level route definitions wrapping CatalogContextProvider |
| `CatalogHomePage.tsx` | `clients/ui/frontend/src/app/pages/genericCatalog/screens/CatalogHomePage.tsx` | Plugin gallery with PatternFly Gallery and Cards |
| `PluginEntityListPage.tsx` | `clients/ui/frontend/src/app/pages/genericCatalog/screens/PluginEntityListPage.tsx` | Dynamic entity list with search, filters, pagination |
| `PluginEntityDetailPage.tsx` | `clients/ui/frontend/src/app/pages/genericCatalog/screens/PluginEntityDetailPage.tsx` | Dynamic entity detail with actions and breadcrumbs |
| `GenericListView.tsx` | `clients/ui/frontend/src/app/pages/genericCatalog/components/GenericListView.tsx` | Reusable PatternFly Table driven by V2ColumnHint |
| `GenericDetailView.tsx` | `clients/ui/frontend/src/app/pages/genericCatalog/components/GenericDetailView.tsx` | Reusable detail panel with section grouping and type rendering |
| `GenericActionBar.tsx` | `clients/ui/frontend/src/app/pages/genericCatalog/components/GenericActionBar.tsx` | Action button bar from ActionDefinition[] |
| `GenericActionDialog.tsx` | `clients/ui/frontend/src/app/pages/genericCatalog/components/GenericActionDialog.tsx` | Modal dialog with dynamic form from ActionParameter[] |
| `GenericFilterBar.tsx` | `clients/ui/frontend/src/app/pages/genericCatalog/components/GenericFilterBar.tsx` | Dynamic toolbar with search and filter controls |
| `utils.ts` | `clients/ui/frontend/src/app/pages/genericCatalog/utils.ts` | getFieldValue and formatFieldValue helper functions |
| `CatalogContext.tsx` | `clients/ui/frontend/src/app/context/catalog/CatalogContext.tsx` | CatalogContextProvider and useCatalogPlugins hook |
| `capabilities.ts` | `clients/ui/frontend/src/app/types/capabilities.ts` | TypeScript types for V2 capabilities, actions, entities |
| `asset.ts` | `clients/ui/frontend/src/app/types/asset.ts` | GenericEntity and GenericEntityList types |
| `service.ts` (capabilities) | `clients/ui/frontend/src/app/api/catalogCapabilities/service.ts` | API functions for plugin list and capabilities |
| `service.ts` (entities) | `clients/ui/frontend/src/app/api/catalogEntities/service.ts` | API functions for entity list, get, and action execution |
| `usePluginEntities.ts` | `clients/ui/frontend/src/app/hooks/usePluginEntities.ts` | Custom hook for paginated entity list lifecycle |

---

[Back to Clients](./README.md) | [Prev: BFF Integration](./bff-integration.md) | [Next: catalogctl and Conformance](./catalogctl-and-conformance.md)

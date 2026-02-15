# State Management

This document covers the React Context API patterns and custom hooks used for state management.

## Overview

The frontend uses React Context API for state management, organized in a layered architecture:

1. **Global Contexts** - App-wide state (config, user)
2. **Feature Contexts** - Feature-specific state (Model Registry, Catalog)
3. **Custom Hooks** - Abstracted data fetching and state

## Context Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Context Hierarchy                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ModularArchContextProvider (from mod-arch-core)            │
│      │                                                       │
│      └── ThemeProvider                                       │
│          │                                                   │
│          └── NotificationContextProvider                     │
│              │                                               │
│              └── App (AppContext.Provider)                  │
│                  │                                           │
│                  └── ModelRegistrySelectorContextProvider   │
│                      │                                       │
│                      ├── ModelRegistryContextProvider       │
│                      │   (per selected registry)            │
│                      │                                       │
│                      ├── ModelCatalogContextProvider        │
│                      │   (catalog browsing)                 │
│                      │                                       │
│                      └── McpCatalogContextProvider          │
│                          (MCP server browsing)              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Global Contexts

### AppContext

Provides global application configuration and user settings.

```typescript
// app/context/AppContext.ts
import { ConfigSettings, UserSettings } from 'mod-arch-core';

type AppContextProps = {
  config: ConfigSettings;
  user: UserSettings;
};

const AppContext = createContext<AppContextProps | undefined>(undefined);

export const useAppContext = () => {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('useAppContext must be used within AppContext.Provider');
  }
  return context;
};
```

**Usage:**

```typescript
const MyComponent: React.FC = () => {
  const { config, user } = useAppContext();

  return (
    <div>
      <p>User: {user.name}</p>
      <p>API Host: {config.apiHost}</p>
    </div>
  );
};
```

### ModelRegistrySelectorContext

Manages multi-registry selection in federated mode.

```typescript
// app/context/ModelRegistrySelectorContext.tsx
type ModelRegistrySelectorContextType = {
  modelRegistriesLoaded: boolean;
  modelRegistries: ModelRegistry[];
  preferredModelRegistry: ModelRegistry | undefined;
  updatePreferredModelRegistry: (registry: ModelRegistry | undefined) => void;
};

const ModelRegistrySelectorContext = createContext<ModelRegistrySelectorContextType | undefined>(undefined);

export const ModelRegistrySelectorContextProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [modelRegistries] = useModelRegistries();
  const [preferredModelRegistry, setPreferredModelRegistry] = useBrowserStorage<ModelRegistry | undefined>(
    'preferredModelRegistry',
    undefined
  );

  const contextValue = useMemo(() => ({
    modelRegistriesLoaded: modelRegistries !== undefined,
    modelRegistries: modelRegistries ?? [],
    preferredModelRegistry,
    updatePreferredModelRegistry: setPreferredModelRegistry,
  }), [modelRegistries, preferredModelRegistry, setPreferredModelRegistry]);

  return (
    <ModelRegistrySelectorContext.Provider value={contextValue}>
      {children}
    </ModelRegistrySelectorContext.Provider>
  );
};
```

## Feature Contexts

### ModelRegistryContext

Provides API state for Model Registry operations.

```typescript
// app/context/ModelRegistryContext.tsx
type ModelRegistryContextType = {
  apiState: ModelRegistryAPIState;
  refreshAPIState: () => void;
};

type ModelRegistryAPIState = {
  apiAvailable: boolean;
  api: {
    createRegisteredModel: (opts: APIOptions, data: CreateModelData) => Promise<RegisteredModel>;
    getRegisteredModel: (opts: APIOptions, id: string) => Promise<RegisteredModel>;
    getRegisteredModels: (opts: APIOptions) => Promise<RegisteredModelList>;
    // ... more API methods
  } | null;
};

export const ModelRegistryContextProvider: React.FC<PropsWithChildren<{ hostPath: string }>> = ({
  children,
  hostPath,
}) => {
  const [apiState, refreshAPIState] = useModelRegistryAPIState(hostPath);

  const contextValue = useMemo(() => ({
    apiState,
    refreshAPIState,
  }), [apiState, refreshAPIState]);

  return (
    <ModelRegistryContext.Provider value={contextValue}>
      {children}
    </ModelRegistryContext.Provider>
  );
};
```

### ModelCatalogContext

Manages catalog browsing state including filters and data.

```typescript
// app/context/modelCatalog/ModelCatalogContext.tsx
type ModelCatalogContextType = {
  // Data state
  catalogModelsLoaded: boolean;
  catalogModels: CatalogModelList | null;
  catalogModelLoadError: Error | undefined;
  catalogSources: CatalogSourceList | null;

  // Filter state
  filters: ModelCatalogFilterState;
  searchTerm: string;
  filterOptions: FilterOptions;

  // Actions
  updateFilters: (filters: ModelCatalogFilterState) => void;
  updateSearchTerm: (term: string) => void;
  resetFilters: () => void;

  // API state
  apiState: ModelCatalogAPIState;
  refreshAPIState: () => void;
  refreshCatalogModels: () => void;
  refreshCatalogSources: () => void;
};

export const ModelCatalogContextProvider: React.FC<PropsWithChildren> = ({ children }) => {
  // API state
  const [apiState, refreshAPIState] = useModelCatalogAPIState(hostPath);

  // Data state
  const [catalogModels, setCatalogModels] = useState<CatalogModelList | null>(null);
  const [catalogModelsLoaded, setCatalogModelsLoaded] = useState(false);
  const [catalogModelLoadError, setCatalogModelLoadError] = useState<Error | undefined>();
  const [catalogSources, setCatalogSources] = useState<CatalogSourceList | null>(null);

  // Filter state (local)
  const [filters, setFilters] = useState<ModelCatalogFilterState>(defaultFilters);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterOptions, setFilterOptions] = useState<FilterOptions>(defaultFilterOptions);

  // Fetch catalog models on mount and filter change
  useEffect(() => {
    if (!apiState.apiAvailable || !apiState.api) return;

    const fetchModels = async () => {
      try {
        const query = buildFilterQuery(filters, searchTerm);
        const result = await apiState.api.getCatalogModels({}, query);
        setCatalogModels(result);
        setCatalogModelsLoaded(true);
      } catch (error) {
        setCatalogModelLoadError(error as Error);
      }
    };

    fetchModels();
  }, [apiState, filters, searchTerm]);

  // ... rest of implementation
};
```

### McpCatalogContext

Similar pattern for MCP server browsing.

```typescript
// app/context/mcpCatalog/McpCatalogContext.tsx
type McpCatalogContextType = {
  // Data
  mcpServersLoaded: boolean;
  mcpServers: McpServerList | null;
  mcpSources: McpCatalogSourceList | null;

  // Filters
  filters: McpServerFilterState;
  searchTerm: string;
  filterOptions: McpFilterOptions;

  // Actions
  updateFilters: (filters: McpServerFilterState) => void;
  updateSearchTerm: (term: string) => void;
  resetFilters: () => void;

  // API
  apiState: McpCatalogAPIState;
  refreshMcpServers: () => void;
};
```

## Custom Hooks

### useModelRegistryAPIState

Wraps API functions in stateful container.

```typescript
// app/hooks/useModelRegistryAPIState.tsx
const useModelRegistryAPIState = (
  hostPath: string | null,
  queryParameters?: Record<string, unknown>,
): [apiState: ModelRegistryAPIState, refreshAPIState: () => void] => {

  const createAPI = useCallback(
    (path: string) => ({
      createRegisteredModel: createRegisteredModel(path, queryParameters),
      getRegisteredModel: getRegisteredModel(path, queryParameters),
      getRegisteredModels: getRegisteredModels(path, queryParameters),
      updateRegisteredModel: updateRegisteredModel(path, queryParameters),
      // ... more API methods
    }),
    [queryParameters],
  );

  return useAPIState(hostPath, createAPI);
};
```

### useRegisteredModels

Fetches and manages registered models list.

```typescript
// app/hooks/useRegisteredModels.ts
const useRegisteredModels = (
  params?: GetRegisteredModelsParams,
): [models: RegisteredModel[], loaded: boolean, error: Error | undefined, refresh: () => void] => {
  const { apiState } = useModelRegistryContext();
  const [models, setModels] = useState<RegisteredModel[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState<Error | undefined>();

  const fetchModels = useCallback(async () => {
    if (!apiState.apiAvailable || !apiState.api) return;

    try {
      const result = await apiState.api.getRegisteredModels({}, params);
      setModels(result.items);
      setLoaded(true);
    } catch (e) {
      setError(e as Error);
    }
  }, [apiState, params]);

  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  return [models, loaded, error, fetchModels];
};
```

### useModelRegistries

Fetches available model registries (federated mode).

```typescript
// app/hooks/useModelRegistries.ts
const useModelRegistries = (): [
  registries: ModelRegistry[] | undefined,
  loaded: boolean,
  error: Error | undefined,
] => {
  const { config } = useAppContext();
  const [registries, setRegistries] = useState<ModelRegistry[] | undefined>();
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState<Error | undefined>();

  useEffect(() => {
    const fetchRegistries = async () => {
      try {
        const result = await getModelRegistries(config.apiHost);
        setRegistries(result);
        setLoaded(true);
      } catch (e) {
        setError(e as Error);
      }
    };

    fetchRegistries();
  }, [config.apiHost]);

  return [registries, loaded, error];
};
```

### useCatalogSourcesWithPolling

Implements polling for status updates.

```typescript
// app/hooks/modelCatalog/useCatalogSourcesWithPolling.ts
const useCatalogSourcesWithPolling = (
  pollingInterval: number = 30000,
): [sources: CatalogSource[], loaded: boolean] => {
  const { apiState } = useModelCatalogContext();
  const [sources, setSources] = useState<CatalogSource[]>([]);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    if (!apiState.apiAvailable || !apiState.api) return;

    const fetchSources = async () => {
      const result = await apiState.api.getCatalogSources({});
      setSources(result.items);
      setLoaded(true);
    };

    // Initial fetch
    fetchSources();

    // Set up polling
    const intervalId = setInterval(fetchSources, pollingInterval);

    return () => clearInterval(intervalId);
  }, [apiState, pollingInterval]);

  return [sources, loaded];
};
```

## Filter State Management

### Filter State Pattern

```typescript
// Types
type ModelCatalogFilterState = {
  provider: string[];
  license: string[];
  tasks: string[];
  maturity: string[];
};

const defaultFilters: ModelCatalogFilterState = {
  provider: [],
  license: [],
  tasks: [],
  maturity: [],
};

// In context
const [filters, setFilters] = useState<ModelCatalogFilterState>(defaultFilters);

const updateFilters = useCallback((newFilters: ModelCatalogFilterState) => {
  setFilters(newFilters);
}, []);

const resetFilters = useCallback(() => {
  setFilters(defaultFilters);
  setSearchTerm('');
}, []);
```

### Filter Query Building

```typescript
// utilities/filterQuery.ts
const buildFilterQuery = (filters: ModelCatalogFilterState, searchTerm: string): string => {
  const conditions: string[] = [];

  if (searchTerm) {
    conditions.push(`q=${encodeURIComponent(searchTerm)}`);
  }

  if (filters.provider.length > 0) {
    const providerFilter = filters.provider
      .map(p => `provider='${p}'`)
      .join(' OR ');
    conditions.push(`filterQuery=${encodeURIComponent(`(${providerFilter})`)}`);
  }

  // ... more filter building

  return conditions.join('&');
};
```

## Browser Storage Integration

Using mod-arch-core's browser storage for persistence.

```typescript
// Persisting user preferences
const [preferredRegistry, setPreferredRegistry] = useBrowserStorage<ModelRegistry | undefined>(
  'preferredModelRegistry',
  undefined
);

const [viewMode, setViewMode] = useBrowserStorage<'gallery' | 'list'>(
  'catalogViewMode',
  'gallery'
);
```

## Best Practices

### 1. Memoize Context Values

```typescript
const contextValue = useMemo(() => ({
  data,
  actions: {
    updateData,
    refreshData,
  },
}), [data, updateData, refreshData]);

return (
  <MyContext.Provider value={contextValue}>
    {children}
  </MyContext.Provider>
);
```

### 2. Separate Loading/Error States

```typescript
type ContextType = {
  data: Data | null;
  loaded: boolean;
  error: Error | undefined;
  refresh: () => void;
};
```

### 3. Use Callback Memoization

```typescript
const updateFilters = useCallback((newFilters: FilterState) => {
  setFilters(newFilters);
}, []);
```

### 4. Provide Refresh Functions

```typescript
const contextValue = {
  // Data
  models,
  loaded,

  // Actions
  refreshModels,  // Allow manual refresh
};
```

---

[Back to Frontend Index](./README.md) | [Previous: Architecture](./architecture.md) | [Next: Component Library](./component-library.md)

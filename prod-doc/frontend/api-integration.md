# API Integration

This document covers the REST API client patterns used in the frontend.

## Overview

The frontend uses a layered approach for API integration:

1. **API Functions** - Low-level HTTP calls
2. **API State Hooks** - State management for API availability
3. **Context Integration** - React Context for component access
4. **Custom Hooks** - Data fetching with loading/error states

## API Function Pattern

API functions use a higher-order function pattern for flexibility and testability.

### Pattern Structure

```typescript
// Curried function: (config) => (options, args) => Promise<Result>
const apiFunction =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, ...args: Args): Promise<Result> =>
    handleRestFailures(
      restMethod(hostPath, endpoint, body, queryParams, opts),
    ).then(response => extractData(response));
```

### Example: Model Registry API

```typescript
// app/api/service.ts
import { restCREATE, restGET, restPATCH, handleRestFailures, assembleModArchBody } from 'mod-arch-core';

// Create registered model
export const createRegisteredModel =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, data: CreateRegisteredModelData): Promise<RegisteredModel> =>
    handleRestFailures(
      restCREATE(hostPath, `/registered_models`, assembleModArchBody(data), queryParams, opts),
    ).then((response) => {
      if (isModArchResponse<RegisteredModel>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

// Get registered model by ID
export const getRegisteredModel =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, id: string): Promise<RegisteredModel> =>
    handleRestFailures(
      restGET(hostPath, `/registered_models/${id}`, queryParams, opts),
    ).then((response) => {
      if (isModArchResponse<RegisteredModel>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

// List registered models
export const getRegisteredModels =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, params?: GetRegisteredModelsParams): Promise<RegisteredModelList> =>
    handleRestFailures(
      restGET(hostPath, `/registered_models`, { ...queryParams, ...params }, opts),
    ).then((response) => {
      if (isModArchResponse<RegisteredModelList>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

// Update registered model
export const updateRegisteredModel =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, id: string, data: UpdateRegisteredModelData): Promise<RegisteredModel> =>
    handleRestFailures(
      restPATCH(hostPath, `/registered_models/${id}`, assembleModArchBody(data), queryParams, opts),
    ).then((response) => {
      if (isModArchResponse<RegisteredModel>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });
```

### Example: Model Catalog API

```typescript
// app/api/modelCatalog/service.ts
export const getCatalogModels =
  (hostPath: string) =>
  (opts: APIOptions, query?: CatalogModelsQuery): Promise<CatalogModelList> =>
    handleRestFailures(
      restGET(hostPath, `/models`, query ? buildQueryParams(query) : {}, opts),
    ).then(extractData);

export const getCatalogModel =
  (hostPath: string) =>
  (opts: APIOptions, modelName: string): Promise<CatalogModel> =>
    handleRestFailures(
      restGET(hostPath, `/models/${encodeURIComponent(modelName)}`, {}, opts),
    ).then(extractData);

export const getCatalogSources =
  (hostPath: string) =>
  (opts: APIOptions): Promise<CatalogSourceList> =>
    handleRestFailures(
      restGET(hostPath, `/sources`, {}, opts),
    ).then(extractData);

export const getFilterOptions =
  (hostPath: string) =>
  (opts: APIOptions): Promise<FilterOptions> =>
    handleRestFailures(
      restGET(hostPath, `/models/filter_options`, {}, opts),
    ).then(extractData);
```

## API State Hook

The `useAPIState` hook from mod-arch-core wraps API functions.

```typescript
// app/hooks/useModelRegistryAPIState.tsx
import { useAPIState } from 'mod-arch-core';

type ModelRegistryAPIs = {
  createRegisteredModel: ReturnType<typeof createRegisteredModel>;
  getRegisteredModel: ReturnType<typeof getRegisteredModel>;
  getRegisteredModels: ReturnType<typeof getRegisteredModels>;
  updateRegisteredModel: ReturnType<typeof updateRegisteredModel>;
  // ... more methods
};

export type ModelRegistryAPIState = {
  apiAvailable: boolean;
  api: ModelRegistryAPIs | null;
};

const useModelRegistryAPIState = (
  hostPath: string | null,
  queryParameters?: Record<string, unknown>,
): [apiState: ModelRegistryAPIState, refreshAPIState: () => void] => {
  const createAPI = useCallback(
    (path: string): ModelRegistryAPIs => ({
      createRegisteredModel: createRegisteredModel(path, queryParameters),
      getRegisteredModel: getRegisteredModel(path, queryParameters),
      getRegisteredModels: getRegisteredModels(path, queryParameters),
      updateRegisteredModel: updateRegisteredModel(path, queryParameters),
      // ... more methods
    }),
    [queryParameters],
  );

  return useAPIState(hostPath, createAPI);
};

export default useModelRegistryAPIState;
```

## Context Integration

API state is provided through React Context.

```typescript
// app/context/ModelRegistryContext.tsx
import useModelRegistryAPIState, { ModelRegistryAPIState } from '~/app/hooks/useModelRegistryAPIState';

type ModelRegistryContextType = {
  apiState: ModelRegistryAPIState;
  refreshAPIState: () => void;
};

const ModelRegistryContext = createContext<ModelRegistryContextType | undefined>(undefined);

export const ModelRegistryContextProvider: React.FC<PropsWithChildren<{
  hostPath: string;
}>> = ({ children, hostPath }) => {
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

export const useModelRegistryContext = () => {
  const context = useContext(ModelRegistryContext);
  if (!context) {
    throw new Error('useModelRegistryContext must be used within ModelRegistryContextProvider');
  }
  return context;
};
```

## Data Fetching Hooks

Custom hooks abstract data fetching with loading and error states.

### Basic Pattern

```typescript
// app/hooks/useRegisteredModels.ts
const useRegisteredModels = (
  params?: GetRegisteredModelsParams,
): [
  models: RegisteredModel[],
  loaded: boolean,
  error: Error | undefined,
  refresh: () => void,
] => {
  const { apiState } = useModelRegistryContext();
  const [models, setModels] = useState<RegisteredModel[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState<Error | undefined>();

  const fetchModels = useCallback(async () => {
    if (!apiState.apiAvailable || !apiState.api) {
      return;
    }

    try {
      setError(undefined);
      const result = await apiState.api.getRegisteredModels({}, params);
      setModels(result.items);
      setLoaded(true);
    } catch (e) {
      setError(e as Error);
      setLoaded(true);
    }
  }, [apiState, params]);

  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  return [models, loaded, error, fetchModels];
};
```

### With Pagination

```typescript
const useRegisteredModelsPaginated = (
  pageSize: number = 20,
): {
  models: RegisteredModel[];
  page: number;
  totalPages: number;
  loading: boolean;
  error: Error | undefined;
  nextPage: () => void;
  prevPage: () => void;
  goToPage: (page: number) => void;
} => {
  const { apiState } = useModelRegistryContext();
  const [models, setModels] = useState<RegisteredModel[]>([]);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | undefined>();

  useEffect(() => {
    if (!apiState.apiAvailable || !apiState.api) return;

    const fetchPage = async () => {
      setLoading(true);
      try {
        const result = await apiState.api.getRegisteredModels({}, {
          pageSize,
          pageToken: page > 1 ? `page-${page}` : undefined,
        });
        setModels(result.items);
        setTotalPages(Math.ceil(result.size / pageSize));
      } catch (e) {
        setError(e as Error);
      } finally {
        setLoading(false);
      }
    };

    fetchPage();
  }, [apiState, page, pageSize]);

  return {
    models,
    page,
    totalPages,
    loading,
    error,
    nextPage: () => setPage(p => Math.min(p + 1, totalPages)),
    prevPage: () => setPage(p => Math.max(p - 1, 1)),
    goToPage: setPage,
  };
};
```

## Error Handling

### Error Types

```typescript
// Common error types
interface APIError extends Error {
  statusCode?: number;
  details?: Record<string, unknown>;
}

const isAPIError = (error: unknown): error is APIError => {
  return error instanceof Error && 'statusCode' in error;
};
```

### Error Handling in Components

```typescript
const ModelDetails: React.FC<{ modelId: string }> = ({ modelId }) => {
  const { apiState } = useModelRegistryContext();
  const [model, setModel] = useState<RegisteredModel | null>(null);
  const [error, setError] = useState<Error | undefined>();

  useEffect(() => {
    if (!apiState.apiAvailable || !apiState.api) return;

    apiState.api.getRegisteredModel({}, modelId)
      .then(setModel)
      .catch((e) => {
        setError(e);
        if (isAPIError(e) && e.statusCode === 404) {
          // Handle not found
        }
      });
  }, [apiState, modelId]);

  if (error) {
    return (
      <Alert variant="danger" title="Error loading model">
        {error.message}
      </Alert>
    );
  }

  if (!model) {
    return <Spinner />;
  }

  return <div>{model.name}</div>;
};
```

## Query Parameter Building

### Filter Query Builder

```typescript
// app/utilities/filterQuery.ts
interface FilterState {
  provider: string[];
  license: string[];
  tasks: string[];
}

const buildFilterQueryParam = (filters: FilterState): string | undefined => {
  const conditions: string[] = [];

  if (filters.provider.length > 0) {
    const providerConditions = filters.provider
      .map(p => `provider='${escapeQuotes(p)}'`)
      .join(' OR ');
    conditions.push(`(${providerConditions})`);
  }

  if (filters.license.length > 0) {
    const licenseConditions = filters.license
      .map(l => `license='${escapeQuotes(l)}'`)
      .join(' OR ');
    conditions.push(`(${licenseConditions})`);
  }

  if (filters.tasks.length > 0) {
    const taskConditions = filters.tasks
      .map(t => `tasks CONTAINS '${escapeQuotes(t)}'`)
      .join(' OR ');
    conditions.push(`(${taskConditions})`);
  }

  if (conditions.length === 0) {
    return undefined;
  }

  return conditions.join(' AND ');
};

const escapeQuotes = (s: string): string => s.replace(/'/g, "''");
```

### Request Parameter Assembly

```typescript
const buildCatalogQueryParams = (
  searchTerm: string,
  filters: FilterState,
  sortBy?: string,
  sortOrder?: 'ASC' | 'DESC',
): Record<string, string> => {
  const params: Record<string, string> = {};

  if (searchTerm) {
    params.q = searchTerm;
  }

  const filterQuery = buildFilterQueryParam(filters);
  if (filterQuery) {
    params.filterQuery = filterQuery;
  }

  if (sortBy) {
    params.orderBy = sortBy;
    params.sortOrder = sortOrder || 'ASC';
  }

  return params;
};
```

## Host Path Configuration

### Environment-Based Configuration

```typescript
// app/utilities/const.ts
export const API_PATHS = {
  modelRegistry: '/api/v1/model_registry',
  modelCatalog: '/api/v1/model_catalog',
  mcpCatalog: '/api/v1/mcp_catalog',
};

// Using with mod-arch-core
const getApiHost = () => {
  const { config } = useModularArchContext();
  return config.apiHost || '';
};

const getModelRegistryPath = (registryName: string) => {
  const host = getApiHost();
  return `${host}${API_PATHS.modelRegistry}/${registryName}`;
};
```

## Testing API Functions

```typescript
// app/api/__tests__/service.spec.ts
import { createRegisteredModel, getRegisteredModel } from '../service';

jest.mock('mod-arch-core', () => ({
  restCREATE: jest.fn(),
  restGET: jest.fn(),
  handleRestFailures: jest.fn((promise) => promise),
  assembleModArchBody: jest.fn((data) => data),
  isModArchResponse: jest.fn(() => true),
}));

describe('Model Registry API', () => {
  const hostPath = '/api/v1/model_registry/test';
  const mockOpts = { signal: new AbortController().signal };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('createRegisteredModel', () => {
    it('should call restCREATE with correct parameters', async () => {
      const mockResponse = { data: { id: '1', name: 'test-model' } };
      (restCREATE as jest.Mock).mockResolvedValue(mockResponse);

      const result = await createRegisteredModel(hostPath)(mockOpts, {
        name: 'test-model',
      });

      expect(restCREATE).toHaveBeenCalledWith(
        hostPath,
        '/registered_models',
        { name: 'test-model' },
        {},
        mockOpts,
      );
      expect(result).toEqual(mockResponse.data);
    });
  });

  describe('getRegisteredModel', () => {
    it('should call restGET with correct parameters', async () => {
      const mockResponse = { data: { id: '1', name: 'test-model' } };
      (restGET as jest.Mock).mockResolvedValue(mockResponse);

      const result = await getRegisteredModel(hostPath)(mockOpts, '1');

      expect(restGET).toHaveBeenCalledWith(
        hostPath,
        '/registered_models/1',
        {},
        mockOpts,
      );
      expect(result).toEqual(mockResponse.data);
    });
  });
});
```

---

[Back to Frontend Index](./README.md) | [Previous: Routing](./routing.md) | [Next: Testing](./testing.md)

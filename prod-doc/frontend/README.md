# Frontend Documentation

This section covers the React-based frontend for the Kubeflow Model Registry.

## Overview

The frontend is a modern React 18 single-page application (SPA) built with TypeScript and PatternFly/MUI component libraries. It provides the user interface for:

- Model Registry management
- Model Catalog discovery
- MCP Catalog browsing
- Settings and configuration

## Technology Stack

| Technology | Version | Purpose |
|------------|---------|---------|
| React | 18.x | UI framework |
| TypeScript | 5.8.2 | Type safety |
| React Router | 7.5.2 | Client-side routing |
| PatternFly | 6.4.0 | Primary component library |
| Material-UI | 7.3.4 | Additional components |
| Webpack | 5.97.1 | Build and bundling |
| Jest | 29.7.0 | Unit testing |
| Cypress | - | E2E testing |

## Documentation

| Document | Description |
|----------|-------------|
| [Architecture](./architecture.md) | Directory structure and component organization |
| [State Management](./state-management.md) | Context API patterns and hooks |
| [Component Library](./component-library.md) | PatternFly and MUI usage |
| [Routing](./routing.md) | React Router configuration |
| [API Integration](./api-integration.md) | REST API client patterns |
| [Testing](./testing.md) | Jest and Cypress testing setup |

## Directory Structure

```
clients/ui/frontend/src/
├── app/                       # Main application code
│   ├── api/                   # API client functions
│   ├── context/               # React Context providers
│   ├── hooks/                 # Custom React hooks
│   ├── pages/                 # Page components
│   ├── routes/                # Route utilities
│   ├── shared/                # Shared components
│   ├── standalone/            # Standalone mode components
│   └── utilities/             # Constants and utilities
├── concepts/                  # Domain-specific types
├── bootstrap.tsx              # React entry point
└── index.ts                   # Module entry
```

## Quick Start

### Development

```bash
cd clients/ui/frontend

# Install dependencies
npm install

# Start development server
npm run start:dev

# Run tests
npm test

# Type checking
npm run test:type-check
```

### Build

```bash
# Production build
npm run build

# Output: dist/
```

## Key Patterns

### Context API Pattern

```typescript
// Context provides both data and actions
type AppContextType = {
  // Data
  models: Model[];
  loading: boolean;
  error: Error | null;

  // Actions
  refreshModels: () => void;
  updateModel: (id: string, data: Partial<Model>) => void;
};
```

### API Pattern

```typescript
// Higher-order function pattern for API calls
const getModel = (hostPath: string) =>
  (opts: APIOptions, id: string): Promise<Model> =>
    handleRestFailures(restGET(hostPath, `/models/${id}`, {}, opts));
```

### Hook Pattern

```typescript
// Custom hooks abstract API state management
const useModels = () => {
  const { apiState } = useContext(ModelContext);
  const [models, setModels] = useState<Model[]>([]);

  useEffect(() => {
    if (apiState.api) {
      apiState.api.getModels().then(setModels);
    }
  }, [apiState.api]);

  return { models, loading: !apiState.apiAvailable };
};
```

## Deployment Modes

The frontend adapts to different deployment modes:

| Mode | Description | Features |
|------|-------------|----------|
| **Kubeflow** | Integrated with Kubeflow dashboard | Uses mod-arch-kubeflow |
| **Standalone** | Independent deployment | NavBar, full routing |
| **Federated** | Multi-registry mode | Registry selector |

## Component Hierarchy

```
App
├── NavBar (standalone only)
├── AppNavSidebar
│   └── Navigation links
└── Routes
    ├── ModelRegistryRoutes
    │   ├── RegisteredModels
    │   ├── ModelVersions
    │   └── ModelArtifacts
    ├── ModelCatalogRoutes
    │   ├── CatalogGallery
    │   └── ModelDetails
    ├── McpCatalogRoutes
    │   ├── McpGallery
    │   └── ServerDetails
    └── SettingsRoutes
```

---

[Back to Main Index](../README.md)

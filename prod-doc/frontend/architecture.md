# Frontend Architecture

This document covers the directory structure and component organization of the frontend.

## Directory Overview

```
clients/ui/frontend/src/
├── app/                           # Main application directory
│   ├── api/                       # API integration layer
│   │   ├── mcpCatalog/           # MCP Catalog API services
│   │   ├── modelCatalog/         # Model Catalog API services
│   │   ├── modelCatalogSettings/ # Settings API
│   │   ├── service.ts            # Model Registry API
│   │   ├── k8s.ts                # Kubernetes API
│   │   └── __tests__/            # API unit tests
│   │
│   ├── context/                   # React Context providers
│   │   ├── AppContext.ts         # Global app context
│   │   ├── ModelRegistryContext.tsx
│   │   ├── ModelRegistrySelectorContext.tsx
│   │   ├── mcpCatalog/           # MCP Catalog contexts
│   │   ├── modelCatalog/         # Model Catalog contexts
│   │   └── modelCatalogSettings/ # Settings contexts
│   │
│   ├── hooks/                     # Custom React hooks (31+)
│   │   ├── useModelRegistryAPI.ts
│   │   ├── useModelRegistryAPIState.tsx
│   │   ├── useRegisteredModels.ts
│   │   ├── modelCatalog/         # Catalog-specific hooks
│   │   ├── mcpCatalog/           # MCP-specific hooks
│   │   └── __tests__/
│   │
│   ├── pages/                     # Page components
│   │   ├── modelRegistry/        # Model Registry pages
│   │   │   ├── screens/          # Top-level screens
│   │   │   └── components/       # Page-specific components
│   │   ├── modelCatalog/         # Model Catalog pages
│   │   ├── mcpCatalog/           # MCP Catalog pages
│   │   ├── modelCatalogSettings/ # Settings pages
│   │   └── settings/             # General settings
│   │
│   ├── routes/                    # Route utilities
│   ├── shared/                    # Shared components
│   │   ├── components/           # Reusable UI components
│   │   └── markdown/             # Markdown rendering
│   │
│   ├── standalone/               # Standalone mode components
│   │   ├── NavBar.tsx
│   │   ├── AppNavSidebar.tsx
│   │   └── ToastNotifications.tsx
│   │
│   ├── utilities/                # Constants and utilities
│   │   └── const.ts
│   │
│   ├── App.tsx                   # Root component
│   ├── AppRoutes.tsx             # Route configuration
│   ├── types.ts                  # Core type definitions
│   └── modelCatalogTypes.ts      # Catalog types
│
├── concepts/                      # Domain-specific types
│   ├── modelCatalog/
│   │   └── const.ts              # Enums and constants
│   ├── modelRegistry/
│   ├── k8s/
│   └── modelCatalogSettings/
│
├── bootstrap.tsx                 # React DOM initialization
├── index.ts                      # Entry point
└── index.html                    # HTML template
```

## Entry Point Flow

```
index.ts
    │
    └──> bootstrap.tsx
            │
            ├──> Creates React root with createRoot()
            └──> Renders RootLayout
                    │
                    ├──> ModularArchContextProvider (config)
                    ├──> ThemeProvider (PatternFly/MUI)
                    ├──> BrowserStorageContextProvider
                    ├──> NotificationContextProvider
                    └──> RouterProvider
                            │
                            └──> App.tsx
```

### bootstrap.tsx

```typescript
// Entry point that sets up all providers
import { createRoot } from 'react-dom/client';
import { RouterProvider, createBrowserRouter } from 'react-router-dom';
import { ModularArchContextProvider } from 'mod-arch-core';

const router = createBrowserRouter([...]);

const RootLayout: React.FC = () => (
  <ModularArchContextProvider config={modArchConfig}>
    <ThemeProvider>
      <BrowserStorageContextProvider>
        <NotificationContextProvider>
          <RouterProvider router={router} />
        </NotificationContextProvider>
      </BrowserStorageContextProvider>
    </ThemeProvider>
  </ModularArchContextProvider>
);

createRoot(document.getElementById('root')!).render(<RootLayout />);
```

### App.tsx

```typescript
// Root application component
const App: React.FC = () => {
  const { config, user } = useAppContext();

  return (
    <AppContext.Provider value={{ config, user }}>
      <ModelRegistrySelectorContextProvider>
        {isStandalone && <NavBar />}
        <Page sidebar={<AppNavSidebar />}>
          <Outlet />  {/* React Router renders routes here */}
        </Page>
      </ModelRegistrySelectorContextProvider>
    </AppContext.Provider>
  );
};
```

## Component Organization

### Page Components

Pages are organized by feature with a consistent structure:

```
pages/modelCatalog/
├── ModelCatalogCoreLoader.tsx     # Data loading wrapper
├── ModelCatalogRoutes.tsx         # Route definitions
├── EmptyModelCatalogState.tsx     # Empty state
├── screens/
│   ├── ModelCatalog.tsx           # Main catalog page
│   ├── ModelCatalogGalleryView.tsx
│   └── ModelDetailsPage.tsx
└── components/
    ├── ModelCatalogCard.tsx
    ├── ModelCatalogFilters.tsx
    └── ModelCatalogLabels.tsx
```

### Component Patterns

**Screen Components** - Full page views:

```typescript
const ModelCatalog: React.FC = () => {
  const { models, filters, updateFilters } = useModelCatalogContext();

  return (
    <PageSection>
      <Sidebar>
        <SidebarPanel>
          <ModelCatalogFilters />
        </SidebarPanel>
        <SidebarContent>
          <ModelCatalogGalleryView />
        </SidebarContent>
      </Sidebar>
    </PageSection>
  );
};
```

**Card Components** - Individual item displays:

```typescript
const ModelCatalogCard: React.FC<{ model: CatalogModel }> = ({ model }) => (
  <Card>
    <CardHeader>
      <CardTitle>{model.name}</CardTitle>
    </CardHeader>
    <CardBody>
      <ModelCatalogLabels labels={model.labels} />
      <p>{model.description}</p>
    </CardBody>
    <CardFooter>
      <Button component={Link} to={`/model-catalog/${model.name}`}>
        View Details
      </Button>
    </CardFooter>
  </Card>
);
```

**Filter Components** - User input for filtering:

```typescript
const ModelCatalogFilters: React.FC = () => {
  const { filters, updateFilters, filterOptions } = useModelCatalogContext();

  return (
    <>
      <SelectFilter
        label="Provider"
        value={filters.provider}
        options={filterOptions.providers}
        onChange={(value) => updateFilters({ ...filters, provider: value })}
      />
      <SelectFilter
        label="License"
        value={filters.license}
        options={filterOptions.licenses}
        onChange={(value) => updateFilters({ ...filters, license: value })}
      />
    </>
  );
};
```

## Shared Components

Located in `app/shared/components/`:

| Component | Purpose |
|-----------|---------|
| `ErrorBoundary` | Catches and displays errors |
| `LoadingSpinner` | Consistent loading indicator |
| `ConfirmModal` | Confirmation dialogs |
| `SearchInput` | Search input with debounce |
| `Pagination` | Page navigation |

## Concepts Directory

Domain-specific constants and types:

```typescript
// concepts/modelCatalog/const.ts
export enum ModelCatalogTask {
  TEXT_GENERATION = 'text-generation',
  CLASSIFICATION = 'classification',
  // ...
}

export enum ModelCatalogLicense {
  APACHE_2_0 = 'apache-2.0',
  MIT = 'mit',
  // ...
}

export const taskDisplayNames: Record<ModelCatalogTask, string> = {
  [ModelCatalogTask.TEXT_GENERATION]: 'Text Generation',
  [ModelCatalogTask.CLASSIFICATION]: 'Classification',
  // ...
};
```

## Type Definitions

### Core Types (app/types.ts)

```typescript
// Model Registry domain types
export interface RegisteredModel {
  id: string;
  name: string;
  description?: string;
  state: ModelState;
  createTimeSinceEpoch: string;
  lastUpdateTimeSinceEpoch: string;
  customProperties?: Record<string, MetadataValue>;
}

export interface ModelVersion {
  id: string;
  name: string;
  registeredModelId: string;
  state: ModelVersionState;
  author?: string;
  // ...
}

export interface ModelArtifact {
  id: string;
  name: string;
  uri: string;
  state: ArtifactState;
  // ...
}
```

### Catalog Types (app/modelCatalogTypes.ts)

```typescript
// Model Catalog domain types
export interface CatalogModel {
  name: string;
  description?: string;
  provider?: string;
  license?: string;
  tasks?: string[];
  customProperties?: Record<string, MetadataValue>;
}

export interface CatalogSource {
  id: string;
  name: string;
  enabled: boolean;
  labels?: string[];
}
```

## Build Configuration

### Webpack Configuration

```javascript
// config/webpack.dev.js
module.exports = merge(common, {
  mode: 'development',
  devServer: {
    port: 9000,
    hot: true,
    historyApiFallback: true,
    proxy: {
      '/api': {
        target: 'http://localhost:4000',
        changeOrigin: true,
      },
    },
  },
});
```

### TypeScript Configuration

```json
// tsconfig.json
{
  "compilerOptions": {
    "target": "ES2021",
    "module": "ESNext",
    "strict": true,
    "esModuleInterop": true,
    "paths": {
      "~/*": ["./src/*"]
    }
  }
}
```

## Dependencies

### Core Dependencies

- **react**: UI framework
- **react-dom**: DOM rendering
- **react-router-dom**: Client-side routing
- **@patternfly/react-core**: PatternFly components
- **@patternfly/react-table**: Table components
- **@patternfly/react-icons**: Icons
- **@mui/material**: Material-UI components

### Mod-Arch Dependencies

- **mod-arch-core**: Core utilities and components
- **mod-arch-kubeflow**: Kubeflow integration
- **mod-arch-shared**: Shared utilities

### Build Dependencies

- **webpack**: Bundler
- **typescript**: Type checking
- **@swc/core**: Fast transpilation
- **sass**: Styling

---

[Back to Frontend Index](./README.md) | [Next: State Management](./state-management.md)

# Routing

This document covers the React Router configuration and navigation patterns.

## Overview

The frontend uses React Router v7.x with the data router pattern (`createBrowserRouter`).

## Route Configuration

### Main Routes (AppRoutes.tsx)

```typescript
// app/AppRoutes.tsx
import { createBrowserRouter, Navigate } from 'react-router-dom';

const router = createBrowserRouter([
  {
    path: '/',
    element: <App />,
    children: [
      // Redirect root to model registry
      {
        index: true,
        element: <Navigate to="/model-registry" replace />,
      },

      // Model Registry routes
      {
        path: 'model-registry/*',
        element: <ModelRegistryRoutes />,
      },

      // Model Catalog routes (Standalone/Federated only)
      {
        path: 'model-catalog/*',
        element: <ModelCatalogRoutes />,
      },

      // MCP Catalog routes (Standalone/Federated only)
      {
        path: 'mcp-catalog/*',
        element: <McpCatalogRoutes />,
      },

      // Settings routes
      {
        path: 'model-registry-settings/*',
        element: <ModelRegistrySettingsRoutes />,
      },

      // Model Catalog Settings (admin only)
      {
        path: 'model-catalog-settings/*',
        element: <ModelCatalogSettingsRoutes />,
      },
    ],
  },
]);
```

## Feature Routes

### Model Registry Routes

```typescript
// app/pages/modelRegistry/ModelRegistryRoutes.tsx
const ModelRegistryRoutes: React.FC = () => (
  <Routes>
    {/* Registered Models list */}
    <Route index element={<Navigate to="registered-models" replace />} />
    <Route path="registered-models" element={<RegisteredModels />} />

    {/* Model details */}
    <Route path="registered-models/:modelId" element={<RegisteredModelDetails />} />

    {/* Model versions */}
    <Route
      path="registered-models/:modelId/versions"
      element={<ModelVersions />}
    />
    <Route
      path="registered-models/:modelId/versions/:versionId"
      element={<ModelVersionDetails />}
    />

    {/* Artifacts */}
    <Route
      path="registered-models/:modelId/versions/:versionId/artifacts"
      element={<ModelArtifacts />}
    />
    <Route
      path="registered-models/:modelId/versions/:versionId/artifacts/:artifactId"
      element={<ModelArtifactDetails />}
    />
  </Routes>
);
```

### Model Catalog Routes

```typescript
// app/pages/modelCatalog/ModelCatalogRoutes.tsx
const ModelCatalogRoutes: React.FC = () => (
  <ModelCatalogCoreLoader>
    <Routes>
      {/* Catalog gallery */}
      <Route index element={<ModelCatalog />} />

      {/* Model details with tabs */}
      <Route path=":modelName" element={<ModelDetailsPage />}>
        <Route index element={<Navigate to="overview" replace />} />
        <Route path="overview" element={<ModelOverview />} />
        <Route path="artifacts" element={<ModelArtifacts />} />
        <Route path="performance" element={<ModelPerformance />} />
      </Route>
    </Routes>
  </ModelCatalogCoreLoader>
);
```

### MCP Catalog Routes

```typescript
// app/pages/mcpCatalog/McpCatalogRoutes.tsx
const McpCatalogRoutes: React.FC = () => (
  <McpCatalogCoreLoader>
    <Routes>
      {/* MCP gallery */}
      <Route index element={<McpCatalog />} />

      {/* Server details */}
      <Route path=":serverId" element={<McpServerDetailsPage />} />
    </Routes>
  </McpCatalogCoreLoader>
);
```

## Route Parameters

### Accessing Parameters

```typescript
import { useParams } from 'react-router-dom';

const ModelVersionDetails: React.FC = () => {
  const { modelId, versionId } = useParams<{
    modelId: string;
    versionId: string;
  }>();

  return (
    <div>
      <h1>Model: {modelId}</h1>
      <h2>Version: {versionId}</h2>
    </div>
  );
};
```

### Query Parameters

```typescript
import { useSearchParams } from 'react-router-dom';

const ModelCatalog: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  const provider = searchParams.get('provider');
  const search = searchParams.get('q');

  const updateFilter = (key: string, value: string) => {
    setSearchParams(prev => {
      if (value) {
        prev.set(key, value);
      } else {
        prev.delete(key);
      }
      return prev;
    });
  };

  return (
    <div>
      <input
        value={search || ''}
        onChange={e => updateFilter('q', e.target.value)}
      />
    </div>
  );
};
```

## Navigation

### Link Component

```typescript
import { Link } from 'react-router-dom';

const ModelCard: React.FC<{ model: Model }> = ({ model }) => (
  <Card>
    <CardBody>
      <Link to={`/model-registry/registered-models/${model.id}`}>
        {model.name}
      </Link>
    </CardBody>
  </Card>
);
```

### Programmatic Navigation

```typescript
import { useNavigate } from 'react-router-dom';

const CreateModelForm: React.FC = () => {
  const navigate = useNavigate();

  const handleSubmit = async (data: CreateModelData) => {
    const model = await createModel(data);
    navigate(`/model-registry/registered-models/${model.id}`);
  };

  const handleCancel = () => {
    navigate(-1); // Go back
  };

  return (
    <form onSubmit={handleSubmit}>
      {/* form fields */}
      <Button type="submit">Create</Button>
      <Button variant="link" onClick={handleCancel}>Cancel</Button>
    </form>
  );
};
```

### NavLink for Active States

```typescript
import { NavLink } from 'react-router-dom';

const SidebarNav: React.FC = () => (
  <Nav>
    <NavList>
      <NavItem>
        <NavLink
          to="/model-registry"
          className={({ isActive }) => isActive ? 'pf-m-current' : ''}
        >
          Model Registry
        </NavLink>
      </NavItem>
      <NavItem>
        <NavLink
          to="/model-catalog"
          className={({ isActive }) => isActive ? 'pf-m-current' : ''}
        >
          Model Catalog
        </NavLink>
      </NavItem>
    </NavList>
  </Nav>
);
```

## Deployment Mode Routing

Routes adapt based on deployment mode (Kubeflow, Standalone, Federated).

### Deployment Mode Context

```typescript
import { useModularArchContext } from 'mod-arch-core';

const AppRoutes: React.FC = () => {
  const { config } = useModularArchContext();
  const isStandalone = config.deploymentMode === 'standalone';
  const isFederated = config.deploymentMode === 'federated';

  return (
    <Routes>
      <Route path="/model-registry/*" element={<ModelRegistryRoutes />} />

      {/* Catalog routes only for Standalone/Federated */}
      {(isStandalone || isFederated) && (
        <>
          <Route path="/model-catalog/*" element={<ModelCatalogRoutes />} />
          <Route path="/mcp-catalog/*" element={<McpCatalogRoutes />} />
        </>
      )}

      {/* Settings routes only for admins */}
      {config.isAdmin && (
        <Route path="/settings/*" element={<SettingsRoutes />} />
      )}
    </Routes>
  );
};
```

### Navigation Sidebar

```typescript
// app/standalone/AppNavSidebar.tsx
const AppNavSidebar: React.FC = () => {
  const { config } = useModularArchContext();
  const location = useLocation();

  const navItems = [
    { label: 'Model Registry', path: '/model-registry', icon: CubeIcon },
  ];

  // Add catalog routes for non-Kubeflow deployments
  if (config.deploymentMode !== 'kubeflow') {
    navItems.push(
      { label: 'Model Catalog', path: '/model-catalog', icon: SearchIcon },
      { label: 'MCP Catalog', path: '/mcp-catalog', icon: PlugIcon },
    );
  }

  // Add settings for admins
  if (config.isAdmin) {
    navItems.push(
      { label: 'Settings', path: '/settings', icon: CogIcon },
    );
  }

  return (
    <PageSidebar>
      <Nav>
        <NavList>
          {navItems.map(item => (
            <NavItem
              key={item.path}
              isActive={location.pathname.startsWith(item.path)}
            >
              <Link to={item.path}>
                <item.icon /> {item.label}
              </Link>
            </NavItem>
          ))}
        </NavList>
      </Nav>
    </PageSidebar>
  );
};
```

## Breadcrumbs

### Breadcrumb Component

```typescript
import { Breadcrumb, BreadcrumbItem } from '@patternfly/react-core';
import { Link, useParams } from 'react-router-dom';

const ModelVersionBreadcrumb: React.FC = () => {
  const { modelId, versionId } = useParams();

  return (
    <Breadcrumb>
      <BreadcrumbItem>
        <Link to="/model-registry">Model Registry</Link>
      </BreadcrumbItem>
      <BreadcrumbItem>
        <Link to="/model-registry/registered-models">Registered Models</Link>
      </BreadcrumbItem>
      <BreadcrumbItem>
        <Link to={`/model-registry/registered-models/${modelId}`}>
          {modelId}
        </Link>
      </BreadcrumbItem>
      <BreadcrumbItem isActive>
        Version {versionId}
      </BreadcrumbItem>
    </Breadcrumb>
  );
};
```

## Route Guards

### Protected Routes

```typescript
const ProtectedRoute: React.FC<PropsWithChildren<{ requiredRole?: string }>> = ({
  children,
  requiredRole,
}) => {
  const { user } = useAppContext();
  const location = useLocation();

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  if (requiredRole && !user.roles.includes(requiredRole)) {
    return <Navigate to="/unauthorized" replace />;
  }

  return <>{children}</>;
};

// Usage
<Route
  path="/settings/*"
  element={
    <ProtectedRoute requiredRole="admin">
      <SettingsRoutes />
    </ProtectedRoute>
  }
/>
```

## Data Loading

### Loader Pattern

```typescript
// Using React Router loaders
const modelLoader = async ({ params }: LoaderFunctionArgs) => {
  const { modelId } = params;
  const model = await fetchModel(modelId!);
  return { model };
};

const routes = createBrowserRouter([
  {
    path: '/model-registry/registered-models/:modelId',
    element: <ModelDetails />,
    loader: modelLoader,
  },
]);

// In component
const ModelDetails: React.FC = () => {
  const { model } = useLoaderData() as { model: Model };

  return <div>{model.name}</div>;
};
```

### Context-Based Loading

```typescript
// Using context providers for data loading
const ModelCatalogCoreLoader: React.FC<PropsWithChildren> = ({ children }) => {
  const { apiState } = useModelCatalogContext();

  if (!apiState.apiAvailable) {
    return <LoadingSpinner message="Loading catalog..." />;
  }

  return <>{children}</>;
};
```

---

[Back to Frontend Index](./README.md) | [Previous: Component Library](./component-library.md) | [Next: API Integration](./api-integration.md)

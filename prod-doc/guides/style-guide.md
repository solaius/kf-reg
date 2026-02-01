# Style Guide

This document outlines the code style standards for the Kubeflow Model Registry project.

## Go Style

### Linting

The project uses `golangci-lint` for Go code quality:

```bash
# Run linter
make lint

# BFF-specific linting
cd clients/ui/bff
make lint
```

### Configuration

**BFF golangci-lint config** (`.golangci.yaml`):

```yaml
version: "2"
linters:
  exclusions:
    generated: lax
    presets:
      - comments
      - common-false-positives
      - legacy
      - std-error-handling
    paths:
      - third_party$
      - builtin$
      - examples$
```

### Go Conventions

#### Package Naming

```go
// Good
package kubernetes
package httpclient
package validation

// Avoid
package k8s      // Too abbreviated
package httpClient // Mixed case
```

#### Error Handling

```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to get model registry: %w", err)
}

// Avoid: Naked returns
if err != nil {
    return err
}
```

#### Interface Design

```go
// Good: Small, focused interfaces
type ModelReader interface {
    GetModel(id string) (*Model, error)
    ListModels() ([]*Model, error)
}

// Avoid: Large, monolithic interfaces
type ModelService interface {
    GetModel(id string) (*Model, error)
    ListModels() ([]*Model, error)
    CreateModel(m *Model) error
    UpdateModel(m *Model) error
    DeleteModel(id string) error
    // ... many more methods
}
```

#### Naming Conventions

```go
// Exported names
type RegisteredModel struct {}
func GetAllModelVersions() {}

// Unexported names
type internalKubernetesClient struct {}
func parseModelVersion() {}

// Constants
const DefaultPageSize = 100
const modelRegistryAPIVersion = "v1alpha3"

// Acronyms: consistent case
type HTTPClient struct {}  // Not HttpClient
func GetAPIEndpoint() {}   // Not GetApiEndpoint
```

#### Struct Tags

```go
type RegisteredModel struct {
    ID          string `json:"id,omitempty"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    State       State  `json:"state"`
}
```

## TypeScript/React Style

### Linting and Formatting

The frontend uses ESLint and Prettier:

```bash
cd clients/ui/frontend

# Lint
npm run lint

# Format
npm run format
```

### Configuration

**ESLint** (`.eslintrc.cjs`):
- Parser: `@typescript-eslint/parser`
- Extends: `eslint:recommended`, `plugin:react/recommended`, `plugin:@typescript-eslint/recommended`
- Plugins: `react-hooks`, `import`, `prettier`

**Prettier** (`.prettierrc`):
```json
{
  "arrowParens": "always",
  "printWidth": 100,
  "singleQuote": true,
  "trailingComma": "all"
}
```

### TypeScript Conventions

#### Naming Conventions

```typescript
// Variables: camelCase
const modelVersion = getVersion();
const isLoading = true;

// Functions: camelCase
function fetchRegisteredModels() {}
const handleSubmit = () => {};

// Types/Interfaces: PascalCase
interface RegisteredModel {}
type ModelState = 'LIVE' | 'ARCHIVED';

// Constants: UPPER_CASE
const API_BASE_URL = '/api/v1';
const DEFAULT_PAGE_SIZE = 20;

// Components: PascalCase
function ModelVersionTable() {}
const RegisteredModelCard: React.FC = () => {};
```

#### Type Annotations

```typescript
// Good: Explicit return types for exported functions
export function getModelById(id: string): Promise<RegisteredModel | null> {
  // ...
}

// Good: Interface for props
interface ModelCardProps {
  model: RegisteredModel;
  onSelect?: (id: string) => void;
}

// Avoid: any type
function processData(data: any) {} // Bad
function processData(data: unknown) {} // Better
```

#### Import Organization

```typescript
// 1. External packages
import React from 'react';
import { useNavigate } from 'react-router-dom';

// 2. Internal absolute imports (~/)
import { useModelRegistryContext } from '~/context/ModelRegistryContext';
import { RegisteredModel } from '~/types';

// 3. Relative imports (same folder only)
import './ModelCard.css';
import { formatDate } from './utils';
```

#### React Patterns

```typescript
// Good: Use destructuring
const ModelCard: React.FC<ModelCardProps> = ({ model, onSelect }) => {
  // ...
};

// Good: Self-closing components
<EmptyState />

// Avoid: Boolean props with true
<Button isDisabled={true} />  // Bad
<Button isDisabled />         // Good

// Good: Avoid constructed context values
const contextValue = useMemo(() => ({ model, updateModel }), [model, updateModel]);
```

### Disallowed Patterns

#### No console.log

```typescript
// Error: no-console
console.log(data);

// Use proper logging or remove
```

#### No .only() in Tests

```typescript
// Error: no-only-tests
it.only('should work', () => {});  // Bad
it('should work', () => {});       // Good
```

#### Use toSorted instead of sort

```typescript
// Error: no-restricted-properties
array.sort();                    // Bad - mutates
array.toSorted();                // Good - returns new array
```

#### No Type Assertions (Outside Tests)

```typescript
// Error in non-test files
const model = data as RegisteredModel;  // Bad

// Use type guards or proper typing
function isRegisteredModel(data: unknown): data is RegisteredModel {
  return typeof data === 'object' && data !== null && 'name' in data;
}
```

## File Organization

### Go Files

```
internal/
├── api/
│   ├── app.go                    # Main app setup
│   ├── middleware.go             # HTTP middleware
│   ├── errors.go                 # Error responses
│   ├── helpers.go                # Utility functions
│   ├── registered_models_handler.go
│   ├── model_versions_handler.go
│   └── *_test.go                 # Tests alongside code
├── config/
│   └── environment.go            # Configuration
└── models/
    └── types.go                  # Data models
```

### TypeScript Files

```
src/
├── app/
│   ├── pages/                    # Route pages
│   │   ├── modelRegistry/
│   │   │   ├── ModelRegistryPage.tsx
│   │   │   ├── components/       # Page-specific components
│   │   │   └── hooks/            # Page-specific hooks
│   ├── components/               # Shared components
│   ├── context/                  # React contexts
│   └── hooks/                    # Shared hooks
├── api/                          # API client
├── types/                        # TypeScript types
└── utils/                        # Utility functions
```

## Comments

### Go Comments

```go
// Package kubernetes provides Kubernetes client integration.
package kubernetes

// GetKubernetesClient returns a configured Kubernetes client.
// It uses in-cluster configuration when available, otherwise
// falls back to kubeconfig.
func GetKubernetesClient() (*Client, error) {
    // ...
}
```

### TypeScript Comments

```typescript
/**
 * Fetches registered models from the API.
 * @param params - Query parameters for filtering
 * @returns Promise resolving to model list
 */
export async function getRegisteredModels(
  params: ListParams
): Promise<RegisteredModelList> {
  // ...
}
```

### Comment Guidelines

- Comment exported functions/types
- Explain **why**, not **what**
- Keep comments up to date
- Don't comment obvious code
- Use TODO format: `// TODO(username): description`

## Testing Style

### Go Tests

```go
func TestGetRegisteredModel(t *testing.T) {
    // Arrange
    client := NewMockClient()
    service := NewService(client)

    // Act
    model, err := service.GetRegisteredModel("1")

    // Assert
    require.NoError(t, err)
    assert.Equal(t, "test-model", model.Name)
}
```

### TypeScript Tests

```typescript
describe('ModelCard', () => {
  it('should render model name', () => {
    // Arrange
    const model = createMockModel({ name: 'Test Model' });

    // Act
    render(<ModelCard model={model} />);

    // Assert
    expect(screen.getByText('Test Model')).toBeInTheDocument();
  });
});
```

## Documentation Style

### API Documentation

Use OpenAPI specifications in `api/openapi/`.

### Code Documentation

- README.md in each major directory
- Inline comments for complex logic
- Examples in documentation

---

[Back to Guides Index](./README.md) | [Previous: Contributor Requirements](./contributor-requirements.md) | [Next: UI Design Requirements](./ui-design-requirements.md)

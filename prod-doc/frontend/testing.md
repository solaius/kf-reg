# Testing

This document covers the testing setup and patterns for the frontend.

## Overview

The frontend uses a comprehensive testing strategy:

| Type | Framework | Purpose |
|------|-----------|---------|
| Unit Tests | Jest + RTL | Component and hook testing |
| Integration Tests | Jest | API and context integration |
| E2E Tests | Cypress | Full user flow testing |
| Type Checking | TypeScript | Static type validation |
| Linting | ESLint | Code quality |

## Jest Configuration

### jest.config.js

```javascript
module.exports = {
  testEnvironment: 'jest-environment-jsdom',

  testMatch: [
    '**/src/__tests__/unit/**/?(*.)+(spec|test).ts?(x)',
    '**/__tests__/?(*.)+(spec|test).ts?(x)',
  ],

  setupFilesAfterEnv: ['<rootDir>/src/__tests__/unit/jest.setup.ts'],

  moduleNameMapper: {
    // Handle CSS imports
    '\\.(css|less|sass|scss)$': '<rootDir>/config/transform.style.js',
    // Path aliases
    '~/(.*)': '<rootDir>/src/$1',
  },

  transformIgnorePatterns: [
    'node_modules/(?!yaml|lodash-es|uuid|@patternfly|delaunator|mod-arch-shared|mod-arch-core|mod-arch-kubeflow)',
  ],

  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/__tests__/**',
    '!src/__mocks__/**',
  ],
};
```

### Jest Setup

```typescript
// src/__tests__/unit/jest.setup.ts
import '@testing-library/jest-dom';

// Mock ResizeObserver
global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// Mock matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: jest.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: jest.fn(),
    removeListener: jest.fn(),
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
  })),
});
```

## Unit Testing

### Component Tests

```typescript
// app/pages/modelCatalog/components/__tests__/ModelCatalogCard.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import ModelCatalogCard from '../ModelCatalogCard';

const mockModel = {
  name: 'test-model',
  description: 'A test model',
  provider: 'Test Provider',
  license: 'apache-2.0',
  tasks: ['text-generation', 'classification'],
};

const renderWithRouter = (component: React.ReactElement) => {
  return render(
    <BrowserRouter>
      {component}
    </BrowserRouter>
  );
};

describe('ModelCatalogCard', () => {
  it('renders model information correctly', () => {
    renderWithRouter(<ModelCatalogCard model={mockModel} />);

    expect(screen.getByText('test-model')).toBeInTheDocument();
    expect(screen.getByText('Test Provider')).toBeInTheDocument();
    expect(screen.getByText('A test model')).toBeInTheDocument();
  });

  it('displays task labels', () => {
    renderWithRouter(<ModelCatalogCard model={mockModel} />);

    expect(screen.getByText('text-generation')).toBeInTheDocument();
    expect(screen.getByText('classification')).toBeInTheDocument();
  });

  it('calls onClick when clicked', () => {
    const handleClick = jest.fn();
    renderWithRouter(<ModelCatalogCard model={mockModel} onClick={handleClick} />);

    fireEvent.click(screen.getByRole('article'));
    expect(handleClick).toHaveBeenCalled();
  });

  it('renders without logo when not provided', () => {
    renderWithRouter(<ModelCatalogCard model={mockModel} />);

    expect(screen.queryByRole('img')).not.toBeInTheDocument();
  });
});
```

### Hook Tests

```typescript
// app/hooks/__tests__/useRegisteredModels.test.tsx
import { renderHook, waitFor } from '@testing-library/react';
import { ModelRegistryContext } from '~/app/context/ModelRegistryContext';
import useRegisteredModels from '../useRegisteredModels';

const mockApi = {
  getRegisteredModels: jest.fn(),
};

const mockApiState = {
  apiAvailable: true,
  api: mockApi,
};

const wrapper: React.FC<React.PropsWithChildren> = ({ children }) => (
  <ModelRegistryContext.Provider value={{ apiState: mockApiState, refreshAPIState: jest.fn() }}>
    {children}
  </ModelRegistryContext.Provider>
);

describe('useRegisteredModels', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('fetches models on mount', async () => {
    const mockModels = [
      { id: '1', name: 'model-1' },
      { id: '2', name: 'model-2' },
    ];
    mockApi.getRegisteredModels.mockResolvedValue({ items: mockModels });

    const { result } = renderHook(() => useRegisteredModels(), { wrapper });

    await waitFor(() => {
      expect(result.current[1]).toBe(true); // loaded
    });

    expect(result.current[0]).toEqual(mockModels);
    expect(mockApi.getRegisteredModels).toHaveBeenCalled();
  });

  it('handles errors', async () => {
    const error = new Error('API Error');
    mockApi.getRegisteredModels.mockRejectedValue(error);

    const { result } = renderHook(() => useRegisteredModels(), { wrapper });

    await waitFor(() => {
      expect(result.current[2]).toBe(error); // error
    });
  });

  it('does not fetch when API is not available', () => {
    const unavailableWrapper: React.FC<React.PropsWithChildren> = ({ children }) => (
      <ModelRegistryContext.Provider
        value={{ apiState: { apiAvailable: false, api: null }, refreshAPIState: jest.fn() }}
      >
        {children}
      </ModelRegistryContext.Provider>
    );

    renderHook(() => useRegisteredModels(), { wrapper: unavailableWrapper });

    expect(mockApi.getRegisteredModels).not.toHaveBeenCalled();
  });
});
```

### API Function Tests

```typescript
// app/api/__tests__/service.spec.ts
import { restCREATE, restGET, restPATCH, handleRestFailures } from 'mod-arch-core';
import { createRegisteredModel, getRegisteredModel } from '../service';

jest.mock('mod-arch-core', () => ({
  restCREATE: jest.fn(),
  restGET: jest.fn(),
  restPATCH: jest.fn(),
  handleRestFailures: jest.fn((promise) => promise),
  assembleModArchBody: jest.fn((data) => data),
  isModArchResponse: jest.fn(() => true),
}));

describe('Model Registry API', () => {
  const hostPath = '/api/v1/model_registry/test';
  const mockOpts = {};

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('createRegisteredModel', () => {
    it('creates a model with correct endpoint', async () => {
      const mockResponse = { data: { id: '1', name: 'new-model' } };
      (restCREATE as jest.Mock).mockResolvedValue(mockResponse);

      const result = await createRegisteredModel(hostPath)(mockOpts, {
        name: 'new-model',
        description: 'A new model',
      });

      expect(restCREATE).toHaveBeenCalledWith(
        hostPath,
        '/registered_models',
        { name: 'new-model', description: 'A new model' },
        {},
        mockOpts,
      );
      expect(result).toEqual(mockResponse.data);
    });
  });

  describe('getRegisteredModel', () => {
    it('fetches a model by ID', async () => {
      const mockResponse = { data: { id: '123', name: 'test-model' } };
      (restGET as jest.Mock).mockResolvedValue(mockResponse);

      const result = await getRegisteredModel(hostPath)(mockOpts, '123');

      expect(restGET).toHaveBeenCalledWith(
        hostPath,
        '/registered_models/123',
        {},
        mockOpts,
      );
      expect(result).toEqual(mockResponse.data);
    });
  });
});
```

## Context Tests

```typescript
// app/context/__tests__/ModelRegistryContext.test.tsx
import { render, screen, waitFor } from '@testing-library/react';
import { ModelRegistryContextProvider, useModelRegistryContext } from '../ModelRegistryContext';

// Mock the API state hook
jest.mock('~/app/hooks/useModelRegistryAPIState', () => ({
  __esModule: true,
  default: jest.fn(() => [
    { apiAvailable: true, api: { getRegisteredModels: jest.fn() } },
    jest.fn(),
  ]),
}));

const TestConsumer: React.FC = () => {
  const { apiState } = useModelRegistryContext();
  return <div data-testid="api-available">{apiState.apiAvailable.toString()}</div>;
};

describe('ModelRegistryContext', () => {
  it('provides API state to consumers', () => {
    render(
      <ModelRegistryContextProvider hostPath="/api/test">
        <TestConsumer />
      </ModelRegistryContextProvider>
    );

    expect(screen.getByTestId('api-available')).toHaveTextContent('true');
  });

  it('throws error when used outside provider', () => {
    const consoleError = jest.spyOn(console, 'error').mockImplementation(() => {});

    expect(() => {
      render(<TestConsumer />);
    }).toThrow('useModelRegistryContext must be used within ModelRegistryContextProvider');

    consoleError.mockRestore();
  });
});
```

## Mocking

### Mock Data

```typescript
// src/__mocks__/mockModelData.ts
import { RegisteredModel, ModelVersion, ModelArtifact } from '~/app/types';

export const mockRegisteredModel: RegisteredModel = {
  id: '1',
  name: 'test-model',
  description: 'A test model for unit tests',
  state: 'LIVE',
  createTimeSinceEpoch: '1700000000000',
  lastUpdateTimeSinceEpoch: '1700000000000',
  customProperties: {},
};

export const mockModelVersion: ModelVersion = {
  id: '1',
  name: '1.0.0',
  registeredModelId: '1',
  state: 'LIVE',
  author: 'test-author',
  createTimeSinceEpoch: '1700000000000',
  lastUpdateTimeSinceEpoch: '1700000000000',
};

export const mockModelArtifact: ModelArtifact = {
  id: '1',
  name: 'model-artifact',
  uri: 's3://bucket/model.pkl',
  state: 'LIVE',
  modelFormatName: 'sklearn',
  modelFormatVersion: '1.0',
};
```

### Module Mocks

```typescript
// src/__mocks__/mod-arch-core.ts
export const restGET = jest.fn();
export const restCREATE = jest.fn();
export const restPATCH = jest.fn();
export const restDELETE = jest.fn();
export const handleRestFailures = jest.fn((promise) => promise);
export const assembleModArchBody = jest.fn((data) => data);
export const isModArchResponse = jest.fn(() => true);
export const useModularArchContext = jest.fn(() => ({
  config: { apiHost: 'http://localhost:4000' },
}));
export const useBrowserStorage = jest.fn((key, defaultValue) => [defaultValue, jest.fn()]);
```

## Cypress E2E Tests

### Configuration

```javascript
// cypress.config.ts
import { defineConfig } from 'cypress';

export default defineConfig({
  e2e: {
    baseUrl: 'http://localhost:9000',
    viewportWidth: 1280,
    viewportHeight: 720,
    video: false,
    screenshotOnRunFailure: true,
    defaultCommandTimeout: 10000,
    env: {
      MOCK: process.env.CY_MOCK === '1',
    },
  },
});
```

### E2E Test Example

```typescript
// cypress/e2e/modelRegistry.cy.ts
describe('Model Registry', () => {
  beforeEach(() => {
    cy.visit('/model-registry');
  });

  it('displays the registered models list', () => {
    cy.get('[data-testid="models-table"]').should('be.visible');
    cy.get('[data-testid="model-row"]').should('have.length.greaterThan', 0);
  });

  it('navigates to model details', () => {
    cy.get('[data-testid="model-row"]').first().click();
    cy.url().should('include', '/registered-models/');
    cy.get('[data-testid="model-name"]').should('be.visible');
  });

  it('creates a new model', () => {
    cy.get('[data-testid="create-model-btn"]').click();
    cy.get('[data-testid="model-name-input"]').type('cypress-test-model');
    cy.get('[data-testid="model-description-input"]').type('Created by Cypress');
    cy.get('[data-testid="submit-btn"]').click();

    cy.get('[data-testid="success-alert"]').should('contain', 'Model created');
    cy.url().should('include', '/registered-models/');
  });

  it('filters models by search', () => {
    cy.get('[data-testid="search-input"]').type('test');
    cy.get('[data-testid="model-row"]').each(($row) => {
      cy.wrap($row).should('contain', 'test');
    });
  });
});
```

### Mock Server for E2E

```typescript
// cypress/support/commands.ts
Cypress.Commands.add('mockApi', () => {
  cy.intercept('GET', '/api/v1/model_registry/*/registered_models', {
    fixture: 'registeredModels.json',
  }).as('getModels');

  cy.intercept('POST', '/api/v1/model_registry/*/registered_models', {
    statusCode: 201,
    fixture: 'newModel.json',
  }).as('createModel');
});
```

## Test Scripts

```json
// package.json
{
  "scripts": {
    "test": "run-s test:lint test:type-check test:unit test:cypress-ci",
    "test:lint": "eslint --max-warnings 0 --ext .js,.ts,.jsx,.tsx ./src",
    "test:type-check": "tsc --noEmit",
    "test:unit": "jest --silent",
    "test:unit:watch": "jest --watch",
    "test:unit:coverage": "jest --coverage",
    "cypress:open": "cypress open",
    "cypress:run": "cypress run",
    "cypress:run:mock": "CY_MOCK=1 cypress run"
  }
}
```

## Best Practices

### 1. Test IDs

Use `data-testid` attributes for reliable element selection:

```tsx
<Button data-testid="submit-btn" onClick={handleSubmit}>
  Submit
</Button>
```

### 2. Arrange-Act-Assert

```typescript
it('updates filter when selection changes', async () => {
  // Arrange
  const onFilterChange = jest.fn();
  render(<FilterSelect options={options} onChange={onFilterChange} />);

  // Act
  await userEvent.click(screen.getByRole('button'));
  await userEvent.click(screen.getByText('Option 1'));

  // Assert
  expect(onFilterChange).toHaveBeenCalledWith('option-1');
});
```

### 3. Avoid Implementation Details

```typescript
// Good - tests behavior
it('shows error message on failed submission', async () => {
  render(<Form onSubmit={mockFailingSubmit} />);
  await userEvent.click(screen.getByRole('button', { name: /submit/i }));
  expect(screen.getByRole('alert')).toHaveTextContent('Submission failed');
});

// Bad - tests implementation
it('sets error state to true', async () => {
  // This tests internal state, not behavior
});
```

### 4. Test Accessibility

```typescript
it('has accessible form controls', () => {
  render(<CreateModelForm />);

  expect(screen.getByLabelText('Model Name')).toBeInTheDocument();
  expect(screen.getByRole('textbox', { name: /description/i })).toBeInTheDocument();
});
```

---

[Back to Frontend Index](./README.md) | [Previous: API Integration](./api-integration.md)

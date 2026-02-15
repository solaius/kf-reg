# Testing Coverage

This document analyzes the testing practices and coverage in the Kubeflow Model Registry codebase.

## Testing Overview

### Test Stack

| Component | Framework | Type |
|-----------|-----------|------|
| Go Backend | testing + testify | Unit, Integration |
| Go Backend | Testcontainers | Integration |
| Go Backend | envtest | K8s Integration |
| Frontend | Jest | Unit |
| Frontend | Cypress | E2E |
| Python Client | pytest | Unit, E2E |
| Python Client | Schemathesis | Fuzz |

## Go Backend Testing

### Unit Tests

**Location**: `*_test.go` files alongside source

**Pattern**:
```go
func TestGetRegisteredModel(t *testing.T) {
    // Arrange
    repo := NewMockRepository()
    service := NewService(repo)

    // Act
    model, err := service.GetRegisteredModel("1")

    // Assert
    require.NoError(t, err)
    assert.Equal(t, "test-model", model.Name)
}
```

**Coverage Areas**:
- Core service logic
- Repository operations
- Type conversions
- Validation functions

### Integration Tests

**Location**: `internal/datastore/`, `pkg/inferenceservice-controller/`

**Approach**: Testcontainers for database, envtest for Kubernetes

```go
func TestWithMySQL(t *testing.T) {
    ctx := context.Background()
    container, _ := mysql.Run(ctx, "mysql:8.3")
    defer container.Terminate(ctx)

    // Test with real database
    dsn := container.MustConnectionString(ctx)
    db := connectDB(dsn)
    // ...
}
```

**Coverage Areas**:
- Database migrations
- Query operations
- Connection handling

### Test Organization

```
internal/
├── core/
│   ├── modelregistry_service.go
│   └── modelregistry_service_test.go
├── datastore/
│   ├── connector.go
│   └── connector_test.go
└── db/
    └── service/
        ├── generic_repository.go
        └── generic_repository_test.go
```

### Mock Implementations

**Mocks provided for**:
- Model Registry client
- Catalog client
- Kubernetes client
- HTTP client

```go
// internal/mocks/model_registry_client_mock.go
type ModelRegistryClientMock struct {
    data *StaticDataMock
}

func (m *ModelRegistryClientMock) GetAllRegisteredModels(...) (*models.RegisteredModelList, error) {
    return m.data.RegisteredModels, nil
}
```

## Frontend Testing

### Jest Unit Tests

**Location**: `__tests__/unit/`

**Coverage**:
- Utility functions
- Hook logic
- Component rendering

```typescript
describe('formatDate', () => {
  it('should format date correctly', () => {
    const date = new Date('2024-01-15T10:30:00Z');
    expect(formatDate(date)).toBe('Jan 15, 2024');
  });
});
```

### Cypress E2E Tests

**Location**: `src/__tests__/cypress/`

**Coverage**:
- User flows
- Navigation
- Form submissions
- Error handling

```typescript
describe('Model Registry', () => {
  beforeEach(() => {
    cy.intercept('GET', '/api/v1/model_registry/**/registered_models', {
      fixture: 'registered-models.json'
    });
    cy.visit('/model-registry');
  });

  it('should display registered models', () => {
    cy.get('[data-testid="model-table"]').should('exist');
    cy.contains('test-model').should('be.visible');
  });
});
```

### Test Configuration

```javascript
// cypress.config.js
module.exports = {
  e2e: {
    baseUrl: 'http://localhost:4000',
    supportFile: 'src/__tests__/cypress/support/e2e.ts',
    specPattern: 'src/__tests__/cypress/tests/**/*.cy.ts'
  }
};
```

## BFF Testing

### Unit Tests

**Location**: `clients/ui/bff/internal/api/*_test.go`

**Pattern**: Table-driven tests with httptest

```go
func TestGetAllRegisteredModelsHandler(t *testing.T) {
    tests := []struct {
        name       string
        setup      func(*App)
        wantStatus int
    }{
        {
            name:       "success",
            setup:      setupMockClient,
            wantStatus: http.StatusOK,
        },
        {
            name:       "not found",
            setup:      setupEmptyClient,
            wantStatus: http.StatusNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            app := NewTestApp()
            tt.setup(app)
            // ...
        })
    }
}
```

### Kubernetes Integration Tests

Uses `envtest` for realistic Kubernetes environment:

```go
func SetupEnvTest(input TestEnvInput) (*envtest.Environment, kubernetes.Interface, error) {
    testEnv := &envtest.Environment{
        CRDDirectoryPaths: []string{
            filepath.Join("..", "..", "..", "config", "crd", "bases"),
        },
    }
    cfg, err := testEnv.Start()
    // ...
}
```

## Python Client Testing

### Unit Tests

**Location**: `clients/python/tests/`

```python
def test_register_model():
    registry = ModelRegistry("http://localhost:8080", author="test")
    model = registry.register_model(
        name="test-model",
        uri="s3://bucket/model.onnx",
        version="1.0.0",
        model_format_name="onnx",
        model_format_version="1"
    )
    assert model.name == "test-model"
```

### E2E Tests

**Location**: `clients/python/tests/e2e/`

Requires running Model Registry server.

### Fuzz Testing

Uses Schemathesis for property-based API testing:

```python
# Generates random valid inputs based on OpenAPI schema
schemathesis run --base-url http://localhost:8080 api/openapi/model-registry.yaml
```

## Test Coverage Analysis

### Estimated Coverage by Component

| Component | Unit | Integration | E2E |
|-----------|------|-------------|-----|
| Core Backend | 70% | 50% | - |
| Repository Layer | 60% | 70% | - |
| API Handlers | 50% | 30% | - |
| Frontend | 40% | - | 30% |
| BFF | 50% | 40% | - |
| Python Client | 60% | - | 30% |
| MCP Catalog | 40% | 20% | 10% |

### Coverage Gaps

1. **Error Path Testing**
   - Happy paths well covered
   - Error scenarios need more attention

2. **Integration Scenarios**
   - Cross-component integration limited
   - Full workflow tests sparse

3. **MCP Catalog**
   - Newer feature with less coverage
   - Provider integration tests needed

4. **Edge Cases**
   - Pagination boundaries
   - Large data sets
   - Concurrent operations

## Test Quality Observations

### Strengths

1. **Mocking Strategy**: Comprehensive mocks enable isolated testing
2. **Table-Driven Tests**: Go tests follow best practices
3. **E2E Coverage**: Cypress tests cover critical user flows
4. **Fuzz Testing**: Property-based testing for API robustness

### Weaknesses

1. **Test Isolation**: Some tests depend on execution order
2. **Flaky Tests**: Some Cypress tests have timing issues
3. **Coverage Metrics**: No automated coverage reporting
4. **Documentation**: Test documentation sparse

## Test Infrastructure

### CI Pipeline

Tests run on:
- Pull requests
- Main branch commits
- Scheduled runs

### Test Environments

| Environment | Purpose |
|-------------|---------|
| Local | Developer testing |
| CI (GitHub Actions) | Automated validation |
| Kind cluster | Integration testing |

### Test Data

**Fixtures**:
- JSON files for API responses
- YAML files for configurations
- SQL scripts for database state

**Factories**:
- Builder patterns for test objects
- Random data generation

## Recommendations

### High Priority

1. **Add Coverage Reporting**
   - Configure go test coverage
   - Add Jest coverage
   - Track trends over time

2. **MCP Catalog Testing**
   - Add provider unit tests
   - Integration tests with YAML sources
   - E2E tests for MCP pages

3. **Error Path Testing**
   - Test all error conditions
   - Verify error messages
   - Test recovery scenarios

### Medium Priority

4. **Integration Test Suite**
   - Full workflow tests
   - Cross-service scenarios
   - Performance benchmarks

5. **Test Documentation**
   - Document test patterns
   - Testing guidelines
   - Fixture documentation

6. **Flaky Test Resolution**
   - Identify flaky tests
   - Add retries where appropriate
   - Fix root causes

### Low Priority

7. **Property-Based Testing**
   - Expand Schemathesis usage
   - Add Go property tests
   - Frontend property tests

8. **Visual Regression**
   - Screenshot comparison
   - Component visual tests

## Test Commands Reference

```bash
# Go tests
make test
go test ./... -v
go test ./... -cover

# Frontend tests
npm run test           # Jest
npm run test:cypress   # Cypress interactive
npm run test:cypress-ci # Cypress headless

# BFF tests
cd clients/ui/bff
make test

# Python tests
cd clients/python
poetry run pytest
poetry run pytest tests/e2e/
```

## Conclusion

Testing in the Kubeflow Model Registry is functional but has room for improvement:

- **Strengths**: Good mocking, table-driven Go tests, E2E coverage
- **Gaps**: Error path testing, MCP coverage, integration tests

Recommended focus: Improve coverage reporting and expand MCP Catalog tests.

---

[Back to Code Review Index](./README.md) | [Previous: Security Analysis](./security-analysis.md)

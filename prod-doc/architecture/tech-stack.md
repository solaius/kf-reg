# Technology Stack

This document provides a comprehensive overview of all technologies, frameworks, and tools used in the Kubeflow Model Registry project.

## Backend Technologies

### Primary Language: Go

| Component | Version | Purpose |
|-----------|---------|---------|
| Go | 1.24.6+ | Primary backend language |
| Go Modules | 1.24 | Dependency management |
| Go Workspaces | - | Multi-module monorepo support |

### Web Framework & Routing

| Library | Version | Purpose |
|---------|---------|---------|
| go-chi/chi | v5 | HTTP router for REST APIs |
| go-chi/cors | v1 | CORS middleware |
| net/http | stdlib | HTTP server |

### Database & ORM

| Library | Version | Purpose |
|---------|---------|---------|
| gorm.io/gorm | latest | ORM for database operations |
| gorm.io/driver/mysql | latest | MySQL driver |
| gorm.io/driver/postgres | latest | PostgreSQL driver |
| golang-migrate/migrate | v4 | Database migrations |
| gorm.io/gen | latest | GORM struct generation |

**Supported Databases:**
- MySQL 8.3+
- PostgreSQL (latest stable)

### CLI & Configuration

| Library | Version | Purpose |
|---------|---------|---------|
| spf13/cobra | latest | CLI framework |
| spf13/viper | latest | Configuration management |

### Code Generation

| Tool | Purpose |
|------|---------|
| openapi-generator-cli | OpenAPI server/client generation |
| goverter | Type converter generation |
| gorm-gen | Database struct generation |

### Logging

| Library | Purpose |
|---------|---------|
| golang/glog | Structured logging |
| uber-go/zap | High-performance logging (catalog) |

### Testing

| Library | Purpose |
|---------|---------|
| onsi/ginkgo | BDD testing framework |
| onsi/gomega | Matcher library |
| testcontainers-go | Container-based integration tests |

### Kubernetes Integration

| Library | Purpose |
|---------|---------|
| controller-runtime | Kubernetes controller framework |
| client-go | Kubernetes API client |
| sigs.k8s.io/kind | Local Kubernetes for testing |

---

## Frontend Technologies

### Primary Stack

| Technology | Version | Purpose |
|------------|---------|---------|
| React | ^18 | UI framework |
| TypeScript | ^5.8.2 | Type-safe JavaScript |
| Webpack | 5 | Module bundler |

### UI Component Libraries

| Library | Version | Purpose |
|---------|---------|---------|
| @patternfly/react-core | ^6.4.0 | Enterprise UI components |
| @patternfly/react-table | ^6.4.0 | Table components |
| @mui/material | ^7.3.4 | Material UI components |
| @mui/icons-material | ^7.3.4 | Material icons |
| @emotion/react | ^11.14.0 | CSS-in-JS styling |
| @emotion/styled | ^11.14.0 | Styled components |

### Routing & State

| Library | Version | Purpose |
|---------|---------|---------|
| react-router-dom | ^7 | Client-side routing |
| React Context API | built-in | State management |

### Build Tools

| Tool | Purpose |
|------|---------|
| Webpack 5 | Module bundling and optimization |
| SWC | Fast TypeScript/JavaScript compilation |
| Babel | JavaScript transpilation |
| PostCSS | CSS processing |
| Sass | CSS preprocessor |

### Testing

| Tool | Version | Purpose |
|------|---------|---------|
| Jest | ^29.7.0 | Unit testing framework |
| Cypress | ^14.4.1 | E2E testing framework |
| @testing-library/react | latest | React component testing |
| cypress-axe | ^1.5.0 | Accessibility testing |

### Code Quality

| Tool | Purpose |
|------|---------|
| ESLint | JavaScript/TypeScript linting |
| Prettier | Code formatting |
| TypeScript | Static type checking |

---

## Python Client

### Stack

| Technology | Version | Purpose |
|------------|---------|---------|
| Python | 3.9+ | Client language |
| Poetry | latest | Dependency management |
| OpenAPI Generator | - | Client code generation |

### Testing & Quality

| Tool | Purpose |
|------|---------|
| pytest | Testing framework |
| ruff | Linting |
| mypy | Type checking |
| nox | Test automation |

---

## Infrastructure & DevOps

### Containerization

| Technology | Purpose |
|------------|---------|
| Docker | Container runtime |
| Docker Compose | Multi-container orchestration |
| Podman | Alternative container runtime |

### Kubernetes

| Technology | Purpose |
|------------|---------|
| Kubernetes | Container orchestration |
| Kustomize | Kubernetes manifest management |
| Kind | Local Kubernetes clusters |
| Helm | Package management (optional) |

### CI/CD

| Technology | Purpose |
|------------|---------|
| GitHub Actions | CI/CD pipelines |
| Pre-commit | Git hooks |
| DCO | Developer Certificate of Origin |

### Security & Compliance

| Tool | Purpose |
|------|---------|
| FOSSA | License scanning |
| Trivy | Container vulnerability scanning |
| OpenSSF Scorecard | Security best practices |
| Dependabot | Dependency updates |

---

## API Specifications

### OpenAPI

| Specification | Location | Purpose |
|---------------|----------|---------|
| Model Registry API | `api/openapi/model-registry.yaml` | Core REST API |
| Catalog API | `api/openapi/catalog.yaml` | Catalog service API |
| UI BFF API | `clients/ui/api/openapi/mod-arch.yaml` | BFF API |

**OpenAPI Version:** 3.0.3

### GraphQL

| Component | Purpose |
|-----------|---------|
| genqlient | GraphQL client generation |
| Red Hat Catalog Schema | External catalog integration |

---

## Development Environment

### Required Tools

| Tool | Minimum Version | Purpose |
|------|-----------------|---------|
| Go | 1.25+ | Backend development |
| Node.js | 20.0.0+ | Frontend development |
| npm | 10.2.0+ | Package management |
| Java | 11.0+ | OpenAPI generation |
| Python | 3.9+ | Python client development |
| Docker | latest | Container builds |
| Make | 4.0+ | Build automation |

### Optional Tools

| Tool | Purpose |
|------|---------|
| Colima | Docker alternative for macOS |
| DevContainers | Consistent dev environments |
| Kind | Local Kubernetes |
| Tilt | Development workflow |

---

## Version Compatibility Matrix

| Component | Compatible With |
|-----------|-----------------|
| Go 1.24.6 | MySQL 8.3+, PostgreSQL 13+ |
| React 18 | Node.js 18+, TypeScript 5+ |
| Kubernetes 1.25+ | controller-runtime 0.15+ |
| Python 3.9+ | Poetry 1.5+ |

---

## Package Registry Locations

| Type | Registry |
|------|----------|
| Go Modules | proxy.golang.org |
| npm Packages | registry.npmjs.org |
| Python Packages | pypi.org |
| Container Images | quay.io/kubeflow |

---

[Back to Architecture Index](./README.md) | [Previous: Overview](./overview.md) | [Next: Data Models](./data-models.md)

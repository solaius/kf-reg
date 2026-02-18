# Kubeflow Model Registry Documentation

Comprehensive documentation for the Kubeflow Model Registry project.

## Overview

The Kubeflow Model Registry is a metadata management system for machine learning models. It provides:

- **Model Registration**: Track and version ML models
- **Artifact Management**: Manage model files and artifacts
- **Model Catalog**: Curated model discovery from various sources
- **MCP Catalog**: Model Context Protocol server registry
- **Serving Integration**: Deploy models via KServe
- **Kubernetes Native**: Full Kubernetes RBAC integration

## Documentation Structure

```
prod-doc/
├── README.md                      # This file
│
├── architecture/                  # System architecture
│   ├── README.md
│   ├── overview.md               # High-level architecture
│   ├── tech-stack.md             # Technology stack
│   ├── data-models.md            # Entity relationships
│   ├── api-design.md             # REST API patterns
│   └── deployment-modes.md       # Deployment options
│
├── backend/                       # Core backend service
│   ├── README.md
│   ├── core-service.md           # ModelRegistryService
│   ├── repository-pattern.md     # Generic repository
│   ├── datastore-abstraction.md  # Pluggable datastores
│   ├── database-layer.md         # GORM and migrations
│   ├── converter-mapper.md       # Type conversion
│   ├── middleware.md             # Validation and routing
│   └── configuration.md          # Cobra/Viper config
│
├── catalog-service/               # Model Catalog service
│   ├── README.md
│   ├── architecture.md           # Catalog architecture
│   ├── source-providers.md       # YAML, HuggingFace
│   ├── filtering-system.md       # Query filtering
│   ├── database-models.md        # Catalog DB models
│   └── performance-metrics.md    # Performance artifacts
│
├── frontend/                      # React UI
│   ├── README.md
│   ├── architecture.md           # React app structure
│   ├── state-management.md       # Context API patterns
│   ├── component-library.md      # PatternFly, MUI
│   ├── routing.md                # React Router
│   ├── api-integration.md        # REST client patterns
│   └── testing.md                # Jest, Cypress
│
├── bff/                           # Backend for Frontend
│   ├── README.md
│   ├── architecture.md           # BFF layer design
│   ├── handlers.md               # API handlers
│   ├── repositories.md           # Data access layer
│   └── kubernetes-integration.md # K8s client
│
├── clients/                       # Client libraries
│   ├── README.md
│   └── python-client.md          # Python SDK
│
├── kubernetes/                    # K8s components
│   ├── README.md
│   ├── controller.md             # InferenceService controller
│   ├── csi-driver.md             # Storage initializer
│   └── deployment-manifests.md   # Kustomize configs
│
├── guides/                        # Developer guides
│   ├── README.md
│   ├── developer-guide.md        # Development setup
│   ├── contributor-requirements.md # DCO, PR workflow
│   ├── style-guide.md            # Code style (Go, TS)
│   └── ui-design-requirements.md # UI/UX patterns
│
├── mcp-catalog/                   # MCP Catalog feature
│   ├── README.md
│   ├── implementation-overview.md # What was built
│   ├── files-changed.md          # File inventory
│   ├── architecture.md           # MCP architecture
│   ├── configuration-guide.md    # Configuration
│   ├── data-models.md            # McpServer, McpTool
│   ├── remaining-work.md         # Work left to do
│   └── step-by-step-creation.md  # Implementation guide
│
├── code-review/                   # Code review analysis
│   ├── README.md
│   ├── executive-summary.md      # High-level findings
│   ├── issues-by-priority.md     # Detailed issues
│   ├── architecture-observations.md # Design patterns
│   ├── security-analysis.md      # Security review
│   └── testing-coverage.md       # Test coverage
│
├── extensibility/                 # Extending the registry
│   ├── README.md
│   ├── asset-type-framework.md   # Framework overview
│   ├── adding-new-assets.md      # Step-by-step guide
│   └── proposed-assets.md        # Future asset types
│
└── catalog_of_catalogs/          # Catalog-of-Catalogs Platform
    ├── README.md
    ├── plugin-framework/         # Plugin architecture and lifecycle
    │   ├── README.md
    │   ├── architecture.md       # Core interfaces, registry, server
    │   ├── creating-plugins.md   # Plugin creation guide
    │   └── configuration.md      # sources.yaml, env vars, flags
    ├── source-management/        # Source CRUD, persistence, validation
    │   ├── README.md
    │   ├── config-stores.md      # File and K8s config stores
    │   ├── validation-pipeline.md # Multi-layer validation
    │   └── refresh-and-diagnostics.md # Refresh, rate limits
    ├── universal-assets/         # Universal asset framework
    │   ├── README.md
    │   ├── capabilities-discovery.md # V2 capabilities schema
    │   ├── asset-contract.md     # AssetResource, overlay store
    │   └── action-framework.md   # Actions, dry-run, builtins
    ├── plugins/                  # Concrete plugin documentation
    │   ├── README.md
    │   ├── model-and-mcp-plugins.md # Model + MCP plugins
    │   └── asset-type-plugins.md # Knowledge, Prompts, Agents, etc.
    ├── operations/               # Deployment and security
    │   ├── README.md
    │   ├── deployment.md         # Docker, K8s, health probes
    │   └── security.md           # RBAC, JWT, SecretRef
    └── clients/                  # Client surfaces
        ├── README.md
        ├── bff-integration.md    # BFF proxy handlers
        ├── generic-ui.md         # Generic React components
        └── catalogctl-and-conformance.md # CLI + test suite
```

## Quick Navigation

### Getting Started

| Need | Go To |
|------|-------|
| Understand the system | [Architecture Overview](./architecture/overview.md) |
| Set up development | [Developer Guide](./guides/developer-guide.md) |
| Contribute code | [Contributor Requirements](./guides/contributor-requirements.md) |
| Deploy the registry | [Deployment Manifests](./kubernetes/deployment-manifests.md) |

### Component Documentation

| Component | Description | Link |
|-----------|-------------|------|
| **Backend** | Core Model Registry service | [Backend Docs](./backend/README.md) |
| **Catalog** | Model and MCP catalog service | [Catalog Docs](./catalog-service/README.md) |
| **Frontend** | React web UI | [Frontend Docs](./frontend/README.md) |
| **BFF** | Backend for Frontend | [BFF Docs](./bff/README.md) |
| **Python Client** | Python SDK | [Client Docs](./clients/python-client.md) |
| **K8s Controller** | InferenceService sync | [Controller Docs](./kubernetes/controller.md) |

### Feature Documentation

| Feature | Description | Link |
|---------|-------------|------|
| **MCP Catalog** | Model Context Protocol server registry | [MCP Docs](./mcp-catalog/README.md) |
| **Extensibility** | Adding new asset types | [Extensibility Docs](./extensibility/README.md) |
| **Catalog of Catalogs** | Multi-plugin catalog platform (8 asset types) | [Catalog of Catalogs](./catalog_of_catalogs/README.md) |

### Quality & Review

| Topic | Description | Link |
|-------|-------------|------|
| Code Review | Analysis and findings | [Code Review](./code-review/README.md) |
| Style Guide | Code style standards | [Style Guide](./guides/style-guide.md) |
| UI Design | UI/UX patterns | [UI Design](./guides/ui-design-requirements.md) |

## Technology Stack

| Layer | Technologies |
|-------|--------------|
| **Backend** | Go 1.24+, GORM, Cobra, Viper |
| **Database** | MySQL 8.3+, PostgreSQL 16+ |
| **Frontend** | React 18, TypeScript 5.8, PatternFly 6.4 |
| **API** | OpenAPI 3.0, REST |
| **Kubernetes** | client-go, controller-runtime |
| **Build** | Make, Docker, Kustomize |

## API Versions

| API | Version | Status |
|-----|---------|--------|
| Model Registry | v1alpha3 | Current |
| Model Catalog | v1alpha1 | Current |
| MCP Catalog | v1alpha1 | Feature Branch |
| Catalog Server | v1alpha1 | Feature Branch |

## Repository Links

- **Main Repository**: [github.com/kubeflow/model-registry](https://github.com/kubeflow/model-registry)
- **MCP Feature Branch**: `feature/mcp-catalog`
- **Documentation Site**: [kubeflow.org/docs](https://www.kubeflow.org/docs/components/model-registry/)

## Document Conventions

### Status Indicators

| Indicator | Meaning |
|-----------|---------|
| Production | Stable, in main branch |
| Feature Branch | In development branch |
| Proposed | Not yet implemented |

### Code Examples

Code examples use the following conventions:

- **Go**: Standard Go formatting with golangci-lint
- **TypeScript**: ESLint + Prettier formatting
- **YAML**: 2-space indentation
- **SQL**: Uppercase keywords

### Diagrams

Architecture diagrams use ASCII art for maximum compatibility:

```
┌─────────────┐    ┌─────────────┐
│  Component  │───▶│  Component  │
└─────────────┘    └─────────────┘
```

## Contributing to Documentation

To contribute to this documentation:

1. Fork the repository
2. Create a branch from `main`
3. Make your changes in `prod-doc/`
4. Follow the existing structure and style
5. Submit a pull request with DCO sign-off

See [Contributor Requirements](./guides/contributor-requirements.md) for details.

## Document Maintenance

This documentation was generated from analysis of the codebase and should be updated when:

- New features are added
- Architecture changes
- APIs are modified
- Significant refactoring occurs

Last updated: February 2026

---

**Kubeflow Model Registry** | [GitHub](https://github.com/kubeflow/model-registry) | [Kubeflow](https://kubeflow.org)

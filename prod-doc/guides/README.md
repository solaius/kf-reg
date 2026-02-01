# Guides Documentation

This section provides practical guides for developers and contributors to the Kubeflow Model Registry project.

## Overview

These guides cover everything from setting up your development environment to understanding code style requirements and UI design patterns.

## Documentation

| Document | Description |
|----------|-------------|
| [Developer Guide](./developer-guide.md) | Complete development environment setup and workflows |
| [Contributor Requirements](./contributor-requirements.md) | DCO, PR workflow, and contribution standards |
| [Style Guide](./style-guide.md) | Code style standards for Go and TypeScript |
| [UI Design Requirements](./ui-design-requirements.md) | UI/UX patterns and component guidelines |

## Quick Reference

### Prerequisites

| Component | Requirement |
|-----------|-------------|
| Go | >= 1.24.6 |
| Node.js | >= 20.17.0 |
| NPM | >= 10.8.2 |
| Python | >= 3.9 |
| Docker/Podman | Latest |
| kubectl | Latest |
| kind | Latest (for local K8s) |

### Quick Start

```bash
# Clone the repository
git clone https://github.com/kubeflow/model-registry
cd model-registry

# Backend development
make build
make run

# Frontend development
cd clients/ui/frontend
npm install
npm run start:dev

# BFF development
cd clients/ui/bff
make build
make run MOCK_K8S_CLIENT=true MOCK_MR_CLIENT=true
```

### Key Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build the Model Registry server |
| `make test` | Run tests |
| `make lint` | Run linters |
| `make gen/openapi` | Generate OpenAPI client |
| `make gen/gorm` | Generate GORM models |
| `make docker-build` | Build Docker image |

---

[Back to Main Index](../README.md)

# Developer Guide

This document provides comprehensive instructions for setting up and working with the Kubeflow Model Registry development environment.

## Prerequisites

### Required Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Go | >= 1.24.6 | Backend development |
| Node.js | >= 20.17.0 | Frontend development |
| NPM | >= 10.8.2 | Frontend package management |
| Python | >= 3.9 | Python client development |
| Docker/Podman | Latest | Container builds |
| kubectl | Latest | Kubernetes CLI |
| kind | Latest | Local Kubernetes clusters |
| make | >= 4.0 | Build automation |

### Optional Tools

| Tool | Purpose |
|------|---------|
| Colima | Docker alternative for macOS |
| DevContainer | Containerized development |
| golangci-lint | Go linting |

## Repository Structure

```
model-registry/
├── api/                    # OpenAPI specifications
├── cmd/                    # Entry points (main, csi, controller)
├── internal/               # Internal packages
├── pkg/                    # Public packages
├── catalog/                # Catalog service
├── clients/
│   ├── python/            # Python client
│   └── ui/                # React frontend + BFF
├── manifests/             # Kubernetes manifests
├── scripts/               # Build and utility scripts
└── test/                  # Integration tests
```

## Getting Started

### Clone the Repository

```bash
git clone https://github.com/kubeflow/model-registry
cd model-registry
```

### Backend Development

#### Build the Server

```bash
make build
```

This creates the `model-registry` binary in the project root.

#### Run Locally

```bash
# Start with embedded database
./model-registry proxy --hostname=0.0.0.0 --port=8080 --datastore-type=embedmd

# Or use make
make run
```

#### Run with MySQL

```bash
# Start MySQL container
make start/mysql

# Run with MySQL connection
./model-registry proxy --hostname=0.0.0.0 --port=8080 \
  --datastore-type=mysql \
  --db-host=localhost \
  --db-port=3306 \
  --db-name=model-registry \
  --db-username=root \
  --db-password=root

# Stop MySQL when done
make stop/mysql
```

#### Run with PostgreSQL

```bash
# Start PostgreSQL container
make start/postgres

# Run with PostgreSQL connection
./model-registry proxy --hostname=0.0.0.0 --port=8080 \
  --datastore-type=postgres \
  --db-host=localhost \
  --db-port=5432 \
  --db-name=model-registry \
  --db-username=postgres \
  --db-password=postgres

# Stop PostgreSQL when done
make stop/postgres
```

#### Docker Compose

```bash
# Use pre-built image
docker compose -f docker-compose.yaml up

# Build from source
docker compose -f docker-compose-local.yaml up
```

### Frontend Development

#### Install Dependencies

```bash
cd clients/ui/frontend
npm install
```

#### Development Server

```bash
npm run start:dev
```

This starts the development server with hot reloading.

#### Build Production

```bash
npm run build
```

### BFF Development

#### Build

```bash
cd clients/ui/bff
make build
```

#### Run with Mocks

```bash
make run MOCK_K8S_CLIENT=true MOCK_MR_CLIENT=true
```

#### Run with Real Services

```bash
# Ensure Model Registry is running
make run PORT=4000
```

### Python Client Development

#### Setup

```bash
cd clients/python
pip install poetry
poetry install
```

#### Run Tests

```bash
poetry run pytest
```

## Code Generation

### OpenAPI Generation

```bash
# Validate OpenAPI specs
make openapi/validate

# Generate server stubs
make gen/openapi-server

# Generate Go client
make gen/openapi
```

### GORM Model Generation

```bash
# Generate from MySQL
make gen/gorm GORM_DB_TYPE=mysql

# Generate from PostgreSQL
make gen/gorm GORM_DB_TYPE=postgres
```

### Type Converter Generation

```bash
make gen/converter
```

## Testing

### Backend Tests

```bash
# Run all tests
make test

# Run specific package
go test ./internal/core/...

# Run with verbose output
go test -v ./...
```

### Frontend Tests

```bash
cd clients/ui/frontend

# Unit tests
npm run test

# Cypress E2E tests
npm run test:cypress-ci
```

### BFF Tests

```bash
cd clients/ui/bff
make test
```

### Python Client Tests

```bash
cd clients/python
poetry run pytest

# E2E tests (requires running server)
poetry run pytest tests/e2e/
```

## Local Kubernetes Development

### Create Kind Cluster

```bash
kind create cluster
```

### Deploy Model Registry

```bash
# With MySQL
kubectl apply -k manifests/kustomize/overlays/db -n kubeflow

# Port forward
kubectl port-forward svc/model-registry-service 8080:8080 -n kubeflow
```

### Deploy with Ingress

See [docs/mr_kind_deploy_ingress.md](https://github.com/kubeflow/model-registry/blob/main/docs/mr_kind_deploy_ingress.md) for detailed instructions.

### UI Development with Kind

```bash
cd clients/ui
make kind-deployment
```

## Docker Builds

### Server Image

```bash
make docker-build
make docker-push
```

### UI Image

```bash
make IMG_REPO=model-registry/ui docker-build
```

### CSI Image

```bash
make IMG_REPO=model-registry/storage-initializer docker-build
```

### Controller Image

```bash
make IMG_REPO=model-registry/controller docker-build
```

## macOS/ARM Development

### Install GNU Make

```bash
brew install make
# Use 'gmake' instead of 'make'
```

### Install Coreutils

```bash
brew install coreutils
# Add gnubin to PATH
```

### Docker Options

**Podman with Rosetta:**
```bash
# Setup Podman machine with Rosetta
podman machine init --rootful
podman machine start

# Set environment for Testcontainers
export TESTCONTAINERS_RYUK_PRIVILEGED=true
```

**Colima:**
```bash
# Start with x86 emulation
colima start --vz-rosetta --vm-type vz --arch x86_64 --cpu 4 --memory 8

# Configure for Testcontainers
export DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock"
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE="/var/run/docker.sock"
```

### DevContainer for Python Client

The Python client requires x86 for MLMD dependency. Use DevContainer:

```jsonc
// .devcontainer/devcontainer.json
{
  "name": "Python 3",
  "build": {
    "dockerfile": "Dockerfile"
  },
  "runArgs": ["--network=host"]
}
```

## Debugging

### Backend Debugging

```bash
# Run with debug logging
./model-registry proxy --log-level=debug

# With Delve debugger
dlv debug ./cmd/main.go -- proxy
```

### Frontend Debugging

Use browser DevTools with React Developer Tools extension.

### BFF Debugging

```bash
make run LOG_LEVEL=DEBUG
```

## Common Issues

### OpenAPI Generator Not Found

```
make: openapi-generator-cli: No such file or directory
```

**Solution:** Run `make bin/openapi-generator-cli` or ensure `bin/` is in PATH.

### Database Migration Error

```
Dirty database version X. Fix and force version.
```

**Solution:**
```sql
UPDATE schema_migrations SET dirty = 0;
```

### Testcontainers Ryuk Error

**Solution:** Set `TESTCONTAINERS_RYUK_PRIVILEGED=true`

### Frontend Build Failures

**Solution:** Ensure Node.js version matches requirements:
```bash
node --version  # Should be >= 20.17.0
```

## IDE Configuration

### VS Code

Recommended extensions:
- Go
- ESLint
- Prettier
- TypeScript
- GitLens

### GoLand/IntelliJ

- Enable Go Modules
- Configure GOPATH
- Install File Watchers for auto-formatting

---

[Back to Guides Index](./README.md) | [Next: Contributor Requirements](./contributor-requirements.md)

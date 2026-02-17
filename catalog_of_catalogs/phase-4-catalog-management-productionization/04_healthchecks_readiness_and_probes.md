# 04_healthchecks_readiness_and_probes

**Date**: 2026-02-16  
**Owner**: catalog-server plus docker-compose plus deployment manifests  
**Goal**: Replace placeholder health checks with real HTTP endpoints and meaningful readiness semantics

## Problem statement

Docker compose currently uses a binary runs with --help style health check. This does not indicate whether:
- the HTTP server is accepting traffic
- DB connectivity is working
- plugins initialized
- migrations completed
- initial catalog load completed

We need real HTTP health endpoints and to wire them into Docker and Kubernetes probes.

## Requirements

### R1: Implement HTTP endpoints in catalog-server

Implement:
- GET /livez (or keep /healthz alias)
- GET /readyz

Semantics:
- /livez:
  - 200 if process is alive and HTTP server loop is functioning
  - should not fail due to transient downstream dependencies
- /readyz:
  - 200 only when the service is ready to serve API traffic
  - should check at least:
    - DB connectivity (ping)
    - plugin init completed
    - migrations completed
    - initial load completed (or explicitly decided that initial load is not required for readiness)

Optional:
- include JSON payload with component statuses when Accept: application/json

### R2: Update Kubernetes probes (deployment manifests)

Add:
- startupProbe (if initial load plus migrations can take time)
- livenessProbe using /livez
- readinessProbe using /readyz

These are HTTP GET probes, so the image does not need curl or a shell.

### R3: Update Docker compose healthcheck

Docker healthchecks run inside the container. If the runtime image is distroless, curl or wget may not exist.

Best path:
- Add a tiny static healthcheck binary in the image that does an HTTP GET to localhost and returns 0 or 1
- Use it in compose:
  - test: ["CMD", "/usr/local/bin/healthcheck", "http://localhost:8080/readyz"]

### R4: Document the probe contract for BFF (optional)

If the BFF is deployed separately, it should also expose /livez and /readyz
- /readyz for BFF should include can reach catalog-server

## Acceptance criteria

- Docker compose shows catalog-server as healthy only when HTTP is up and readyz passes
- Kubernetes readiness flips to ready only when DB and plugin system are ready
- Liveness and readiness semantics match standard expectations
- No new dependencies are added to distroless runtime beyond the small healthcheck helper binary

## Definition of Done

- Endpoints implemented with tests
- Docker compose healthcheck updated and verified
- Deployment probe config added (or documented) and verified in a cluster environment

## References

- Kubernetes probe configuration guide  
  https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/  
- Docker compose healthcheck patterns  
  https://last9.io/blog/docker-compose-health-checks/

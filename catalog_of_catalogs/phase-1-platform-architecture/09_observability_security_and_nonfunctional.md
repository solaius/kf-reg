# Observability, security, and non-functional requirements

## Observability

### Logs
- Structured logs
- Include pluginName, sourceId, and operation name for ingestion events
- Include request identifiers for API calls when available

### Metrics
At minimum, emit metrics per plugin and per source:
- ingestion_last_attempt_timestamp
- ingestion_last_success_timestamp
- ingestion_duration_seconds
- ingestion_error_total
- entities_ingested_total
- artifacts_ingested_total
- api_requests_total (optional but recommended)
- api_request_duration_seconds (optional but recommended)

### Health and readiness
Health should answer: is the server process alive
Readiness should answer: can the server serve requests correctly

Readiness must consider:
- DB connectivity
- route registration for enabled plugins
- plugin initialization status
- optional: at least one successful refresh per enabled plugin, if configured as required

## Reliability

### Fault isolation
- A plugin that fails configuration validation should not take down other plugins
- A single failing source must not prevent other sources from refreshing
- Plugin health should surface errors without hiding them

### Migrations
- Migrations must be idempotent
- Migrations should run exactly once per plugin per server start
- Failure in migrations should mark the plugin unhealthy and not register routes

## Performance
- Ingestion should support multiple sources concurrently
- List endpoints should be performant for typical UI usage
- Define indexes for common query fields
- Avoid expensive joins for default list views

## Security
- Integrate with existing authn and authz patterns in the platform
- Avoid encouraging plaintext secrets in sources.yaml
- Prefer secret references consistent with repository conventions

## Maintainability
- Generated code must be deterministic and validated in CI
- Keep plugin surface area small and well documented
- Favor shared framework utilities over copy-paste

## Backward compatibility
- Model plugin behavior is a protected contract
- Shared schemas can only evolve additively


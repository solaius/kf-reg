# Observability and Operations

## What ops need to know at a glance

- Is the catalog server healthy
- Are plugins healthy
- Are sources refreshing successfully
- When was the last successful refresh
- What failed and what to do next

## Required signals

Health

- /healthz for liveness
- /readyz for readiness
- Per plugin health surfaced via a status endpoint

Metrics

- Refresh duration per plugin and source
- Refresh success or failure counters
- Entity counts per plugin and source
- Error counts by provider type

Logs

- Structured logs with
  - plugin name
  - source id
  - provider type
  - correlation id per refresh run

Tracing (optional but recommended)

- Trace refresh requests through loader and providers

Diagnostics UX

- UI shows
  - last refresh times
  - last error message and hint
  - recommended next steps

## Failure modes to handle well

- Bad YAML format
- Missing file path or unreadable file
- HTTP source unreachable
- Schema mismatch between source data and entity schema
- Database migration failure

## Acceptance Criteria

- For each failure mode, the UI and CLI show an actionable message
- Health and readiness reflect failures appropriately without flapping

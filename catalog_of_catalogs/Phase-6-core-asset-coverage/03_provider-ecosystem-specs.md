# Provider ecosystem specifications

## Goals
- Make it easy for any plugin to load assets from:
  - Local YAML (baseline)
  - Remote HTTP catalogs
  - Git repositories (catalog-as-code)
  - OCI registries (assets-as-artifacts)
- Keep provider behaviors consistent:
  - Same error model
  - Same filtering and pagination expectations
  - Same artifact integrity checks

## Provider contract (applies to all provider types)
### Required behaviors
- Enumerate assets for a source
- Provide stable asset IDs (within a source)
- Provide artifact references and digests when artifacts are external
- Return provider diagnostics:
  - last successful sync time
  - last attempted sync time
  - last error (if any)
  - item counts (assets, artifacts)
- Respect include and exclude patterns when applicable
- Respect per-source enable/disable state

### Error taxonomy
- ConfigError: invalid source config (no retries)
- AuthError: missing/invalid credentials (no retries until fixed)
- NotFound: remote resource missing (no retries until changed)
- RateLimited: provider should backoff and retry
- TransientNetwork: retry with exponential backoff
- ValidationError: input data fails schema validation (no retries until source changes)
- IntegrityError: digest mismatch for external artifacts (no retries until source changes)

### Caching and idempotency
- Providers must be safe to re-run
- Sync operations must be idempotent
- Partial results must not corrupt DB state
- If a provider supports incremental sync, it must maintain a checkpoint and be able to fall back to full sync

## YAML provider (baseline)
### Source config
- yamlCatalogPath (local file path)
- watch (bool)
- pollingIntervalSeconds (optional)
- includeGlobs, excludeGlobs (optional)
- allowUnknownFields (default false)
- strictSchemaValidation (default true)

### Responsibilities
- Validate YAML shape against plugin schema
- Apply include/exclude filtering before writing to DB
- Support hot reload with file watching where supported

### Acceptance criteria
- Works for every Phase 6 plugin
- Produces consistent diagnostics and clear schema validation errors

## HTTP provider (remote catalogs)
### Source config
- baseUrl (required)
- auth:
  - none
  - bearerToken
  - oauth2ClientCredentials
  - mTLS (optional)
- headers (optional)
- rateLimit:
  - maxRequestsPerSecond
  - burst
- pagination:
  - supportsPageToken (bool)
  - pageSize (int)
- tls:
  - caBundlePath
  - insecureSkipVerify (default false)
- cache:
  - etagSupport (bool)
  - ifModifiedSinceSupport (bool)

### Fetch strategy
- Prefer stable list endpoints with pagination
- Support conditional requests (ETag or If-Modified-Since) when available
- Map remote schema to local plugin schema through explicit translation logic

### Security requirements
- Do not log secrets
- Timeouts and retry policies are mandatory
- Reject insecure TLS by default

### Acceptance criteria
- Can load at least one Phase 6 plugin catalog from a remote endpoint
- Error handling and retries behave predictably and are observable

## Git provider (catalog-as-code)
### Source config
- repoUrl (required)
- ref:
  - branch (default main)
  - tag (optional)
  - commit (optional)
- auth:
  - none
  - sshKeyPath
  - httpsToken
- paths:
  - includePaths
  - excludePaths
  - globPatterns
- sync:
  - pollingIntervalSeconds
  - shallowClone (default true)
- manifest:
  - manifestFile (optional) for multi-plugin catalogs

### Fetch strategy
- Clone or fetch updates on schedule
- Discover catalog files:
  - Default: plugin-specific folder conventions
  - Optional: manifest file listing plugin catalogs
- Apply validation and include/exclude filtering before DB writes

### Integrity and traceability
- Capture commit SHA used for every sync
- Store file path and file hash for each asset entry
- Enable rollback by switching to previous commit and re-syncing

### Acceptance criteria
- Can load at least one Phase 6 plugin catalog from a Git repo
- Commit provenance is visible in UI and CLI

## OCI registry provider (assets as artifacts)
### Source config
- registry (required)
- repository (required)
- auth:
  - anonymous
  - basic
  - bearer token
  - cloud registry token (optional)
- selectors:
  - tagPatterns
  - digestAllowlist
  - artifactTypeAllowlist
- pull:
  - maxConcurrent
  - verifyDigest (default true)
- cache:
  - localCacheDir
  - maxCacheSizeMB

### Artifact conventions
- Each asset is represented by an OCI artifact that includes:
  - artifactType
  - annotations for key metadata
  - a payload layer with a canonical manifest for the asset (JSON or YAML)
- Digests must be verified during fetch

### Acceptance criteria
- Works end-to-end for at least one plugin where artifacts are natural (guardrails, policies, skills)
- UI and CLI show artifact digests and provenance

## Definition of done
- Provider contracts are documented and implemented with tests
- At least two provider types beyond YAML are used end-to-end in Phase 6
- Provider diagnostics are visible in CLI and UI via the existing management surfaces

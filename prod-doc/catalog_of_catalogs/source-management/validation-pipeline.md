# Validation Pipeline

## Overview

The catalog-server enforces a **multi-layer validation pipeline** before any source configuration mutation takes effect. Each layer checks a different aspect of the input -- from basic YAML syntax through plugin-specific semantics -- and the pipeline short-circuits on critical failures so that later layers never run against garbage input.

Validation is invoked in two contexts:

1. **Explicit** -- the operator calls `POST .../management/validate-source` or `POST .../management/sources/{sourceId}:validate` to dry-run validation without applying changes.
2. **Implicit** -- the `apply-source` handler runs the full pipeline internally and rejects the request with HTTP 422 if validation fails.

**Location:** `pkg/catalog/plugin/validator.go`

## MultiLayerValidator

The `MultiLayerValidator` is a pipeline runner. It holds an ordered slice of `ValidationLayer` values and executes them sequentially against a `SourceConfigInput`.

```go
// pkg/catalog/plugin/validator.go

type MultiLayerValidator struct {
    layers []ValidationLayer
}

func (v *MultiLayerValidator) Validate(ctx context.Context, input SourceConfigInput) *DetailedValidationResult {
    result := &DetailedValidationResult{Valid: true}

    for _, layer := range v.layers {
        errs := layer.Check(ctx, input)

        layerResult := LayerValidationResult{
            Layer: layer.Name,
            Valid: len(errs) == 0,
            Errors: errs,
        }
        result.LayerResults = append(result.LayerResults, layerResult)

        if len(errs) > 0 {
            if layer.WarningOnly {
                // Populate Warnings; do NOT affect Valid flag.
                result.Warnings = append(result.Warnings, errs...)
            } else {
                result.Valid = false
                result.Errors = append(result.Errors, errs...)

                if layer.Critical {
                    break   // Stop pipeline on critical failure.
                }
            }
        }
    }
    return result
}
```

Key behaviors:

- Layers run in registration order.
- A layer marked `Critical: true` halts the pipeline on failure.
- A layer marked `WarningOnly: true` populates `Warnings` instead of `Errors` and never sets `Valid = false`.
- Non-critical, non-warning layers append to `Errors` and set `Valid = false` but allow subsequent layers to continue.

## Built-in Layers

`NewDefaultValidator(sm SourceManager)` assembles the standard pipeline with five layers:

| Order | Layer Name | Constructor | Critical | WarningOnly | What It Checks |
|-------|------------|-------------|----------|-------------|----------------|
| 1 | `yaml_parse` | `YAMLParseLayer()` | Yes | No | Extracts `properties.content` and runs `yaml.Unmarshal`. Rejects malformed YAML before any further inspection. |
| 2 | `strict_fields` | `StrictFieldsLayer()` | No | No | Re-encodes the envelope fields (`id`, `name`, `type`, `properties` minus `content`) and strict-decodes into `SourceConfig` to detect unknown or misspelled fields. |
| 3 | `semantic` | `SemanticLayer()` | No | No | Checks required fields (`id`, `name`, `type`), validates ID format against `^[a-z0-9][a-z0-9_-]*$`, enforces name length <= 256 characters. |
| 4 | `security_warnings` | `SecurityWarningsLayer()` | No | Yes | Scans property keys for sensitive patterns (`password`, `token`, `secret`, `apikey`, `api_key`, `credential`) and warns when the value is a plain string instead of a `SecretRef` map. |
| 5 | `provider` | `ProviderLayer(sm)` | No | No | Delegates to the plugin's `SourceManager.ValidateSource()` for provider-specific checks (e.g., URL reachability, schema compatibility). Only added when `sm != nil`. |

```go
func NewDefaultValidator(sm SourceManager) *MultiLayerValidator {
    v := NewMultiLayerValidator()
    v.AddLayer(YAMLParseLayer())
    v.AddLayer(StrictFieldsLayer())
    v.AddLayer(SemanticLayer())
    v.AddLayer(SecurityWarningsLayer())
    if sm != nil {
        v.AddLayer(ProviderLayer(sm))
    }
    return v
}
```

## Layer Execution Flow

```
  SourceConfigInput
        |
        v
+------------------+         +---------+
| 1. yaml_parse    |--fail-->| STOP    |   Critical = true
|    (Critical)    |         | Valid=F  |   Pipeline halts immediately
+------------------+         +---------+
        |
       pass
        |
        v
+------------------+
| 2. strict_fields |--fail-->  Errors += [...], Valid = false
+------------------+           (continue to next layer)
        |
       pass
        |
        v
+------------------+
| 3. semantic      |--fail-->  Errors += [...], Valid = false
+------------------+           (continue to next layer)
        |
       pass
        |
        v
+---------------------+
| 4. security_warnings|--issues-->  Warnings += [...]
|    (WarningOnly)     |            Valid unchanged
+---------------------+
        |
       pass
        |
        v
+------------------+
| 5. provider      |--fail-->  Errors += [...], Valid = false
+------------------+
        |
       pass
        |
        v
  DetailedValidationResult
    Valid: true/false
    Errors: [...]
    Warnings: [...]
    LayerResults: [per-layer breakdown]
```

Summary of short-circuit and flag rules:

| Condition | Pipeline Continues? | Affects `Valid`? |
|-----------|---------------------|------------------|
| Critical layer fails | No -- pipeline halts | Yes, set to `false` |
| Non-critical layer fails | Yes | Yes, set to `false` |
| WarningOnly layer has issues | Yes | No |
| Layer passes | Yes | No |

## DetailedValidationResult

The full result returned from `Validate()` and from the `:validate` endpoint:

```json
{
  "valid": false,
  "errors": [
    { "field": "id", "message": "source ID is required" },
    { "field": "name", "message": "source name must be 256 characters or fewer" }
  ],
  "warnings": [
    {
      "field": "properties.token",
      "message": "property \"token\" appears to contain an inline credential; consider using a SecretRef instead"
    }
  ],
  "layerResults": [
    { "layer": "yaml_parse",         "valid": true  },
    { "layer": "strict_fields",      "valid": true  },
    { "layer": "semantic",           "valid": false, "errors": [{ "field": "id", "message": "..." }] },
    { "layer": "security_warnings",  "valid": false, "errors": [{ "field": "properties.token", "message": "..." }] },
    { "layer": "provider",           "valid": true  }
  ]
}
```

Notes:

- `layerResults[].valid` reflects whether that individual layer found issues; for warning-only layers this can be `false` while the top-level `valid` remains `true`.
- `errors` and `warnings` at the top level are flattened across all layers for convenience.

## API Endpoints

### Dry-run validation

```
POST /api/{plugin}_catalog/{version}/management/validate-source
```

Accepts a `SourceConfigInput` JSON body and returns a `ValidationResult` (simple valid/errors format) by delegating to `SourceManager.ValidateSource()`. This is a lightweight check that runs only the plugin's own validation.

### Detailed multi-layer validation

```
POST /api/{plugin}_catalog/{version}/management/sources/{sourceId}:validate
```

Accepts a `SourceConfigInput` JSON body and returns a `DetailedValidationResult` with per-layer breakdown. Runs `NewDefaultValidator(sm).Validate()` which includes all five built-in layers.

### Implicit validation during apply

```
POST /api/{plugin}_catalog/{version}/management/apply-source
```

Before mutating any state, the apply handler constructs a `NewDefaultValidator(sm)` and calls `Validate()`. If the result has `Valid == false`, the handler returns HTTP **422 Unprocessable Entity** with the `DetailedValidationResult` as the response body. The apply proceeds only when validation passes.

```go
// management_handlers.go -- inside applyHandler
validator := NewDefaultValidator(sm)
valResult := validator.Validate(r.Context(), input)
if !valResult.Valid {
    w.WriteHeader(http.StatusUnprocessableEntity)
    json.NewEncoder(w).Encode(valResult)
    return
}
// ... proceed with SecretRef resolution and sm.ApplySource()
```

All validation endpoints require the **Operator** role (enforced by `RequireRole(RoleOperator, ...)`).

## Extending Validation

Plugins extend the pipeline through the **provider layer**. When a plugin implements `SourceManager`, its `ValidateSource()` method is called as the final layer:

```go
// SourceManager interface (pkg/catalog/plugin/plugin.go)
type SourceManager interface {
    ListSources(ctx context.Context) ([]SourceStatus, error)
    ValidateSource(ctx context.Context, input SourceConfigInput) (*ValidationResult, error)
    ApplySource(ctx context.Context, input SourceConfigInput) error
    EnableSource(ctx context.Context, id string, enabled bool) error
    DeleteSource(ctx context.Context, id string) error
}
```

The `ProviderLayer` adapter calls `sm.ValidateSource()` and converts the returned `ValidationResult` errors into the pipeline's `[]ValidationError` format:

```go
func ProviderLayer(sm SourceManager) ValidationLayer {
    return ValidationLayer{
        Name: "provider",
        Check: func(ctx context.Context, input SourceConfigInput) []ValidationError {
            result, err := sm.ValidateSource(ctx, input)
            if err != nil {
                return []ValidationError{{Message: fmt.Sprintf("provider validation error: %v", err)}}
            }
            if result != nil && !result.Valid {
                return result.Errors
            }
            return nil
        },
    }
}
```

To add custom validation to a new plugin:

1. Implement `SourceManager.ValidateSource()` in your plugin.
2. The framework automatically includes it as the provider layer when constructing `NewDefaultValidator(sm)`.
3. Your checks run after all generic layers have passed (or at least after the critical YAML parse layer).

To add an entirely new generic layer (rare):

1. Create a `ValidationLayer` with a `Check` function.
2. Call `validator.AddLayer(layer)` after `NewDefaultValidator()`, or fork the default chain.

## Key Files

| File | Purpose |
|------|---------|
| `pkg/catalog/plugin/validator.go` | `MultiLayerValidator`, all built-in layers, `NewDefaultValidator` |
| `pkg/catalog/plugin/management_types.go` | `SourceConfigInput`, `ValidationResult`, `ValidationError` types |
| `pkg/catalog/plugin/management_handlers.go` | HTTP handlers: `validateHandler`, `detailedValidateHandler`, `applyHandler` (implicit validation) |
| `pkg/catalog/plugin/redact.go` | `IsSensitiveKey()` used by the security warnings layer |

---

[Back to Source Management](./README.md) | [Prev: Config Stores](./config-stores.md) | [Next: Refresh and Diagnostics](./refresh-and-diagnostics.md)

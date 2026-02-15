# Performance Metrics

This document covers the performance metrics handling in the Catalog Service.

## Overview

Performance metrics allow models to be compared based on benchmark results including throughput, latency, and hardware configurations.

## Data Format

### NDJSON Input

Performance metrics are loaded from NDJSON (newline-delimited JSON) files:

```json
{"model_id": "meta-llama/Llama-3.1-8B-Instruct", "metric_name": "throughput_tps", "value": 1105.4, "hardware_type": "H100", "hardware_count": 2}
{"model_id": "meta-llama/Llama-3.1-8B-Instruct", "metric_name": "latency_p95_ms", "value": 108.3, "hardware_type": "H100", "hardware_count": 2}
{"model_id": "microsoft/phi-2", "metric_name": "throughput_tps", "value": 2340.1, "hardware_type": "A100", "hardware_count": 1}
```

### Record Structure

```go
type evaluationRecord struct {
    ModelID       string             `json:"model_id"`
    MetricName    string             `json:"metric_name"`
    Value         float64            `json:"value"`
    HardwareType  string             `json:"hardware_type"`
    HardwareCount int                `json:"hardware_count"`
    Metadata      map[string]any     `json:"metadata"`
}
```

## Metric Types

### Standard Metrics

| Metric | Unit | Description |
|--------|------|-------------|
| `throughput_tps` | tokens/sec | Token generation throughput |
| `latency_p50_ms` | milliseconds | 50th percentile latency |
| `latency_p95_ms` | milliseconds | 95th percentile latency |
| `latency_p99_ms` | milliseconds | 99th percentile latency |
| `memory_gb` | gigabytes | Peak memory usage |
| `accuracy` | 0.0-1.0 | Model accuracy score |
| `f1_score` | 0.0-1.0 | F1 score |

### Hardware Configurations

| Hardware | Description |
|----------|-------------|
| `H100` | NVIDIA H100 GPU |
| `A100` | NVIDIA A100 GPU |
| `A10` | NVIDIA A10 GPU |
| `T4` | NVIDIA T4 GPU |
| `CPU` | CPU-only |

## Loading Metrics

### Performance Metrics Loader

```go
// catalog/internal/catalog/performance_metrics.go
type PerformanceMetricsLoader struct {
    metricsPath string
    db          *gorm.DB
}

func (l *PerformanceMetricsLoader) OnLoad(collection *SourceCollection) {
    if l.metricsPath == "" {
        return
    }

    // Read metrics files
    files, _ := filepath.Glob(filepath.Join(l.metricsPath, "*.ndjson"))

    for _, file := range files {
        l.loadMetricsFile(file, collection)
    }
}

func (l *PerformanceMetricsLoader) loadMetricsFile(path string, collection *SourceCollection) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        var record evaluationRecord
        if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
            continue
        }

        // Find model in collection
        model := collection.FindModel(record.ModelID)
        if model == nil {
            continue
        }

        // Create metrics artifact
        metric := &models.CatalogMetricsArtifact{
            CatalogModelID: model.ID,
            MetricName:     record.MetricName,
            MetricValue:    record.Value,
            HardwareType:   &record.HardwareType,
            HardwareCount:  &record.HardwareCount,
        }

        l.db.Create(metric)
    }

    return scanner.Err()
}
```

## API Access

### Get Performance Artifacts

```bash
GET /api/model_catalog/v1alpha1/sources/{sourceId}/models/{modelName}/performance_artifacts
```

### Response

```json
{
  "items": [
    {
      "metricName": "throughput_tps",
      "metricValue": 1105.4,
      "hardwareType": "H100",
      "hardwareCount": 2,
      "customProperties": {
        "framework": {
          "metadataType": "MetadataStringValue",
          "string_value": "vllm"
        }
      }
    },
    {
      "metricName": "latency_p95_ms",
      "metricValue": 108.3,
      "hardwareType": "H100",
      "hardwareCount": 2
    }
  ]
}
```

## Performance Filtering

### Filter by Performance

```bash
# Models with throughput > 1000 tps
GET /models?filterQuery=customProperties.throughput_tps.double_value>=1000

# Models tested on H100
GET /models?filterQuery=customProperties.hardware_type.string_value="H100"
```

### Sort by Performance

```bash
# Sort by throughput descending
GET /models?orderBy=THROUGHPUT&sortOrder=DESC

# Sort by latency ascending
GET /models?orderBy=LATENCY&sortOrder=ASC
```

### Order By Options

| Value | Description |
|-------|-------------|
| `NAME` | Model name (default) |
| `CREATE_TIME` | Creation timestamp |
| `ACCURACY` | Accuracy score |
| `THROUGHPUT` | Throughput (tokens/sec) |
| `LATENCY` | P95 latency |

## Performance Filters UI

### Filter Components

The frontend provides specialized filters for performance metrics:

```typescript
// Performance filter configuration
const performanceFilters = [
  {
    field: 'throughput_tps',
    label: 'Throughput (tokens/sec)',
    type: 'slider',
    min: 0,
    max: 5000,
    step: 100,
  },
  {
    field: 'latency_p95_ms',
    label: 'Latency P95 (ms)',
    type: 'slider',
    min: 0,
    max: 1000,
    step: 10,
  },
  {
    field: 'hardware_type',
    label: 'Hardware',
    type: 'select',
    options: ['H100', 'A100', 'A10', 'T4'],
  },
];
```

### Hardware Configuration Table

```typescript
interface HardwareConfig {
  hardwareType: string;
  hardwareCount: number;
  metrics: {
    throughput: number;
    latencyP95: number;
    memoryGb: number;
  };
}
```

## Compression Metrics

For models that support compression:

```json
{
  "model_id": "meta-llama/Llama-3.1-8B-Instruct",
  "compression_level": "fp16",
  "throughput_tps": 1105.4,
  "model_size_gb": 15.2
}

{
  "model_id": "meta-llama/Llama-3.1-8B-Instruct",
  "compression_level": "int8",
  "throughput_tps": 1450.2,
  "model_size_gb": 8.1
}

{
  "model_id": "meta-llama/Llama-3.1-8B-Instruct",
  "compression_level": "int4",
  "throughput_tps": 1820.5,
  "model_size_gb": 4.2
}
```

### Compression Comparison Card

```typescript
interface CompressionComparison {
  modelId: string;
  compressionLevels: Array<{
    level: string;
    throughput: number;
    modelSize: number;
    accuracyDelta: number;
  }>;
}
```

## Database Storage

### Metrics Artifact Table

```sql
CREATE TABLE catalog_metrics_artifact (
    id INT PRIMARY KEY AUTO_INCREMENT,
    catalog_model_id INT NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DOUBLE NOT NULL,
    hardware_type VARCHAR(50),
    hardware_count INT,
    compression_level VARCHAR(50),
    FOREIGN KEY (catalog_model_id) REFERENCES catalog_model(id) ON DELETE CASCADE
);

CREATE INDEX idx_metrics_model ON catalog_metrics_artifact(catalog_model_id);
CREATE INDEX idx_metrics_hardware ON catalog_metrics_artifact(hardware_type);
```

### Property Table Integration

For filtering, metrics are also stored as custom properties:

```sql
INSERT INTO catalog_model_property (catalog_model_id, name, is_custom_property, double_value)
SELECT catalog_model_id, metric_name, true, metric_value
FROM catalog_metrics_artifact
WHERE hardware_type = 'H100';  -- Default hardware for filtering
```

---

[Back to Catalog Service Index](./README.md) | [Previous: Database Models](./database-models.md)

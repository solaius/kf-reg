# Source Providers

This document covers the pluggable source providers for the Model Catalog Service.

## Overview

Source providers fetch models from external catalogs. The service supports:

- **YAML Catalog** - Static file-based model definitions
- **HuggingFace Hub** - Real-time discovery from HuggingFace

## YAML Catalog Provider

### Configuration

```yaml
catalogs:
  - id: "my-models"
    name: "Organization Models"
    type: "yaml"
    enabled: true
    labels: ["internal", "production"]
    properties:
      yamlCatalogPath: "./models.yaml"
```

### YAML Catalog Format

```yaml
# models.yaml
models:
  - name: my-regression-model
    description: Sales forecasting model
    maturity: Production
    language: ["Python"]
    tasks: ["regression", "forecasting"]
    libraryName: scikit-learn
    license: Apache-2.0
    provider: My Organization
    customProperties:
      model_type:
        metadataType: MetadataStringValue
        string_value: "predictive"
    artifacts:
      - uri: oci://registry.example.com/models/sales-forecast:v1.0
        customProperties:
          format:
            metadataType: MetadataStringValue
            string_value: "pickle"
```

### Implementation

```go
// catalog/internal/catalog/yaml_catalog.go
func YAMLModelProvider(source *Source) ([]CatalogModel, error) {
    path := source.Properties["yamlCatalogPath"]

    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var catalog yamlCatalog
    if err := yaml.Unmarshal(data, &catalog); err != nil {
        return nil, err
    }

    models := make([]CatalogModel, len(catalog.Models))
    for i, m := range catalog.Models {
        models[i] = convertYAMLModel(m)
    }

    return models, nil
}

type yamlCatalog struct {
    Models []yamlModel `yaml:"models"`
}

type yamlModel struct {
    Name             string                       `yaml:"name"`
    Description      string                       `yaml:"description"`
    Readme           string                       `yaml:"readme"`
    Maturity         string                       `yaml:"maturity"`
    Language         []string                     `yaml:"language"`
    Tasks            []string                     `yaml:"tasks"`
    LibraryName      string                       `yaml:"libraryName"`
    License          string                       `yaml:"license"`
    LicenseLink      string                       `yaml:"licenseLink"`
    Provider         string                       `yaml:"provider"`
    CustomProperties map[string]yamlMetadataValue `yaml:"customProperties"`
    Artifacts        []yamlArtifact               `yaml:"artifacts"`
}
```

## HuggingFace Hub Provider

### Configuration

```yaml
catalogs:
  - id: "huggingface"
    name: "Hugging Face Hub"
    type: "hf"
    enabled: true
    labels: ["external", "llm"]
    includedModels:
      - "meta-llama/Llama-3.1-8B-Instruct"
      - "microsoft/phi-*"  # Wildcard pattern
    excludedModels:
      - "meta-llama/*-draft"
    properties:
      apiKeyEnvVar: "HF_API_KEY"
      allowedOrganization: "meta-llama"  # Optional
```

### Implementation

```go
// catalog/internal/catalog/hf_catalog.go
func HFModelProvider(source *Source) ([]CatalogModel, error) {
    // Get API key from environment
    apiKeyEnvVar := source.Properties["apiKeyEnvVar"]
    if apiKeyEnvVar == "" {
        apiKeyEnvVar = "HF_API_KEY"
    }
    apiKey := os.Getenv(apiKeyEnvVar)

    // Create HF client
    client := newHFClient(apiKey)

    // Process included models
    var models []CatalogModel
    for _, pattern := range source.IncludedModels {
        // Apply allowedOrganization prefix
        fullPattern := applyOrgPrefix(pattern, source.Properties["allowedOrganization"])

        // Fetch matching models
        fetched, err := client.fetchModels(fullPattern)
        if err != nil {
            log.Printf("Failed to fetch models for pattern %s: %v", pattern, err)
            continue
        }

        // Filter excluded
        for _, m := range fetched {
            if !isExcluded(m.Name, source.ExcludedModels) {
                models = append(models, convertHFModel(m))
            }
        }
    }

    return models, nil
}
```

### Pattern Matching

```go
// catalog/internal/catalog/model_filter.go
func (f *ModelFilter) Match(modelName string) bool {
    // Check exclusions first
    for _, pattern := range f.excludePatterns {
        if pattern.Match(modelName) {
            return false
        }
    }

    // Check inclusions
    for _, pattern := range f.includePatterns {
        if pattern.Match(modelName) {
            return true
        }
    }

    return false
}

func patternToRegexp(pattern string) (*regexp.Regexp, error) {
    // Convert glob to regex
    // "meta-llama/*" -> "^meta-llama/.*$"
    // "phi-*" -> "^phi-.*$"

    escaped := regexp.QuoteMeta(pattern)
    regexPattern := strings.ReplaceAll(escaped, "\\*", ".*")
    regexPattern = "^" + regexPattern + "$"

    return regexp.Compile("(?i)" + regexPattern)
}
```

### Organization-Restricted Sources

```go
func applyOrgPrefix(pattern string, allowedOrg string) string {
    if allowedOrg == "" {
        return pattern
    }

    // Pattern is relative to organization
    if pattern == "*" {
        return allowedOrg + "/*"
    }

    if !strings.Contains(pattern, "/") {
        return allowedOrg + "/" + pattern
    }

    return pattern
}
```

## Adding New Providers

### 1. Create Provider Function

```go
// catalog/internal/catalog/my_provider.go
func MyProviderFunc(source *Source) ([]CatalogModel, error) {
    // Get configuration from source.Properties
    endpoint := source.Properties["endpoint"]

    // Fetch models from your source
    models, err := fetchFromMySource(endpoint)
    if err != nil {
        return nil, err
    }

    // Convert to CatalogModel
    result := make([]CatalogModel, len(models))
    for i, m := range models {
        result[i] = CatalogModel{
            Name:        m.Name,
            Description: m.Description,
            // ... other fields
        }
    }

    return result, nil
}
```

### 2. Register Provider

```go
// In your catalog initialization
loader.RegisterModelProvider("my-provider", MyProviderFunc)
```

### 3. Configure Source

```yaml
catalogs:
  - id: "my-source"
    name: "My Custom Source"
    type: "my-provider"
    enabled: true
    properties:
      endpoint: "https://my-source.example.com/api"
```

## Source Labels

Labels enable filtering sources by category:

```yaml
catalogs:
  - id: "internal-models"
    type: "yaml"
    labels: ["internal", "production"]

  - id: "hf-llms"
    type: "hf"
    labels: ["external", "llm"]

labels:
  - name: "internal"
    key: "source-type"
  - name: "external"
    key: "source-type"
```

**API Usage:**

```bash
# Get only internal sources
GET /api/model_catalog/v1alpha1/models?sourceLabel=internal
```

---

[Back to Catalog Service Index](./README.md) | [Previous: Architecture](./architecture.md) | [Next: Filtering System](./filtering-system.md)

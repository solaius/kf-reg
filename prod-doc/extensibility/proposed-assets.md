# Proposed Assets

This document outlines potential future AI asset types that could be added to the Kubeflow Model Registry.

## Current State

### Implemented Assets

| Asset Type | Status | Description |
|------------|--------|-------------|
| RegisteredModel | Production | ML model container |
| ModelVersion | Production | Model version with artifacts |
| ModelArtifact | Production | Physical model files |
| InferenceService | Production | Deployed model serving |
| CatalogModel | Production | Curated model metadata |
| McpServer | Feature Branch | MCP server definitions |
| McpTool | Feature Branch | MCP server tools |

## Proposed AI Asset Types

### 1. Prompts

**Purpose**: Manage and version prompt templates for LLM applications.

**Use Cases**:
- Store and version prompt templates
- Share prompts across teams
- Track prompt performance
- A/B test prompt variations

**Proposed Schema**:

```yaml
Prompt:
  properties:
    name: string
    description: string
    template: string          # The prompt text with {{variables}}
    variables:
      - name: string
        type: string|number|boolean|json
        required: boolean
        defaultValue: any
    systemMessage: string     # Optional system prompt
    category: string          # classification, generation, etc.
    modelCompatibility:       # Which models work with this prompt
      - modelName: string
        minVersion: string
    inputSchema: JSONSchema   # Expected input format
    outputSchema: JSONSchema  # Expected output format
    examples:                 # Few-shot examples
      - input: object
        output: object
    version: string
    tags: string[]
    state: LIVE|ARCHIVED
```

**API Endpoints**:
- `GET /prompts` - List prompts
- `POST /prompts` - Create prompt
- `GET /prompts/{id}` - Get prompt
- `PATCH /prompts/{id}` - Update prompt
- `GET /prompts/{id}/versions` - List versions
- `POST /prompts/{id}/test` - Test prompt execution

---

### 2. Knowledge Sources

**Purpose**: Manage knowledge bases for RAG (Retrieval Augmented Generation) applications.

**Use Cases**:
- Index document collections
- Track embedding versions
- Manage vector store configurations
- Link knowledge to models/prompts

**Proposed Schema**:

```yaml
KnowledgeSource:
  properties:
    name: string
    description: string
    sourceType: documents|database|api|web
    sourceConfig:
      uri: string
      credentials: SecretReference
    embeddingModel:
      modelId: string         # Reference to RegisteredModel
      version: string
    vectorStore:
      type: pinecone|chroma|milvus|pgvector
      config: object
    chunkingConfig:
      strategy: fixed|semantic|recursive
      chunkSize: integer
      overlap: integer
    indexStatus: pending|indexing|ready|error
    documentCount: integer
    lastIndexed: timestamp
    refreshSchedule: string   # Cron expression
    state: LIVE|ARCHIVED
```

**API Endpoints**:
- `GET /knowledge_sources` - List sources
- `POST /knowledge_sources` - Create source
- `GET /knowledge_sources/{id}` - Get source
- `POST /knowledge_sources/{id}/reindex` - Trigger reindexing
- `GET /knowledge_sources/{id}/documents` - List indexed documents
- `POST /knowledge_sources/{id}/query` - Query knowledge base

---

### 3. Guardrails

**Purpose**: Define and manage safety constraints for AI applications.

**Use Cases**:
- Content moderation rules
- Output validation
- Input sanitization
- Compliance enforcement

**Proposed Schema**:

```yaml
Guardrail:
  properties:
    name: string
    description: string
    type: input|output|both
    category: safety|quality|compliance|custom
    rules:
      - name: string
        condition: string     # Expression or pattern
        action: block|warn|transform|log
        message: string
        priority: integer
    validationModel:          # Optional ML-based validation
      modelId: string
      threshold: float
    blockedPatterns:
      - pattern: string
        type: regex|keyword|semantic
    allowedTopics: string[]
    blockedTopics: string[]
    piiDetection:
      enabled: boolean
      types: [email, phone, ssn, ...]
      action: block|mask|warn
    metrics:
      triggeredCount: integer
      blockedCount: integer
    state: LIVE|ARCHIVED|TESTING
```

**API Endpoints**:
- `GET /guardrails` - List guardrails
- `POST /guardrails` - Create guardrail
- `GET /guardrails/{id}` - Get guardrail
- `POST /guardrails/{id}/test` - Test guardrail
- `GET /guardrails/{id}/metrics` - Get guardrail metrics

---

### 4. Agents

**Purpose**: Manage AI agent configurations and orchestration metadata.

**Use Cases**:
- Define agent capabilities
- Configure tool access
- Manage agent workflows
- Track agent performance

**Proposed Schema**:

```yaml
Agent:
  properties:
    name: string
    description: string
    type: single|multi|hierarchical
    baseModel:
      modelId: string
      version: string
    systemPrompt: string
    tools:                    # Available tools/functions
      - toolId: string        # McpServer or custom tool reference
        permissions: string[]
    memory:
      type: none|buffer|summary|vector
      config: object
    planningStrategy: react|cot|tot|custom
    maxIterations: integer
    timeout: duration
    guardrails:
      - guardrailId: string
    knowledgeSources:
      - sourceId: string
    workflow:                 # For multi-agent
      steps:
        - agentId: string
          condition: string
          outputs: string[]
    metrics:
      totalInvocations: integer
      avgLatency: duration
      successRate: float
    state: LIVE|ARCHIVED|TESTING
```

**API Endpoints**:
- `GET /agents` - List agents
- `POST /agents` - Create agent
- `GET /agents/{id}` - Get agent
- `POST /agents/{id}/invoke` - Invoke agent
- `GET /agents/{id}/runs` - List agent runs
- `GET /agents/{id}/metrics` - Get agent metrics

---

### 5. Notebooks

**Purpose**: Manage Jupyter notebooks with ML artifacts.

**Use Cases**:
- Version notebooks
- Track notebook outputs
- Link to models and datasets
- Share reproducible experiments

**Proposed Schema**:

```yaml
Notebook:
  properties:
    name: string
    description: string
    notebookUri: string       # S3/GCS path to .ipynb
    environment:
      image: string
      requirements: string[]
      kernelSpec: object
    parameters:               # Papermill-style parameters
      - name: string
        type: string
        defaultValue: any
    outputs:
      - name: string
        type: figure|table|model|dataset
        uri: string
    linkedArtifacts:
      models:
        - modelId: string
          role: input|output
      datasets:
        - datasetId: string
          role: input|output
    executionHistory:
      - executedAt: timestamp
        duration: duration
        status: success|error
        outputUri: string
    tags: string[]
    state: LIVE|ARCHIVED
```

**API Endpoints**:
- `GET /notebooks` - List notebooks
- `POST /notebooks` - Create notebook
- `GET /notebooks/{id}` - Get notebook
- `POST /notebooks/{id}/execute` - Execute notebook
- `GET /notebooks/{id}/executions` - List executions

---

### 6. Pipelines

**Purpose**: Manage ML pipeline definitions and metadata.

**Use Cases**:
- Store pipeline configurations
- Track pipeline versions
- Link to pipeline platforms (Kubeflow Pipelines, Airflow)
- Manage pipeline artifacts

**Proposed Schema**:

```yaml
Pipeline:
  properties:
    name: string
    description: string
    platform: kubeflow|airflow|argo|custom
    definitionUri: string     # URI to pipeline definition
    definitionFormat: yaml|python|json
    parameters:
      - name: string
        type: string
        required: boolean
        defaultValue: any
    components:
      - name: string
        type: data|training|evaluation|deployment
        image: string
        inputs: string[]
        outputs: string[]
    artifacts:
      inputs:
        - name: string
          artifactType: dataset|model|config
      outputs:
        - name: string
          artifactType: model|metrics|report
    schedule: string          # Cron expression
    lastRun:
      runId: string
      status: string
      completedAt: timestamp
    state: LIVE|ARCHIVED
```

**API Endpoints**:
- `GET /pipelines` - List pipelines
- `POST /pipelines` - Create pipeline
- `GET /pipelines/{id}` - Get pipeline
- `POST /pipelines/{id}/trigger` - Trigger pipeline run
- `GET /pipelines/{id}/runs` - List runs

---

### 7. Skills / Commands

**Purpose**: Manage reusable AI capabilities or commands.

**Use Cases**:
- Define composable AI skills
- Share skills across agents
- Version skill implementations
- Track skill usage

**Proposed Schema**:

```yaml
Skill:
  properties:
    name: string
    description: string
    category: string
    inputSchema: JSONSchema
    outputSchema: JSONSchema
    implementation:
      type: prompt|code|agent|pipeline
      config:
        # Type-specific configuration
        promptId: string      # For prompt type
        codeUri: string       # For code type
        agentId: string       # For agent type
        pipelineId: string    # For pipeline type
    dependencies:
      skills: string[]        # Other skills this depends on
      tools: string[]         # MCP tools required
      models: string[]        # Models required
    examples:
      - input: object
        output: object
        description: string
    metrics:
      invocations: integer
      avgLatency: duration
      successRate: float
    state: LIVE|ARCHIVED
```

**API Endpoints**:
- `GET /skills` - List skills
- `POST /skills` - Create skill
- `GET /skills/{id}` - Get skill
- `POST /skills/{id}/invoke` - Invoke skill
- `GET /skills/{id}/metrics` - Get metrics

---

### 8. Datasets

**Purpose**: Manage dataset metadata and versions.

**Use Cases**:
- Track training datasets
- Version dataset snapshots
- Link datasets to models
- Manage data lineage

**Proposed Schema**:

```yaml
Dataset:
  properties:
    name: string
    description: string
    storageUri: string
    format: csv|parquet|json|tfrecord|arrow
    schema:
      columns:
        - name: string
          type: string
          nullable: boolean
    statistics:
      rowCount: integer
      sizeBytes: integer
      columnStats: object
    partitioning:
      columns: string[]
      strategy: string
    version: string
    lineage:
      sourceDatasets:
        - datasetId: string
          relationship: derived|sampled|filtered
      transformations:
        - type: string
          description: string
    quality:
      completeness: float
      uniqueness: float
      validationRules: object[]
    tags: string[]
    state: LIVE|ARCHIVED
```

**API Endpoints**:
- `GET /datasets` - List datasets
- `POST /datasets` - Create dataset
- `GET /datasets/{id}` - Get dataset
- `GET /datasets/{id}/versions` - List versions
- `GET /datasets/{id}/statistics` - Get statistics
- `GET /datasets/{id}/lineage` - Get lineage graph

---

## Implementation Priority

### Phase 1: Foundation (High Priority)

1. **Prompts** - Essential for LLM applications
2. **Guardrails** - Critical for safe AI deployment

### Phase 2: RAG Support (Medium Priority)

3. **Knowledge Sources** - Enable RAG applications
4. **Datasets** - Support data management

### Phase 3: Orchestration (Lower Priority)

5. **Agents** - Complex multi-agent systems
6. **Pipelines** - ML workflow integration
7. **Notebooks** - Experiment tracking
8. **Skills** - Composable AI capabilities

## Shared Infrastructure

### Property System Extension

All new asset types should leverage the existing `customProperties` system:

```go
type CustomProperties map[string]PropertyValue

// Extended for complex types
type PropertyValue struct {
    BoolValue   *bool
    IntValue    *int64
    DoubleValue *float64
    StringValue *string
    // New additions
    JsonValue   *string    // For complex objects
    ListValue   *[]string  // For arrays
}
```

### Relationship Management

Assets should support cross-references:

```yaml
relationships:
  - fromType: Agent
    toType: RegisteredModel
    relationship: uses
  - fromType: Prompt
    toType: Agent
    relationship: configures
  - fromType: Guardrail
    toType: InferenceService
    relationship: protects
```

### Unified Search

All assets should be searchable via a common interface:

```yaml
/search:
  parameters:
    query: string
    types: asset_type[]
    filters: FilterExpression
  response:
    results:
      - type: string
        id: string
        name: string
        description: string
        score: float
```

## Conclusion

The Model Registry architecture is well-suited for extension to support these additional AI asset types. The existing patterns for entities, repositories, and API design can be applied consistently across new asset types.

Key recommendations:
1. Start with Prompts and Guardrails as they have immediate utility
2. Establish relationship patterns early for cross-asset references
3. Build unified search infrastructure for asset discovery
4. Consider a plugin architecture for external integrations

---

[Back to Extensibility Index](./README.md) | [Previous: Adding New Assets](./adding-new-assets.md)

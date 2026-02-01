# Filtering System

This document covers the query filtering and named queries in the Catalog Service.

## Overview

The Catalog Service supports advanced filtering through:

- **Free-form search** (`q` parameter)
- **Filter query DSL** (`filterQuery` parameter)
- **Named queries** (pre-defined filter configurations)

## Free-Form Search

The `q` parameter searches across multiple text fields:

```bash
GET /api/model_catalog/v1alpha1/models?source=hf&q=llama
```

### Searched Fields

- `name`
- `description`
- `provider`

### Implementation

```go
func (c *dbCatalogImpl) applySearch(query *gorm.DB, searchTerm string) *gorm.DB {
    term := "%" + searchTerm + "%"
    return query.Where(
        "name LIKE ? OR description LIKE ? OR provider LIKE ?",
        term, term, term,
    )
}
```

## Filter Query DSL

### Syntax

```
filterQuery = expression ( AND|OR expression )*
expression = field operator value | "(" filterQuery ")"
operator = "=" | "!=" | "IN" | "CONTAINS"
value = string | number | "(" value ("," value)* ")"
```

### Examples

```bash
# Exact match
GET /models?filterQuery=maturity="Production"

# Not equals
GET /models?filterQuery=state!="ARCHIVED"

# IN list
GET /models?filterQuery=maturity IN ("Production", "Beta")

# Contains (for arrays)
GET /models?filterQuery=tasks CONTAINS "classification"

# Custom property
GET /models?filterQuery=customProperties.model_type.string_value="generative"

# Combined
GET /models?filterQuery=maturity="Production" AND tasks CONTAINS "text-generation"
```

### Implementation

```go
// catalog/internal/db/filter/parser.go
type FilterExpression struct {
    Left     *FilterExpression
    Operator string  // AND, OR
    Right    *FilterExpression
    Field    string
    Op       string  // =, !=, IN, CONTAINS
    Value    any
}

func Parse(query string) (*FilterExpression, error) {
    // Uses participle/v2 for parsing
    parser := participle.MustBuild(
        &FilterExpression{},
        participle.Lexer(filterLexer),
    )

    var expr FilterExpression
    if err := parser.ParseString("", query, &expr); err != nil {
        return nil, err
    }

    return &expr, nil
}
```

### Query Builder

```go
// catalog/internal/db/filter/query_builder.go
type QueryBuilder struct {
    entityType   EntityType
    tablePrefix  string
    mappingFuncs EntityMappingFunctions
}

func (q *QueryBuilder) Build(db *gorm.DB, expr *FilterExpression) *gorm.DB {
    switch {
    case expr.Operator == "AND":
        db = q.Build(db, expr.Left)
        db = q.Build(db, expr.Right)
        return db

    case expr.Operator == "OR":
        leftDB := q.Build(db, expr.Left)
        rightDB := q.Build(db, expr.Right)
        return db.Where(leftDB).Or(rightDB)

    default:
        return q.buildComparison(db, expr)
    }
}

func (q *QueryBuilder) buildComparison(db *gorm.DB, expr *FilterExpression) *gorm.DB {
    // Get property definition
    propDef := q.mappingFuncs.GetPropertyDefinitionForRestEntity(
        q.entityType, expr.Field,
    )

    // Handle custom properties
    if strings.HasPrefix(expr.Field, "customProperties.") {
        return q.buildCustomPropertyQuery(db, expr)
    }

    // Build standard comparison
    column := q.tablePrefix + "." + propDef.ColumnName
    switch expr.Op {
    case "=":
        return db.Where(column+" = ?", expr.Value)
    case "!=":
        return db.Where(column+" != ?", expr.Value)
    case "IN":
        return db.Where(column+" IN ?", expr.Value.([]any))
    case "CONTAINS":
        return db.Where("JSON_CONTAINS("+column+", ?)", expr.Value)
    }

    return db
}
```

## Named Queries

Named queries are pre-defined filter configurations stored in the sources.yaml:

### Configuration

```yaml
namedQueries:
  production-models:
    maturity:
      operator: "="
      value: "Production"

  generative-ai:
    customProperties.model_type.string_value:
      operator: "="
      value: "generative"

  high-accuracy:
    customProperties.accuracy.double_value:
      operator: ">="
      value: 0.95

  performance-tier:
    customProperties.throughput_tps.double_value:
      operator: ">="
      # Special substitution for min/max values
      value: "@min"
```

### Usage

```bash
# Use named query
GET /models?source=my-catalog&namedQuery=production-models
```

### Min/Max Substitution

For performance filters, `@min` and `@max` are substituted with actual values:

```go
func (l *Loader) substituteMinMax(query *NamedQuery, field string) {
    if query.Value == "@min" {
        query.Value = l.getMinValue(field)
    } else if query.Value == "@max" {
        query.Value = l.getMaxValue(field)
    }
}
```

## Filter Options

The `/filter_options` endpoint returns available filter values:

```bash
GET /api/model_catalog/v1alpha1/models/filter_options?source=hf
```

### Response

```json
{
  "maturity": ["Production", "Beta", "Alpha"],
  "tasks": ["text-generation", "classification", "summarization"],
  "language": ["English", "Python", "JavaScript"],
  "provider": ["meta-llama", "microsoft", "google"],
  "customProperties": {
    "model_type": ["predictive", "generative", "unknown"]
  }
}
```

### Implementation

```go
// Materialized view for performance
type PropertyOption struct {
    SourceID string
    Field    string
    Value    string
}

func (c *dbCatalogImpl) GetFilterOptions(ctx context.Context) (*openapi.FilterOptions, error) {
    var options []PropertyOption
    c.db.Model(&PropertyOption{}).Find(&options)

    result := make(map[string][]string)
    for _, opt := range options {
        result[opt.Field] = append(result[opt.Field], opt.Value)
    }

    return &openapi.FilterOptions{Options: result}, nil
}
```

### Property Options Refresher

```go
type PropertyOptionsRefresher struct {
    DB *gorm.DB
}

func (p *PropertyOptionsRefresher) OnLoad(collection *SourceCollection) {
    // Refresh materialized view
    p.DB.Exec("TRUNCATE TABLE property_options")

    for sourceID, models := range collection.GetModels() {
        for _, model := range models {
            // Extract unique values for each field
            p.extractOptions(sourceID, model)
        }
    }
}
```

## Entity Mappings

Maps REST fields to database columns:

```go
// catalog/internal/db/filter/entity_mappings.go
var catalogModelMappings = map[string]PropertyDefinition{
    "name":        {ColumnName: "name", Type: PropertyTypeString},
    "description": {ColumnName: "description", Type: PropertyTypeString},
    "maturity":    {ColumnName: "maturity", Type: PropertyTypeString},
    "license":     {ColumnName: "license", Type: PropertyTypeString},
    "provider":    {ColumnName: "provider", Type: PropertyTypeString},
    "tasks":       {ColumnName: "tasks", Type: PropertyTypeJSON},
    "language":    {ColumnName: "language", Type: PropertyTypeJSON},
}
```

---

[Back to Catalog Service Index](./README.md) | [Previous: Source Providers](./source-providers.md) | [Next: Database Models](./database-models.md)

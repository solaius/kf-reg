# Adding New Assets

This document provides a step-by-step guide for adding new asset types to the Kubeflow Model Registry.

## Overview

Adding a new asset type involves changes across multiple layers:

1. OpenAPI specification
2. Database models and migrations
3. Repository implementation
4. Service layer
5. API handlers (generated)
6. Frontend components
7. BFF handlers

## Step 1: Define OpenAPI Specification

### Location

For core entities: `api/openapi/model-registry.yaml`
For catalog entities: `api/openapi/catalog.yaml`

### Add Schema Definitions

```yaml
# api/openapi/model-registry.yaml

components:
  schemas:
    # Main entity
    Prompt:
      type: object
      required:
        - name
      properties:
        id:
          type: string
          readOnly: true
        name:
          type: string
          minLength: 1
          maxLength: 255
        description:
          type: string
        template:
          type: string
          description: The prompt template with placeholders
        variables:
          type: array
          items:
            $ref: '#/components/schemas/PromptVariable'
        category:
          type: string
        tags:
          type: array
          items:
            type: string
        state:
          $ref: '#/components/schemas/State'
        customProperties:
          $ref: '#/components/schemas/CustomProperties'
        createTimeSinceEpoch:
          type: integer
          format: int64
          readOnly: true
        lastUpdateTimeSinceEpoch:
          type: integer
          format: int64
          readOnly: true

    # Create variant
    PromptCreate:
      type: object
      required:
        - name
        - template
      properties:
        name:
          type: string
        description:
          type: string
        template:
          type: string
        variables:
          type: array
          items:
            $ref: '#/components/schemas/PromptVariable'
        category:
          type: string
        tags:
          type: array
          items:
            type: string
        customProperties:
          $ref: '#/components/schemas/CustomProperties'

    # Update variant
    PromptUpdate:
      type: object
      properties:
        description:
          type: string
        template:
          type: string
        variables:
          type: array
          items:
            $ref: '#/components/schemas/PromptVariable'
        category:
          type: string
        tags:
          type: array
          items:
            type: string
        state:
          $ref: '#/components/schemas/State'
        customProperties:
          $ref: '#/components/schemas/CustomProperties'

    # List response
    PromptList:
      type: object
      properties:
        items:
          type: array
          items:
            $ref: '#/components/schemas/Prompt'
        nextPageToken:
          type: string
        pageSize:
          type: integer
        size:
          type: integer

    # Supporting types
    PromptVariable:
      type: object
      required:
        - name
      properties:
        name:
          type: string
        description:
          type: string
        type:
          type: string
          enum: [string, number, boolean, json]
        required:
          type: boolean
        defaultValue:
          type: string
```

### Add Endpoints

```yaml
paths:
  /prompts:
    get:
      operationId: getPrompts
      summary: List prompts
      tags:
        - PromptService
      parameters:
        - $ref: '#/components/parameters/pageSize'
        - $ref: '#/components/parameters/orderBy'
        - $ref: '#/components/parameters/sortOrder'
        - $ref: '#/components/parameters/nextPageToken'
        - name: category
          in: query
          schema:
            type: string
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/PromptList'
    post:
      operationId: createPrompt
      summary: Create prompt
      tags:
        - PromptService
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/PromptCreate'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Prompt'

  /prompts/{promptId}:
    parameters:
      - name: promptId
        in: path
        required: true
        schema:
          type: string
    get:
      operationId: getPrompt
      summary: Get prompt by ID
      tags:
        - PromptService
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Prompt'
    patch:
      operationId: updatePrompt
      summary: Update prompt
      tags:
        - PromptService
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/PromptUpdate'
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Prompt'
```

## Step 2: Generate Code

```bash
# Validate specification
make openapi/validate

# Generate server stubs and models
make gen/openapi-server

# Generate client SDK
make gen/openapi
```

## Step 3: Create Database Model

### GORM Model

```go
// internal/db/models/prompt.go

package models

type Prompt struct {
    ID                       string `gorm:"primaryKey;type:varchar(255)"`
    Name                     string `gorm:"type:varchar(255);not null;uniqueIndex"`
    Description              string `gorm:"type:text"`
    Template                 string `gorm:"type:text;not null"`
    Variables                string `gorm:"type:json"` // JSON array
    Category                 string `gorm:"type:varchar(100)"`
    Tags                     string `gorm:"type:json"` // JSON array
    State                    string `gorm:"type:varchar(50);default:'LIVE'"`
    CustomProperties         string `gorm:"type:json"`
    CreateTimeSinceEpoch     int64  `gorm:"autoCreateTime:milli"`
    LastUpdateTimeSinceEpoch int64  `gorm:"autoUpdateTime:milli"`
}

func (Prompt) TableName() string {
    return "prompts"
}
```

### Migration

```sql
-- internal/datastore/embedmd/mysql/migrations/000X_add_prompts.up.sql

CREATE TABLE IF NOT EXISTS prompts (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    template TEXT NOT NULL,
    variables JSON,
    category VARCHAR(100),
    tags JSON,
    state VARCHAR(50) DEFAULT 'LIVE',
    custom_properties JSON,
    create_time_since_epoch BIGINT,
    last_update_time_since_epoch BIGINT,

    INDEX idx_name (name),
    INDEX idx_category (category),
    INDEX idx_state (state)
);
```

```sql
-- internal/datastore/embedmd/mysql/migrations/000X_add_prompts.down.sql

DROP TABLE IF EXISTS prompts;
```

### Run Migration

```bash
make gen/gorm
```

## Step 4: Implement Repository

```go
// internal/db/service/prompt_repository.go

package service

import (
    "github.com/kubeflow/model-registry/internal/db/models"
    "github.com/kubeflow/model-registry/pkg/openapi"
    "gorm.io/gorm"
)

type PromptRepository struct {
    db *gorm.DB
}

func NewPromptRepository(db *gorm.DB) *PromptRepository {
    return &PromptRepository{db: db}
}

func (r *PromptRepository) GetAll(opts ListOptions) ([]*openapi.Prompt, string, error) {
    var prompts []models.Prompt
    query := r.db.Model(&models.Prompt{})

    // Apply pagination
    if opts.PageSize > 0 {
        query = query.Limit(opts.PageSize + 1) // +1 to check for more
    }

    // Apply ordering
    if opts.OrderBy != "" {
        query = query.Order(opts.OrderBy + " " + opts.SortOrder)
    }

    if err := query.Find(&prompts).Error; err != nil {
        return nil, "", err
    }

    // Convert and handle pagination
    result, nextToken := paginate(prompts, opts.PageSize)
    return mapPromptsToAPI(result), nextToken, nil
}

func (r *PromptRepository) Get(id string) (*openapi.Prompt, error) {
    var prompt models.Prompt
    if err := r.db.First(&prompt, "id = ?", id).Error; err != nil {
        return nil, err
    }
    return mapPromptToAPI(&prompt), nil
}

func (r *PromptRepository) GetByName(name string) (*openapi.Prompt, error) {
    var prompt models.Prompt
    if err := r.db.First(&prompt, "name = ?", name).Error; err != nil {
        return nil, err
    }
    return mapPromptToAPI(&prompt), nil
}

func (r *PromptRepository) Create(p *openapi.PromptCreate) (*openapi.Prompt, error) {
    prompt := mapPromptCreateToModel(p)
    prompt.ID = generateID()

    if err := r.db.Create(&prompt).Error; err != nil {
        return nil, err
    }

    return mapPromptToAPI(&prompt), nil
}

func (r *PromptRepository) Update(id string, p *openapi.PromptUpdate) (*openapi.Prompt, error) {
    updates := mapPromptUpdateToModel(p)

    if err := r.db.Model(&models.Prompt{}).Where("id = ?", id).Updates(updates).Error; err != nil {
        return nil, err
    }

    return r.Get(id)
}
```

## Step 5: Implement Service Layer

```go
// internal/core/prompt_service.go

package core

import (
    "fmt"

    "github.com/kubeflow/model-registry/internal/db/service"
    "github.com/kubeflow/model-registry/pkg/openapi"
)

type PromptService interface {
    GetPrompts(opts ListOptions) (*openapi.PromptList, error)
    GetPrompt(id string) (*openapi.Prompt, error)
    CreatePrompt(create *openapi.PromptCreate) (*openapi.Prompt, error)
    UpdatePrompt(id string, update *openapi.PromptUpdate) (*openapi.Prompt, error)
}

type promptService struct {
    repo *service.PromptRepository
}

func NewPromptService(repo *service.PromptRepository) PromptService {
    return &promptService{repo: repo}
}

func (s *promptService) GetPrompts(opts ListOptions) (*openapi.PromptList, error) {
    prompts, nextToken, err := s.repo.GetAll(opts)
    if err != nil {
        return nil, fmt.Errorf("failed to get prompts: %w", err)
    }

    return &openapi.PromptList{
        Items:         prompts,
        NextPageToken: &nextToken,
        PageSize:      int32(opts.PageSize),
        Size:          int32(len(prompts)),
    }, nil
}

func (s *promptService) GetPrompt(id string) (*openapi.Prompt, error) {
    prompt, err := s.repo.Get(id)
    if err != nil {
        return nil, fmt.Errorf("failed to get prompt %s: %w", id, err)
    }
    return prompt, nil
}

func (s *promptService) CreatePrompt(create *openapi.PromptCreate) (*openapi.Prompt, error) {
    // Validate
    if create.Name == "" {
        return nil, ErrNameRequired
    }
    if create.Template == "" {
        return nil, ErrTemplateRequired
    }

    // Check uniqueness
    existing, _ := s.repo.GetByName(create.Name)
    if existing != nil {
        return nil, ErrDuplicateName
    }

    return s.repo.Create(create)
}

func (s *promptService) UpdatePrompt(id string, update *openapi.PromptUpdate) (*openapi.Prompt, error) {
    // Verify exists
    _, err := s.repo.Get(id)
    if err != nil {
        return nil, fmt.Errorf("prompt not found: %w", err)
    }

    return s.repo.Update(id, update)
}
```

## Step 6: Wire Up Handlers

The handlers are generated from OpenAPI. Wire them to the service:

```go
// internal/server/openapi/api_prompt_service.go (after generation)

func (s *PromptApiService) GetPrompts(ctx context.Context, ...) (ImplResponse, error) {
    opts := ListOptions{
        PageSize:  int(pageSize),
        OrderBy:   orderBy,
        SortOrder: sortOrder,
    }

    result, err := s.promptService.GetPrompts(opts)
    if err != nil {
        return Response(http.StatusInternalServerError, nil), err
    }

    return Response(http.StatusOK, result), nil
}
```

## Step 7: Add Frontend Components

### API Client

```typescript
// clients/ui/frontend/src/api/prompts.ts

export const getPrompts = async (params?: ListParams): Promise<PromptList> => {
  const response = await fetch(`/api/v1/prompts?${buildQueryString(params)}`);
  return response.json();
};

export const getPrompt = async (id: string): Promise<Prompt> => {
  const response = await fetch(`/api/v1/prompts/${id}`);
  return response.json();
};

export const createPrompt = async (prompt: PromptCreate): Promise<Prompt> => {
  const response = await fetch('/api/v1/prompts', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(prompt),
  });
  return response.json();
};
```

### Types

```typescript
// clients/ui/frontend/src/types/prompts.ts

export interface Prompt {
  id: string;
  name: string;
  description?: string;
  template: string;
  variables?: PromptVariable[];
  category?: string;
  tags?: string[];
  state: 'LIVE' | 'ARCHIVED';
  customProperties?: Record<string, PropertyValue>;
  createTimeSinceEpoch: number;
  lastUpdateTimeSinceEpoch: number;
}

export interface PromptCreate {
  name: string;
  description?: string;
  template: string;
  variables?: PromptVariable[];
  category?: string;
  tags?: string[];
  customProperties?: Record<string, PropertyValue>;
}

export interface PromptVariable {
  name: string;
  description?: string;
  type: 'string' | 'number' | 'boolean' | 'json';
  required?: boolean;
  defaultValue?: string;
}
```

### Page Component

```typescript
// clients/ui/frontend/src/app/pages/prompts/PromptsPage.tsx

import React, { useEffect, useState } from 'react';
import { Page, PageSection, Title, Toolbar } from '@patternfly/react-core';
import { getPrompts } from '~/api/prompts';
import { PromptTable } from './components/PromptTable';

export const PromptsPage: React.FC = () => {
  const [prompts, setPrompts] = useState<Prompt[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getPrompts()
      .then((data) => setPrompts(data.items))
      .finally(() => setIsLoading(false));
  }, []);

  return (
    <Page>
      <PageSection variant="light">
        <Title headingLevel="h1">Prompts</Title>
      </PageSection>
      <PageSection>
        <PromptTable prompts={prompts} isLoading={isLoading} />
      </PageSection>
    </Page>
  );
};
```

### Add Route

```typescript
// clients/ui/frontend/src/app/routes.tsx

import { PromptsPage } from './pages/prompts/PromptsPage';

export const routes = [
  // ... existing routes
  {
    path: '/prompts',
    element: <PromptsPage />,
  },
  {
    path: '/prompts/:promptId',
    element: <PromptDetailPage />,
  },
];
```

## Step 8: Update BFF

### Handler

```go
// clients/ui/bff/internal/api/prompts_handler.go

func (app *App) GetPromptsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    client, ok := r.Context().Value(constants.ModelRegistryHttpClientKey).(httpclient.HTTPClientInterface)
    if !ok {
        app.serverErrorResponse(w, r, errors.New("client not found"))
        return
    }

    prompts, err := app.repositories.PromptClient.GetAllPrompts(client, r.URL.Query())
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    response := PromptListEnvelope{Data: prompts}
    app.WriteJSON(w, http.StatusOK, response, nil)
}
```

### Route Registration

```go
// clients/ui/bff/internal/api/app.go

func (app *App) Routes() http.Handler {
    // ...existing routes...

    apiRouter.GET("/api/v1/prompts",
        app.AttachNamespace(
            app.AttachModelRegistryRESTClient(
                app.GetPromptsHandler)))

    // ...
}
```

## Step 9: Testing

### Unit Tests

```go
func TestCreatePrompt(t *testing.T) {
    repo := NewMockPromptRepository()
    service := NewPromptService(repo)

    create := &openapi.PromptCreate{
        Name:     "test-prompt",
        Template: "Hello {{name}}",
    }

    prompt, err := service.CreatePrompt(create)

    require.NoError(t, err)
    assert.Equal(t, "test-prompt", prompt.Name)
    assert.NotEmpty(t, prompt.Id)
}
```

### E2E Tests

```typescript
describe('Prompts', () => {
  it('should create a new prompt', () => {
    cy.visit('/prompts');
    cy.contains('Create Prompt').click();
    cy.get('[data-testid="name-input"]').type('Test Prompt');
    cy.get('[data-testid="template-input"]').type('Hello {{name}}');
    cy.contains('Save').click();
    cy.contains('Test Prompt').should('be.visible');
  });
});
```

## Checklist

- [ ] OpenAPI specification added
- [ ] Code generated (`make gen/openapi`)
- [ ] Database model created
- [ ] Migration written and tested
- [ ] Repository implemented
- [ ] Service layer implemented
- [ ] Handlers wired up
- [ ] Frontend types added
- [ ] API client functions added
- [ ] Page components created
- [ ] Routes configured
- [ ] BFF handlers added
- [ ] Unit tests written
- [ ] E2E tests written
- [ ] Documentation updated

---

[Back to Extensibility Index](./README.md) | [Previous: Asset Type Framework](./asset-type-framework.md) | [Next: Proposed Assets](./proposed-assets.md)

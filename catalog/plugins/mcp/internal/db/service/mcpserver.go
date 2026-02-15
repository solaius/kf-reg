package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/kubeflow/model-registry/catalog/plugins/mcp/internal/db/models"
	"github.com/kubeflow/model-registry/internal/db/schema"
	sharedmodels "github.com/kubeflow/model-registry/internal/db/models"
	"github.com/kubeflow/model-registry/internal/db/service"
)

var ErrMcpServerNotFound = errors.New("mcpserver not found")

// McpServerRepositoryImpl uses the shared GenericRepository for MLMD-based storage.
type McpServerRepositoryImpl struct {
	*service.GenericRepository[models.McpServer, schema.Context, schema.ContextProperty, *models.McpServerListOptions]
}

// NewMcpServerRepository creates a new McpServer repository.
func NewMcpServerRepository(db *gorm.DB, typeID int32) models.McpServerRepository {
	r := &McpServerRepositoryImpl{}

	r.GenericRepository = service.NewGenericRepository(service.GenericRepositoryConfig[
		models.McpServer,
		schema.Context,
		schema.ContextProperty,
		*models.McpServerListOptions,
	]{
		DB:                      db,
		TypeID:                  typeID,
		EntityToSchema:          mapMcpServerToContext,
		SchemaToEntity:          mapContextToMcpServer,
		EntityToProperties:      mapMcpServerToContextProperties,
		NotFoundError:           ErrMcpServerNotFound,
		EntityName:              "mcpserver",
		PropertyFieldName:       "context_id",
		ApplyListFilters:        applyMcpServerListFilters,
		IsNewEntity:             func(entity models.McpServer) bool { return entity.GetID() == nil },
		HasCustomProperties:     func(entity models.McpServer) bool { return entity.GetCustomProperties() != nil },
		PreserveHistoricalTimes: true,
		EntityMappingFuncs:      &entityMappings{},
	})

	return r
}

// Save saves or updates an entity.
func (r *McpServerRepositoryImpl) Save(entity models.McpServer) (models.McpServer, error) {
	config := r.GetConfig()
	if entity.GetTypeID() == nil {
		if config.TypeID > 0 {
			entity.SetTypeID(config.TypeID)
		}
	}

	// Check for existing entity by name if this is a new entity
	attr := entity.GetAttributes()
	if entity.GetID() == nil && attr != nil && attr.Name != nil {
		existing, err := r.GenericRepository.GetByName(*attr.Name)
		if err == nil {
			entity.SetID(*existing.GetID())
		} else if !errors.Is(err, ErrMcpServerNotFound) {
			return nil, fmt.Errorf("error finding existing entity: %w", err)
		}
	}

	return r.GenericRepository.Save(entity, nil)
}

// List returns entities matching the options.
func (r *McpServerRepositoryImpl) List(options models.McpServerListOptions) (*sharedmodels.ListWrapper[models.McpServer], error) {
	return r.GenericRepository.List(&options)
}

// DeleteBySource deletes all entities with the given source ID.
func (r *McpServerRepositoryImpl) DeleteBySource(sourceID string) error {
	config := r.GetConfig()

	return config.DB.Transaction(func(tx *gorm.DB) error {
		// Delete Context records where there's a ContextProperty with name='source_id' and matching value
		deleteContextQuery := `DELETE FROM "Context" WHERE id IN (
			SELECT "Context".id
			FROM "Context"
			INNER JOIN "ContextProperty" ON "Context".id="ContextProperty".context_id
				AND "ContextProperty".name='source_id'
				AND "ContextProperty".string_value=?
			WHERE "Context".type_id=?
		)`
		if err := tx.Exec(deleteContextQuery, sourceID, config.TypeID).Error; err != nil {
			return fmt.Errorf("error deleting mcpservers by source: %w", err)
		}
		return nil
	})
}

// DeleteByID deletes an entity by ID.
func (r *McpServerRepositoryImpl) DeleteByID(id int32) error {
	config := r.GetConfig()
	return config.DB.Where("id = ? AND type_id = ?", id, config.TypeID).Delete(&schema.Context{}).Error
}

// GetDistinctSourceIDs retrieves all unique source_id values.
func (r *McpServerRepositoryImpl) GetDistinctSourceIDs() ([]string, error) {
	config := r.GetConfig()
	var sourceIDs []string

	query := `SELECT DISTINCT string_value FROM "ContextProperty"
		WHERE name='source_id'
		AND context_id IN (SELECT id FROM "Context" WHERE type_id=?)`
	if err := config.DB.Raw(query, config.TypeID).Scan(&sourceIDs).Error; err != nil {
		return nil, fmt.Errorf("error getting distinct source IDs: %w", err)
	}
	return sourceIDs, nil
}

func mapMcpServerToContext(entity models.McpServer) schema.Context {
	ctx := schema.Context{}
	if entity.GetID() != nil {
		ctx.ID = *entity.GetID()
	}
	if entity.GetTypeID() != nil {
		ctx.TypeID = *entity.GetTypeID()
	}
	if attrs := entity.GetAttributes(); attrs != nil {
		if attrs.Name != nil {
			ctx.Name = *attrs.Name
		}
		if attrs.ExternalID != nil {
			ctx.ExternalID = attrs.ExternalID
		}
		if attrs.CreateTimeSinceEpoch != nil {
			ctx.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
		}
		if attrs.LastUpdateTimeSinceEpoch != nil {
			ctx.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
		}
	}
	return ctx
}

func mapContextToMcpServer(ctx schema.Context, props []schema.ContextProperty) models.McpServer {
	// Convert schema properties to model properties and extract known attributes
	var modelProps []sharedmodels.Properties

	for _, p := range props {
		modelProps = append(modelProps, service.MapContextPropertyToProperties(p))
	}

	entity := &models.McpServerImpl{
		ID:     &ctx.ID,
		TypeID: &ctx.TypeID,
		Attributes: &models.McpServerAttributes{
			Name:                     &ctx.Name,
			ExternalID:               ctx.ExternalID,
			CreateTimeSinceEpoch:     &ctx.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &ctx.LastUpdateTimeSinceEpoch,

		},
		Properties: &modelProps,
	}

	return entity
}

func mapMcpServerToContextProperties(entity models.McpServer, entityID int32) []schema.ContextProperty {
	var props []schema.ContextProperty

	// Add other properties
	if entity.GetProperties() != nil {
		for _, p := range *entity.GetProperties() {
			props = append(props, service.MapPropertiesToContextProperty(p, entityID, p.IsCustomProperty))
		}
	}

	return props
}

func applyMcpServerListFilters(db *gorm.DB, opts *models.McpServerListOptions) *gorm.DB {
	if opts == nil {
		return db
	}

	if opts.Name != nil && *opts.Name != "" {
		db = db.Where("name LIKE ?", "%"+*opts.Name+"%")
	}

	if opts.ExternalID != nil && *opts.ExternalID != "" {
		db = db.Where("external_id = ?", *opts.ExternalID)
	}

	// Filter by source IDs using context properties
	var nonEmptySourceIDs []string
	if opts.SourceIDs != nil {
		for _, sourceID := range *opts.SourceIDs {
			if sourceID != "" {
				nonEmptySourceIDs = append(nonEmptySourceIDs, sourceID)
			}
		}
	}

	if len(nonEmptySourceIDs) > 0 {
		db = db.Joins("JOIN \"ContextProperty\" cp ON \"Context\".id = cp.context_id").
			Where("cp.name = ? AND cp.string_value IN ?", "source_id", nonEmptySourceIDs)
	}

	return db
}

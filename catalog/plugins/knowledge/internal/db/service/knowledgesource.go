package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/kubeflow/model-registry/catalog/plugins/knowledge/internal/db/models"
	"github.com/kubeflow/model-registry/internal/db/schema"
	sharedmodels "github.com/kubeflow/model-registry/internal/db/models"
	"github.com/kubeflow/model-registry/internal/db/service"
)

var ErrKnowledgeSourceNotFound = errors.New("knowledgesource not found")

// KnowledgeSourceRepositoryImpl uses the shared GenericRepository for MLMD-based storage.
type KnowledgeSourceRepositoryImpl struct {
	*service.GenericRepository[models.KnowledgeSource, schema.Context, schema.ContextProperty, *models.KnowledgeSourceListOptions]
}

// NewKnowledgeSourceRepository creates a new KnowledgeSource repository.
func NewKnowledgeSourceRepository(db *gorm.DB, typeID int32) models.KnowledgeSourceRepository {
	r := &KnowledgeSourceRepositoryImpl{}

	r.GenericRepository = service.NewGenericRepository(service.GenericRepositoryConfig[
		models.KnowledgeSource,
		schema.Context,
		schema.ContextProperty,
		*models.KnowledgeSourceListOptions,
	]{
		DB:                      db,
		TypeID:                  typeID,
		EntityToSchema:          mapKnowledgeSourceToContext,
		SchemaToEntity:          mapContextToKnowledgeSource,
		EntityToProperties:      mapKnowledgeSourceToContextProperties,
		NotFoundError:           ErrKnowledgeSourceNotFound,
		EntityName:              "knowledgesource",
		PropertyFieldName:       "context_id",
		ApplyListFilters:        applyKnowledgeSourceListFilters,
		IsNewEntity:             func(entity models.KnowledgeSource) bool { return entity.GetID() == nil },
		HasCustomProperties:     func(entity models.KnowledgeSource) bool { return entity.GetCustomProperties() != nil },
		PreserveHistoricalTimes: true,
		EntityMappingFuncs:      &entityMappings{},
	})

	return r
}

// Save saves or updates an entity.
func (r *KnowledgeSourceRepositoryImpl) Save(entity models.KnowledgeSource) (models.KnowledgeSource, error) {
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
		} else if !errors.Is(err, ErrKnowledgeSourceNotFound) {
			return nil, fmt.Errorf("error finding existing entity: %w", err)
		}
	}

	return r.GenericRepository.Save(entity, nil)
}

// List returns entities matching the options.
func (r *KnowledgeSourceRepositoryImpl) List(options models.KnowledgeSourceListOptions) (*sharedmodels.ListWrapper[models.KnowledgeSource], error) {
	return r.GenericRepository.List(&options)
}

// DeleteBySource deletes all entities with the given source ID.
func (r *KnowledgeSourceRepositoryImpl) DeleteBySource(sourceID string) error {
	config := r.GetConfig()

	return config.DB.Transaction(func(tx *gorm.DB) error {
		deleteContextQuery := `DELETE FROM "Context" WHERE id IN (
			SELECT "Context".id
			FROM "Context"
			INNER JOIN "ContextProperty" ON "Context".id="ContextProperty".context_id
				AND "ContextProperty".name='source_id'
				AND "ContextProperty".string_value=?
			WHERE "Context".type_id=?
		)`
		if err := tx.Exec(deleteContextQuery, sourceID, config.TypeID).Error; err != nil {
			return fmt.Errorf("error deleting knowledgesources by source: %w", err)
		}
		return nil
	})
}

// DeleteByID deletes an entity by ID.
func (r *KnowledgeSourceRepositoryImpl) DeleteByID(id int32) error {
	config := r.GetConfig()
	return config.DB.Where("id = ? AND type_id = ?", id, config.TypeID).Delete(&schema.Context{}).Error
}

// GetDistinctSourceIDs retrieves all unique source_id values.
func (r *KnowledgeSourceRepositoryImpl) GetDistinctSourceIDs() ([]string, error) {
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

// CountBySource counts entities with the given source ID.
func (r *KnowledgeSourceRepositoryImpl) CountBySource(sourceID string) (int, error) {
	config := r.GetConfig()
	var count int64

	query := `SELECT COUNT(DISTINCT "Context".id) FROM "Context"
		INNER JOIN "ContextProperty" ON "Context".id="ContextProperty".context_id
			AND "ContextProperty".name='source_id'
			AND "ContextProperty".string_value=?
		WHERE "Context".type_id=?`
	if err := config.DB.Raw(query, sourceID, config.TypeID).Scan(&count).Error; err != nil {
		return 0, fmt.Errorf("error counting entities by source: %w", err)
	}
	return int(count), nil
}

func mapKnowledgeSourceToContext(entity models.KnowledgeSource) schema.Context {
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

func mapContextToKnowledgeSource(ctx schema.Context, props []schema.ContextProperty) models.KnowledgeSource {
	var modelProps []sharedmodels.Properties

	for _, p := range props {
		modelProps = append(modelProps, service.MapContextPropertyToProperties(p))
	}

	entity := &models.KnowledgeSourceImpl{
		ID:     &ctx.ID,
		TypeID: &ctx.TypeID,
		Attributes: &models.KnowledgeSourceAttributes{
			Name:                     &ctx.Name,
			ExternalID:               ctx.ExternalID,
			CreateTimeSinceEpoch:     &ctx.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &ctx.LastUpdateTimeSinceEpoch,
		},
		Properties: &modelProps,
	}

	return entity
}

func mapKnowledgeSourceToContextProperties(entity models.KnowledgeSource, entityID int32) []schema.ContextProperty {
	var props []schema.ContextProperty

	if entity.GetProperties() != nil {
		for _, p := range *entity.GetProperties() {
			props = append(props, service.MapPropertiesToContextProperty(p, entityID, p.IsCustomProperty))
		}
	}

	return props
}

func applyKnowledgeSourceListFilters(db *gorm.DB, opts *models.KnowledgeSourceListOptions) *gorm.DB {
	if opts == nil {
		return db
	}

	if opts.Name != nil && *opts.Name != "" {
		db = db.Where("name LIKE ?", "%"+*opts.Name+"%")
	}

	if opts.ExternalID != nil && *opts.ExternalID != "" {
		db = db.Where("external_id = ?", *opts.ExternalID)
	}

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

package models

import (
	"github.com/kubeflow/model-registry/internal/db/filter"
	"github.com/kubeflow/model-registry/internal/db/models"
)

// RestEntityKnowledgeSource is the filter.RestEntityType constant for KnowledgeSource entities.
const RestEntityKnowledgeSource filter.RestEntityType = "KnowledgeSource"

// KnowledgeSourceAttributes contains the attributes for a KnowledgeSource entity.
type KnowledgeSourceAttributes struct {
	Name                     *string
	ExternalID               *string
	CreateTimeSinceEpoch     *int64
	LastUpdateTimeSinceEpoch *int64
	SourceType               *string
	Location                 *string
	ContentType              *string
	Provider                 *string
	Status                   *string
	DocumentCount            *int32
	VectorDimensions         *int32
	IndexType                *string
}

// KnowledgeSource is the interface for KnowledgeSource entities.
// It extends the shared Entity interface with KnowledgeSource-specific attributes.
type KnowledgeSource interface {
	models.Entity[KnowledgeSourceAttributes]
}

// KnowledgeSourceImpl is the concrete implementation of the KnowledgeSource interface.
// It uses the shared BaseEntity implementation.
type KnowledgeSourceImpl = models.BaseEntity[KnowledgeSourceAttributes]

// NewKnowledgeSource creates a new KnowledgeSource entity.
func NewKnowledgeSource(attrs *KnowledgeSourceAttributes) KnowledgeSource {
	return &KnowledgeSourceImpl{
		Attributes: attrs,
	}
}

// KnowledgeSourceListOptions contains options for listing KnowledgeSource entities.
type KnowledgeSourceListOptions struct {
	models.Pagination
	Name       *string
	ExternalID *string
	SourceIDs  *[]string
	Query      *string
}

// GetRestEntityType implements the FilterApplier interface for advanced filtering.
func (o *KnowledgeSourceListOptions) GetRestEntityType() filter.RestEntityType {
	return RestEntityKnowledgeSource
}

// KnowledgeSourceRepository is the interface for KnowledgeSource data access.
type KnowledgeSourceRepository interface {
	GetByID(id int32) (KnowledgeSource, error)
	GetByName(name string) (KnowledgeSource, error)
	List(options KnowledgeSourceListOptions) (*models.ListWrapper[KnowledgeSource], error)
	Save(entity KnowledgeSource) (KnowledgeSource, error)
	DeleteBySource(sourceID string) error
	DeleteByID(id int32) error
	GetDistinctSourceIDs() ([]string, error)
	CountBySource(sourceID string) (int, error)
}

package catalog

import (
	"context"

	"github.com/golang/glog"
	"github.com/kubeflow/model-registry/pkg/catalog"
	sharedmodels "github.com/kubeflow/model-registry/internal/db/models"
	"github.com/kubeflow/model-registry/catalog/plugins/knowledge/internal/db/models"
	"github.com/kubeflow/model-registry/catalog/plugins/knowledge/internal/db/service"
)

// glogLogger implements catalog.LoaderLogger using glog.
type glogLogger struct{}

func (glogLogger) Infof(format string, args ...any)  { glog.Infof(format, args...) }
func (glogLogger) Errorf(format string, args ...any) { glog.Errorf(format, args...) }

// Loader wraps the generic catalog loader with KnowledgeSource-specific types.
type Loader struct {
	*catalog.Loader[models.KnowledgeSource, any]
	services service.Services
}

// NewLoader creates a new catalog loader.
func NewLoader(services service.Services, paths []string, registry *catalog.ProviderRegistry[models.KnowledgeSource, any]) *Loader {
	cfg := catalog.LoaderConfig[models.KnowledgeSource, any]{
		Paths:            paths,
		ProviderRegistry: registry,
		Logger:           glogLogger{},
		SaveEntity: func(entity models.KnowledgeSource) (models.KnowledgeSource, error) {
			return services.KnowledgeSourceRepository.Save(entity)
		},
		SaveArtifact: func(artifact any, entityID int32) error {
			return nil // No artifacts configured
		},
		GetEntityID: func(entity models.KnowledgeSource) *int32 {
			return entity.GetID()
		},
		GetEntityName: func(entity models.KnowledgeSource) string {
			if attrs := entity.GetAttributes(); attrs != nil && attrs.Name != nil {
				return *attrs.Name
			}
			return ""
		},
		DeleteArtifactsByEntity: func(entityID int32) error {
			return nil
		},
		DeleteEntitiesBySource: func(sourceID string) error {
			return services.KnowledgeSourceRepository.DeleteBySource(sourceID)
		},
		GetDistinctSourceIDs: func() ([]string, error) {
			return services.KnowledgeSourceRepository.GetDistinctSourceIDs()
		},
		SetEntitySourceID: func(entity models.KnowledgeSource, sourceID string) {
			setEntitySourceID(entity, sourceID)
		},
		IsEntityNil: func(entity models.KnowledgeSource) bool {
			return entity == nil
		},
	}

	return &Loader{
		Loader:   catalog.NewLoader(cfg),
		services: services,
	}
}

// Start begins loading catalog data.
func (l *Loader) Start(ctx context.Context) error {
	return l.Loader.Start(ctx)
}

// setEntitySourceID sets the source_id as a property on the entity.
func setEntitySourceID(entity models.KnowledgeSource, sourceID string) {
	props := entity.GetProperties()
	if props == nil {
		newProps := []sharedmodels.Properties{}
		props = &newProps
	}

	for i := range *props {
		if (*props)[i].Name == "source_id" {
			(*props)[i].StringValue = &sourceID
			return
		}
	}

	*props = append(*props, sharedmodels.Properties{
		Name:        "source_id",
		StringValue: &sourceID,
	})
}

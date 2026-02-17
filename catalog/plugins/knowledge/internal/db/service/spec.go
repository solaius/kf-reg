package service

import (
	"github.com/kubeflow/model-registry/catalog/plugins/knowledge/internal/db/models"
	"github.com/kubeflow/model-registry/internal/datastore"
)

const (
	KnowledgeSourceTypeName = "kf.KnowledgeSource"
)

// DatastoreSpec returns the datastore specification for this catalog.
func DatastoreSpec() *datastore.Spec {
	return datastore.NewSpec().
		AddContext(KnowledgeSourceTypeName, datastore.NewSpecType(NewKnowledgeSourceRepository).
			AddString("source_id").
			AddString("sourceType").
			AddString("location").
			AddString("contentType").
			AddString("provider").
			AddString("status").
			AddInt("documentCount").
			AddInt("vectorDimensions").
			AddString("indexType"),
		)
}

// Services holds all repository instances for this catalog.
type Services struct {
	KnowledgeSourceRepository models.KnowledgeSourceRepository
}

// NewServices creates a new Services instance.
func NewServices(
	knowledgesourceRepository models.KnowledgeSourceRepository,
) Services {
	return Services{
		KnowledgeSourceRepository: knowledgesourceRepository,
	}
}

package repositories

import (
	"log/slog"
)

type ModelCatalogClientInterface interface {
	CatalogSourcesInterface
	CatalogModelsInterface
	CatalogSourcePreviewInterface
	CatalogPluginsInterface
	CatalogManagementInterface
	McpCatalogInterface
	CatalogEntitiesInterface
}

type ModelCatalogClient struct {
	logger *slog.Logger
	CatalogSources
	CatalogModels
	CatalogSourcePreview
	CatalogPlugins
	CatalogManagement
	McpCatalog
	CatalogEntities
}

func NewModelCatalogClient(logger *slog.Logger) (ModelCatalogClientInterface, error) {
	return &ModelCatalogClient{logger: logger}, nil
}

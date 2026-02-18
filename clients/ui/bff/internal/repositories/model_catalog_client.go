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
	CatalogGovernanceInterface
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
	CatalogGovernance
}

func NewModelCatalogClient(logger *slog.Logger) (ModelCatalogClientInterface, error) {
	return &ModelCatalogClient{logger: logger}, nil
}

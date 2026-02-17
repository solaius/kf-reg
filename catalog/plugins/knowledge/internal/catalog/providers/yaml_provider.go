package providers

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"github.com/kubeflow/model-registry/catalog/plugins/knowledge/internal/db/models"
	sharedmodels "github.com/kubeflow/model-registry/internal/db/models"
	pkgcatalog "github.com/kubeflow/model-registry/pkg/catalog"
	yamlprovider "github.com/kubeflow/model-registry/pkg/catalog/providers/yaml"
)

// yamlKnowledgeSource represents a KnowledgeSource entry in the YAML catalog file.
type yamlKnowledgeSource struct {
	Name             string         `json:"name" yaml:"name"`
	ExternalId       string         `json:"externalId,omitempty" yaml:"externalId,omitempty"`
	Description      *string        `json:"description,omitempty" yaml:"description,omitempty"`
	SourceType       *string        `json:"sourceType,omitempty" yaml:"sourceType,omitempty"`
	Location         *string        `json:"location,omitempty" yaml:"location,omitempty"`
	ContentType      *string        `json:"contentType,omitempty" yaml:"contentType,omitempty"`
	Provider         *string        `json:"provider,omitempty" yaml:"provider,omitempty"`
	Status           *string        `json:"status,omitempty" yaml:"status,omitempty"`
	DocumentCount    *int32         `json:"documentCount,omitempty" yaml:"documentCount,omitempty"`
	VectorDimensions *int32         `json:"vectorDimensions,omitempty" yaml:"vectorDimensions,omitempty"`
	IndexType        *string        `json:"indexType,omitempty" yaml:"indexType,omitempty"`
	CustomProperties map[string]any `json:"customProperties,omitempty" yaml:"customProperties,omitempty"`
}

// yamlKnowledgeSourceCatalog is the structure of the YAML catalog file.
type yamlKnowledgeSourceCatalog struct {
	KnowledgeSources []yamlKnowledgeSource `json:"knowledgesources" yaml:"knowledgesources"`
}

// glogLogger implements yaml.Logger using glog.
type glogLogger struct{}

func (glogLogger) Infof(format string, args ...any)  { glog.Infof(format, args...) }
func (glogLogger) Errorf(format string, args ...any) { glog.Errorf(format, args...) }

// NewKnowledgeSourceYAMLProvider creates a new YAML provider for KnowledgeSource entities.
func NewKnowledgeSourceYAMLProvider() pkgcatalog.ProviderFunc[models.KnowledgeSource, any] {
	return func(ctx context.Context, source *pkgcatalog.Source, reldir string) (<-chan pkgcatalog.Record[models.KnowledgeSource, any], error) {
		config := yamlprovider.Config[models.KnowledgeSource, any]{
			Parse: func(data []byte) ([]pkgcatalog.Record[models.KnowledgeSource, any], error) {
				return parseKnowledgeSourceCatalog(data)
			},
			Logger: glogLogger{},
		}

		provider, err := yamlprovider.NewProvider(config, source, reldir)
		if err != nil {
			return nil, err
		}
		return provider.Records(ctx)
	}
}

// validateKnowledgeSource checks required fields.
func validateKnowledgeSource(item yamlKnowledgeSource, index int) error {
	if item.Name == "" {
		return fmt.Errorf("entry %d: field 'name' is required", index)
	}
	return nil
}

// parseKnowledgeSourceCatalog parses the YAML catalog file into records.
func parseKnowledgeSourceCatalog(catalogData []byte) ([]pkgcatalog.Record[models.KnowledgeSource, any], error) {
	var entityCatalog yamlKnowledgeSourceCatalog
	if err := k8syaml.UnmarshalStrict(catalogData, &entityCatalog); err != nil {
		return nil, fmt.Errorf("failed to parse knowledge source catalog YAML: %w", err)
	}

	for i, item := range entityCatalog.KnowledgeSources {
		if err := validateKnowledgeSource(item, i); err != nil {
			return nil, err
		}
	}

	records := make([]pkgcatalog.Record[models.KnowledgeSource, any], 0, len(entityCatalog.KnowledgeSources))
	for _, item := range entityCatalog.KnowledgeSources {
		name := item.Name
		var externalID *string
		if item.ExternalId != "" {
			externalID = &item.ExternalId
		}
		entity := models.NewKnowledgeSource(&models.KnowledgeSourceAttributes{
			Name:       &name,
			ExternalID: externalID,
		})

		var props []sharedmodels.Properties
		if item.Description != nil {
			props = append(props, sharedmodels.NewStringProperty("description", *item.Description, false))
		}
		if item.SourceType != nil {
			props = append(props, sharedmodels.NewStringProperty("sourceType", *item.SourceType, false))
		}
		if item.Location != nil {
			props = append(props, sharedmodels.NewStringProperty("location", *item.Location, false))
		}
		if item.ContentType != nil {
			props = append(props, sharedmodels.NewStringProperty("contentType", *item.ContentType, false))
		}
		if item.Provider != nil {
			props = append(props, sharedmodels.NewStringProperty("provider", *item.Provider, false))
		}
		if item.Status != nil {
			props = append(props, sharedmodels.NewStringProperty("status", *item.Status, false))
		}
		if item.DocumentCount != nil {
			props = append(props, sharedmodels.NewIntProperty("documentCount", *item.DocumentCount, false))
		}
		if item.VectorDimensions != nil {
			props = append(props, sharedmodels.NewIntProperty("vectorDimensions", *item.VectorDimensions, false))
		}
		if item.IndexType != nil {
			props = append(props, sharedmodels.NewStringProperty("indexType", *item.IndexType, false))
		}
		if len(props) > 0 {
			entity.(*models.KnowledgeSourceImpl).Properties = &props
		}

		// Set custom properties
		if len(item.CustomProperties) > 0 {
			var customProps []sharedmodels.Properties
			for k, v := range item.CustomProperties {
				switch val := v.(type) {
				case string:
					customProps = append(customProps, sharedmodels.NewStringProperty(k, val, true))
				case float64:
					customProps = append(customProps, sharedmodels.NewDoubleProperty(k, val, true))
				case bool:
					customProps = append(customProps, sharedmodels.NewBoolProperty(k, val, true))
				default:
					customProps = append(customProps, sharedmodels.NewStringProperty(k, fmt.Sprintf("%v", val), true))
				}
			}
			entity.(*models.KnowledgeSourceImpl).CustomProperties = &customProps
		}

		record := pkgcatalog.Record[models.KnowledgeSource, any]{Entity: entity}
		records = append(records, record)
	}

	return records, nil
}

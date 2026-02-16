package providers

import (
	"context"
	"os"
	"path/filepath"

	"fmt"

	"github.com/golang/glog"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"github.com/kubeflow/model-registry/catalog/plugins/mcp/internal/db/models"
	sharedmodels "github.com/kubeflow/model-registry/internal/db/models"
	pkgcatalog "github.com/kubeflow/model-registry/pkg/catalog"
	yamlprovider "github.com/kubeflow/model-registry/pkg/catalog/providers/yaml"
)

// yamlMcpServer represents a McpServer entry in the YAML catalog file.
type yamlMcpServer struct {
	Name                string         `json:"name" yaml:"name"`
	ExternalId          string         `json:"externalId,omitempty" yaml:"externalId,omitempty"`
	Description         *string        `json:"description,omitempty" yaml:"description,omitempty"`
	ServerUrl           string         `json:"serverUrl" yaml:"serverUrl"`
	TransportType       *string        `json:"transportType,omitempty" yaml:"transportType,omitempty"`
	ToolCount           *int32         `json:"toolCount,omitempty" yaml:"toolCount,omitempty"`
	ResourceCount       *int32         `json:"resourceCount,omitempty" yaml:"resourceCount,omitempty"`
	PromptCount         *int32         `json:"promptCount,omitempty" yaml:"promptCount,omitempty"`
	DeploymentMode      *string        `json:"deploymentMode,omitempty" yaml:"deploymentMode,omitempty"`
	Image               *string        `json:"image,omitempty" yaml:"image,omitempty"`
	Endpoint            *string        `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	SupportedTransports *string        `json:"supportedTransports,omitempty" yaml:"supportedTransports,omitempty"`
	License             *string        `json:"license,omitempty" yaml:"license,omitempty"`
	Verified            *bool          `json:"verified,omitempty" yaml:"verified,omitempty"`
	Certified           *bool          `json:"certified,omitempty" yaml:"certified,omitempty"`
	Provider            *string        `json:"provider,omitempty" yaml:"provider,omitempty"`
	Logo                *string        `json:"logo,omitempty" yaml:"logo,omitempty"`
	Category            *string        `json:"category,omitempty" yaml:"category,omitempty"`
	CustomProperties    map[string]any `json:"customProperties,omitempty" yaml:"customProperties,omitempty"`
}

// yamlMcpServerCatalog is the structure of the YAML catalog file.
type yamlMcpServerCatalog struct {
	McpServers []yamlMcpServer `json:"mcpservers" yaml:"mcpservers"`
}

// glogLogger implements yaml.Logger using glog.
type glogLogger struct{}

func (glogLogger) Infof(format string, args ...any)  { glog.Infof(format, args...) }
func (glogLogger) Errorf(format string, args ...any) { glog.Errorf(format, args...) }

// NewMcpServerYAMLProvider creates a new YAML provider for McpServer entities.
// It uses the reusable yaml.NewProviderFunc which includes automatic hot-reload
// via file watching (polling every 5 seconds for file changes).
func NewMcpServerYAMLProvider() pkgcatalog.ProviderFunc[models.McpServer, any] {
	return func(ctx context.Context, source *pkgcatalog.Source, reldir string) (<-chan pkgcatalog.Record[models.McpServer, any], error) {
		// Resolve artifacts path from source properties (captured in parse closure)
		var artifactsPath string
		if ap, ok := source.Properties["yamlArtifactsPath"].(string); ok && ap != "" {
			if !filepath.IsAbs(ap) {
				ap = filepath.Join(reldir, ap)
			}
			artifactsPath = ap
		}

		config := yamlprovider.Config[models.McpServer, any]{
			Parse: func(data []byte) ([]pkgcatalog.Record[models.McpServer, any], error) {
				var artifactsData []byte
				if artifactsPath != "" {
					var err error
					artifactsData, err = os.ReadFile(artifactsPath)
					if err != nil {
						glog.Warningf("failed to read artifacts file %s: %v", artifactsPath, err)
					}
				}
				return parseMcpServerCatalog(data, artifactsData)
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

// parseMcpServerCatalog parses the YAML catalog files into records.
func parseMcpServerCatalog(catalogData, artifactsData []byte) ([]pkgcatalog.Record[models.McpServer, any], error) {
	var entityCatalog yamlMcpServerCatalog
	if err := k8syaml.UnmarshalStrict(catalogData, &entityCatalog); err != nil {
		return nil, err
	}

	records := make([]pkgcatalog.Record[models.McpServer, any], 0, len(entityCatalog.McpServers))
	for _, item := range entityCatalog.McpServers {
		name := item.Name
		var externalID *string
		if item.ExternalId != "" {
			externalID = &item.ExternalId
		}
		entity := models.NewMcpServer(&models.McpServerAttributes{
			Name:       &name,
			ExternalID: externalID,
		})

		// Set properties
		var props []sharedmodels.Properties
		if item.Description != nil {
			props = append(props, sharedmodels.NewStringProperty("description", *item.Description, false))
		}
		if item.ServerUrl != "" {
			props = append(props, sharedmodels.NewStringProperty("serverUrl", item.ServerUrl, false))
		}
		if item.TransportType != nil {
			props = append(props, sharedmodels.NewStringProperty("transportType", *item.TransportType, false))
		}
		if item.ToolCount != nil {
			props = append(props, sharedmodels.NewIntProperty("toolCount", *item.ToolCount, false))
		}
		if item.ResourceCount != nil {
			props = append(props, sharedmodels.NewIntProperty("resourceCount", *item.ResourceCount, false))
		}
		if item.PromptCount != nil {
			props = append(props, sharedmodels.NewIntProperty("promptCount", *item.PromptCount, false))
		}
		if item.DeploymentMode != nil {
			props = append(props, sharedmodels.NewStringProperty("deploymentMode", *item.DeploymentMode, false))
		}
		if item.Image != nil {
			props = append(props, sharedmodels.NewStringProperty("image", *item.Image, false))
		}
		if item.Endpoint != nil {
			props = append(props, sharedmodels.NewStringProperty("endpoint", *item.Endpoint, false))
		}
		if item.SupportedTransports != nil {
			props = append(props, sharedmodels.NewStringProperty("supportedTransports", *item.SupportedTransports, false))
		}
		if item.License != nil {
			props = append(props, sharedmodels.NewStringProperty("license", *item.License, false))
		}
		if item.Verified != nil {
			props = append(props, sharedmodels.NewBoolProperty("verified", *item.Verified, false))
		}
		if item.Certified != nil {
			props = append(props, sharedmodels.NewBoolProperty("certified", *item.Certified, false))
		}
		if item.Provider != nil {
			props = append(props, sharedmodels.NewStringProperty("provider", *item.Provider, false))
		}
		if item.Logo != nil {
			props = append(props, sharedmodels.NewStringProperty("logo", *item.Logo, false))
		}
		if item.Category != nil {
			props = append(props, sharedmodels.NewStringProperty("category", *item.Category, false))
		}
		if len(props) > 0 {
			entity.(*models.McpServerImpl).Properties = &props
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
			entity.(*models.McpServerImpl).CustomProperties = &customProps
		}

		record := pkgcatalog.Record[models.McpServer, any]{Entity: entity}

		records = append(records, record)
	}

	return records, nil
}

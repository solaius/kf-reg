package service

import (
	"github.com/kubeflow/model-registry/internal/db/filter"

	"github.com/kubeflow/model-registry/catalog/plugins/knowledge/internal/db/models"
)

func init() {
	// Register KnowledgeSource entity properties in the global filter property map.
	filter.RestEntityPropertyMap[models.RestEntityKnowledgeSource] = map[string]bool{
		// Built-in entity table properties
		"id": true, "name": true, "externalId": true,
		"createTimeSinceEpoch": true, "lastUpdateTimeSinceEpoch": true,
		"sourceType":       true,
		"location":         true,
		"contentType":      true,
		"provider":         true,
		"status":           true,
		"documentCount":    true,
		"vectorDimensions": true,
		"indexType":        true,
	}
}

// entityMappings implements filter.EntityMappingFunctions for KnowledgeSource entities.
type entityMappings struct{}

// GetMLMDEntityType maps a REST entity type to its underlying MLMD entity type.
func (m *entityMappings) GetMLMDEntityType(t filter.RestEntityType) filter.EntityType {
	return filter.EntityTypeContext
}

// GetPropertyDefinitionForRestEntity returns the property definition for a REST entity property.
func (m *entityMappings) GetPropertyDefinitionForRestEntity(t filter.RestEntityType, prop string) filter.PropertyDefinition {
	switch prop {
	case "id":
		return filter.PropertyDefinition{Location: filter.EntityTable, ValueType: "int_value", Column: "id"}
	case "name":
		return filter.PropertyDefinition{Location: filter.EntityTable, ValueType: "string_value", Column: "name"}
	case "externalId":
		return filter.PropertyDefinition{Location: filter.EntityTable, ValueType: "string_value", Column: "external_id"}
	case "createTimeSinceEpoch":
		return filter.PropertyDefinition{Location: filter.EntityTable, ValueType: "int_value", Column: "create_time_since_epoch"}
	case "lastUpdateTimeSinceEpoch":
		return filter.PropertyDefinition{Location: filter.EntityTable, ValueType: "int_value", Column: "last_update_time_since_epoch"}
	case "sourceType":
		return filter.PropertyDefinition{Location: filter.PropertyTable, ValueType: "string_value", Column: "sourceType"}
	case "location":
		return filter.PropertyDefinition{Location: filter.PropertyTable, ValueType: "string_value", Column: "location"}
	case "contentType":
		return filter.PropertyDefinition{Location: filter.PropertyTable, ValueType: "string_value", Column: "contentType"}
	case "provider":
		return filter.PropertyDefinition{Location: filter.PropertyTable, ValueType: "string_value", Column: "provider"}
	case "status":
		return filter.PropertyDefinition{Location: filter.PropertyTable, ValueType: "string_value", Column: "status"}
	case "documentCount":
		return filter.PropertyDefinition{Location: filter.PropertyTable, ValueType: "int_value", Column: "documentCount"}
	case "vectorDimensions":
		return filter.PropertyDefinition{Location: filter.PropertyTable, ValueType: "int_value", Column: "vectorDimensions"}
	case "indexType":
		return filter.PropertyDefinition{Location: filter.PropertyTable, ValueType: "string_value", Column: "indexType"}

	default:
		return filter.PropertyDefinition{
			Location:  filter.Custom,
			ValueType: "string_value",
			Column:    prop,
		}
	}
}

// IsChildEntity returns true if the REST entity type uses prefixed names.
func (m *entityMappings) IsChildEntity(t filter.RestEntityType) bool {
	return false
}

package plugin

import "context"

// AssetMapper converts plugin-specific entities into the universal AssetResource
// envelope. Each plugin provides its own mapper implementation that knows how
// to extract standard metadata and flatten entity-specific fields into Spec.
type AssetMapper interface {
	// MapToAsset converts a single plugin-specific entity (passed as any)
	// into an AssetResource.
	MapToAsset(entity any) (AssetResource, error)

	// MapToAssets converts a slice of plugin-specific entities into
	// AssetResource items. This is a batch variant of MapToAsset for
	// list operations.
	MapToAssets(entities []any) ([]AssetResource, error)

	// SupportedKinds returns the entity kinds this mapper handles.
	SupportedKinds() []string
}

// AssetMapperProvider is an optional interface that plugins can implement
// to provide an AssetMapper for universal entity projection.
type AssetMapperProvider interface {
	GetAssetMapper() AssetMapper
}

// AssetLister is an optional interface that plugins can implement to support
// listing entities as AssetResource items directly. This is used by the
// generic /api/assets endpoint to provide a unified cross-plugin listing.
type AssetLister interface {
	// ListAssets returns a paginated list of entities as AssetResource items.
	ListAssets(ctx context.Context, opts AssetListOptions) (*AssetList, error)
}

// AssetGetter is an optional interface that plugins can implement to support
// getting a single entity as an AssetResource directly.
type AssetGetter interface {
	// GetAsset returns a single entity as an AssetResource by kind and name.
	GetAsset(ctx context.Context, kind string, name string) (*AssetResource, error)
}

// AssetListOptions specifies options for listing assets.
type AssetListOptions struct {
	// Kind filters to a specific entity kind (empty means all kinds).
	Kind string `json:"kind,omitempty"`

	// PageSize is the maximum number of items to return.
	PageSize int `json:"pageSize,omitempty"`

	// PageToken is the opaque token for pagination.
	PageToken string `json:"pageToken,omitempty"`

	// FilterQuery is an SQL-like filter expression.
	FilterQuery string `json:"filterQuery,omitempty"`

	// OrderBy is the field to sort by.
	OrderBy string `json:"orderBy,omitempty"`

	// SortOrder is "ASC" or "DESC".
	SortOrder string `json:"sortOrder,omitempty"`

	// SourceID filters to a specific source.
	SourceID string `json:"sourceId,omitempty"`
}

// MapToAssetsBatch is a helper that converts a slice of entities by calling
// mapFn for each one. Plugin mappers can use this to implement MapToAssets
// without duplicating the loop and error-handling boilerplate.
func MapToAssetsBatch(entities []any, mapFn func(any) (AssetResource, error)) ([]AssetResource, error) {
	result := make([]AssetResource, 0, len(entities))
	for _, entity := range entities {
		asset, err := mapFn(entity)
		if err != nil {
			return nil, err
		}
		result = append(result, asset)
	}
	return result, nil
}

// DefaultAssetStatus returns an AssetStatus with lifecycle "active" and health "unknown".
func DefaultAssetStatus() AssetStatus {
	return AssetStatus{
		Lifecycle: LifecycleActive,
		Health:    HealthUnknown,
	}
}

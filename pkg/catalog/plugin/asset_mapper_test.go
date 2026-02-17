package plugin

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAssetStatus(t *testing.T) {
	tests := []struct {
		name              string
		wantLifecycle     LifecycleStatus
		wantHealth        HealthStatus
		wantConditionsNil bool
		wantLinksNil      bool
	}{
		{
			name:              "returns active lifecycle and unknown health",
			wantLifecycle:     LifecycleActive,
			wantHealth:        HealthUnknown,
			wantConditionsNil: true,
			wantLinksNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := DefaultAssetStatus()
			assert.Equal(t, tt.wantLifecycle, s.Lifecycle)
			assert.Equal(t, tt.wantHealth, s.Health)
			if tt.wantConditionsNil {
				assert.Nil(t, s.Conditions)
			}
			if tt.wantLinksNil {
				assert.Nil(t, s.Links)
			}
		})
	}
}

func TestLifecycleStatusValues(t *testing.T) {
	tests := []struct {
		constant LifecycleStatus
		want     string
	}{
		{LifecycleActive, "active"},
		{LifecycleDeprecated, "deprecated"},
		{LifecycleRetired, "retired"},
		{LifecycleDraft, "draft"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.constant))
		})
	}
}

func TestHealthStatusValues(t *testing.T) {
	tests := []struct {
		constant HealthStatus
		want     string
	}{
		{HealthUnknown, "unknown"},
		{HealthHealthy, "healthy"},
		{HealthDegraded, "degraded"},
		{HealthUnhealthy, "unhealthy"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.constant))
		})
	}
}

func TestAssetResourceJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		asset AssetResource
	}{
		{
			name: "minimal asset",
			asset: AssetResource{
				APIVersion: "catalog/v1alpha1",
				Kind:       "McpServer",
				Metadata: AssetMetadata{
					UID:  "abc-123",
					Name: "my-server",
				},
				Spec:   map[string]any{"endpoint": "https://example.com"},
				Status: DefaultAssetStatus(),
			},
		},
		{
			name: "fully populated asset",
			asset: AssetResource{
				APIVersion: "catalog/v1alpha1",
				Kind:       "CatalogModel",
				Metadata: AssetMetadata{
					UID:         "def-456",
					Name:        "llama-3",
					DisplayName: "Llama 3",
					Description: "A large language model",
					Labels:      map[string]string{"vendor": "meta"},
					Annotations: map[string]string{"internal": "true"},
					Tags:        []string{"llm", "text-generation"},
					CreatedAt:   "2025-01-01T00:00:00Z",
					UpdatedAt:   "2025-06-01T00:00:00Z",
					Owner: &AssetOwner{
						Name:  "AI Team",
						Email: "ai@example.com",
						Team:  "platform",
					},
					SourceRef: &SourceRef{
						SourceID:   "hf-main",
						SourceName: "Hugging Face",
						SourceType: "hf",
					},
				},
				Spec: map[string]any{
					"task":       "text-generation",
					"parameters": float64(70e9),
				},
				Status: AssetStatus{
					Lifecycle: LifecycleActive,
					Health:    HealthHealthy,
					Conditions: []StatusCondition{
						{
							Type:    "Ready",
							Status:  "True",
							Reason:  "Synced",
							Message: "Successfully synced from source",
						},
					},
					Links: &AssetLinks{
						Related: []LinkRef{
							{Kind: "Artifact", Name: "llama-3-weights", UID: "art-789"},
						},
					},
				},
			},
		},
		{
			name: "asset with nil optional fields",
			asset: AssetResource{
				APIVersion: "catalog/v1alpha1",
				Kind:       "Dataset",
				Metadata: AssetMetadata{
					UID:  "ghi-789",
					Name: "squad-v2",
				},
				Spec:   map[string]any{},
				Status: DefaultAssetStatus(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.asset)
			require.NoError(t, err)

			var got AssetResource
			err = json.Unmarshal(data, &got)
			require.NoError(t, err)

			assert.Equal(t, tt.asset.APIVersion, got.APIVersion)
			assert.Equal(t, tt.asset.Kind, got.Kind)
			assert.Equal(t, tt.asset.Metadata.UID, got.Metadata.UID)
			assert.Equal(t, tt.asset.Metadata.Name, got.Metadata.Name)
			assert.Equal(t, tt.asset.Metadata.DisplayName, got.Metadata.DisplayName)
			assert.Equal(t, tt.asset.Metadata.Description, got.Metadata.Description)
			assert.Equal(t, tt.asset.Status.Lifecycle, got.Status.Lifecycle)
			assert.Equal(t, tt.asset.Status.Health, got.Status.Health)

			if tt.asset.Metadata.Owner != nil {
				require.NotNil(t, got.Metadata.Owner)
				assert.Equal(t, tt.asset.Metadata.Owner.Name, got.Metadata.Owner.Name)
				assert.Equal(t, tt.asset.Metadata.Owner.Email, got.Metadata.Owner.Email)
			}

			if tt.asset.Metadata.SourceRef != nil {
				require.NotNil(t, got.Metadata.SourceRef)
				assert.Equal(t, tt.asset.Metadata.SourceRef.SourceID, got.Metadata.SourceRef.SourceID)
			}

			if tt.asset.Metadata.Tags != nil {
				assert.Equal(t, tt.asset.Metadata.Tags, got.Metadata.Tags)
			}

			if tt.asset.Status.Links != nil {
				require.NotNil(t, got.Status.Links)
				assert.Equal(t, len(tt.asset.Status.Links.Related), len(got.Status.Links.Related))
			}
		})
	}
}

// testAssetMapper is a test implementation of AssetMapper.
type testAssetMapper struct{}

func (m *testAssetMapper) MapToAsset(entity any) (AssetResource, error) {
	name, ok := entity.(string)
	if !ok {
		return AssetResource{}, fmt.Errorf("expected string, got %T", entity)
	}
	return AssetResource{
		APIVersion: "catalog/v1alpha1",
		Kind:       "TestEntity",
		Metadata: AssetMetadata{
			UID:  "test-" + name,
			Name: name,
		},
		Spec:   map[string]any{"original": entity},
		Status: DefaultAssetStatus(),
	}, nil
}

func (m *testAssetMapper) MapToAssets(entities []any) ([]AssetResource, error) {
	return MapToAssetsBatch(entities, m.MapToAsset)
}

func (m *testAssetMapper) SupportedKinds() []string {
	return []string{"TestEntity"}
}

// Verify that testAssetMapper satisfies the AssetMapper interface.
var _ AssetMapper = (*testAssetMapper)(nil)

func TestAssetMapperMock(t *testing.T) {
	mapper := &testAssetMapper{}

	t.Run("MapToAsset produces correct envelope", func(t *testing.T) {
		asset, err := mapper.MapToAsset("my-entity")
		require.NoError(t, err)

		assert.Equal(t, "catalog/v1alpha1", asset.APIVersion)
		assert.Equal(t, "TestEntity", asset.Kind)
		assert.Equal(t, "test-my-entity", asset.Metadata.UID)
		assert.Equal(t, "my-entity", asset.Metadata.Name)
		assert.Equal(t, "my-entity", asset.Spec["original"])
		assert.Equal(t, LifecycleActive, asset.Status.Lifecycle)
		assert.Equal(t, HealthUnknown, asset.Status.Health)
	})

	t.Run("MapToAsset returns error for invalid input", func(t *testing.T) {
		_, err := mapper.MapToAsset(42)
		assert.Error(t, err)
	})

	t.Run("MapToAssets maps multiple entities", func(t *testing.T) {
		entities := []any{"alpha", "beta", "gamma"}
		assets, err := mapper.MapToAssets(entities)
		require.NoError(t, err)

		require.Len(t, assets, 3)
		assert.Equal(t, "alpha", assets[0].Metadata.Name)
		assert.Equal(t, "beta", assets[1].Metadata.Name)
		assert.Equal(t, "gamma", assets[2].Metadata.Name)
	})

	t.Run("MapToAssets returns error if any entity fails", func(t *testing.T) {
		entities := []any{"valid", 42, "also-valid"}
		_, err := mapper.MapToAssets(entities)
		assert.Error(t, err)
	})

	t.Run("SupportedKinds returns expected kinds", func(t *testing.T) {
		kinds := mapper.SupportedKinds()
		assert.Equal(t, []string{"TestEntity"}, kinds)
	})
}

func TestMapToAssetsBatch(t *testing.T) {
	mapFn := func(entity any) (AssetResource, error) {
		s, ok := entity.(string)
		if !ok {
			return AssetResource{}, fmt.Errorf("not a string")
		}
		return AssetResource{
			APIVersion: "catalog/v1alpha1",
			Kind:       "Item",
			Metadata:   AssetMetadata{Name: s},
			Status:     DefaultAssetStatus(),
		}, nil
	}

	t.Run("empty slice", func(t *testing.T) {
		result, err := MapToAssetsBatch(nil, mapFn)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("successful batch", func(t *testing.T) {
		result, err := MapToAssetsBatch([]any{"a", "b"}, mapFn)
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "a", result[0].Metadata.Name)
		assert.Equal(t, "b", result[1].Metadata.Name)
	})

	t.Run("error propagation", func(t *testing.T) {
		_, err := MapToAssetsBatch([]any{"ok", 123}, mapFn)
		assert.Error(t, err)
	})
}

func TestAssetMapperProviderInterface(t *testing.T) {
	// Verify that a struct implementing GetAssetMapper satisfies AssetMapperProvider.
	mapper := &testAssetMapper{}

	provider := &testMapperProvider{mapper: mapper}
	var p AssetMapperProvider = provider

	m := p.GetAssetMapper()
	asset, err := m.MapToAsset("test")
	require.NoError(t, err)
	assert.Equal(t, "test", asset.Metadata.Name)
}

// testMapperProvider is a test implementation of AssetMapperProvider.
type testMapperProvider struct {
	mapper AssetMapper
}

func (p *testMapperProvider) GetAssetMapper() AssetMapper {
	return p.mapper
}

var _ AssetMapperProvider = (*testMapperProvider)(nil)

func TestAssetListJSONRoundTrip(t *testing.T) {
	list := AssetList{
		APIVersion: "catalog/v1alpha1",
		Kind:       "AssetList",
		Items: []AssetResource{
			{
				APIVersion: "catalog/v1alpha1",
				Kind:       "McpServer",
				Metadata:   AssetMetadata{UID: "1", Name: "server-a"},
				Spec:       map[string]any{"protocol": "stdio"},
				Status:     DefaultAssetStatus(),
			},
			{
				APIVersion: "catalog/v1alpha1",
				Kind:       "McpServer",
				Metadata:   AssetMetadata{UID: "2", Name: "server-b"},
				Spec:       map[string]any{"protocol": "sse"},
				Status:     DefaultAssetStatus(),
			},
		},
		NextPageToken: "token-abc",
		TotalSize:     10,
	}

	data, err := json.Marshal(list)
	require.NoError(t, err)

	var got AssetList
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, "catalog/v1alpha1", got.APIVersion)
	assert.Equal(t, "AssetList", got.Kind)
	require.Len(t, got.Items, 2)
	assert.Equal(t, "server-a", got.Items[0].Metadata.Name)
	assert.Equal(t, "server-b", got.Items[1].Metadata.Name)
	assert.Equal(t, "token-abc", got.NextPageToken)
	assert.Equal(t, 10, got.TotalSize)
}

func TestAssetListOptionsJSONRoundTrip(t *testing.T) {
	opts := AssetListOptions{
		Kind:        "McpServer",
		PageSize:    25,
		PageToken:   "abc",
		FilterQuery: "name LIKE '%test%'",
		OrderBy:     "name",
		SortOrder:   "ASC",
		SourceID:    "src-1",
	}

	data, err := json.Marshal(opts)
	require.NoError(t, err)

	var got AssetListOptions
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, opts, got)
}

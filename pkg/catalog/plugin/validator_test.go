package plugin

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLParseLayer(t *testing.T) {
	layer := YAMLParseLayer()
	ctx := context.Background()

	assert.Equal(t, "yaml_parse", layer.Name)
	assert.True(t, layer.Critical)

	tests := []struct {
		name      string
		input     SourceConfigInput
		wantErrs  int
		wantField string
	}{
		{
			name: "valid YAML",
			input: SourceConfigInput{
				Properties: map[string]any{
					"content": "id: test\nname: Test Source\n",
				},
			},
			wantErrs: 0,
		},
		{
			name: "invalid YAML syntax",
			input: SourceConfigInput{
				Properties: map[string]any{
					"content": "not: [valid: yaml: {{",
				},
			},
			wantErrs:  1,
			wantField: "properties.content",
		},
		{
			name: "no content field",
			input: SourceConfigInput{
				Properties: map[string]any{
					"other": "value",
				},
			},
			wantErrs: 0,
		},
		{
			name: "nil properties",
			input: SourceConfigInput{
				Properties: nil,
			},
			wantErrs: 0,
		},
		{
			name: "empty content string",
			input: SourceConfigInput{
				Properties: map[string]any{
					"content": "",
				},
			},
			wantErrs: 0,
		},
		{
			name: "content is not a string",
			input: SourceConfigInput{
				Properties: map[string]any{
					"content": 42,
				},
			},
			wantErrs: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := layer.Check(ctx, tc.input)
			assert.Len(t, errs, tc.wantErrs)
			if tc.wantErrs > 0 && tc.wantField != "" {
				assert.Equal(t, tc.wantField, errs[0].Field)
				assert.Contains(t, errs[0].Message, "YAML parse error")
			}
		})
	}
}

func TestStrictFieldsLayer(t *testing.T) {
	layer := StrictFieldsLayer()
	ctx := context.Background()

	assert.Equal(t, "strict_fields", layer.Name)
	assert.False(t, layer.Critical)

	tests := []struct {
		name     string
		input    SourceConfigInput
		wantErrs int
	}{
		{
			name: "valid known fields",
			input: SourceConfigInput{
				Properties: map[string]any{
					"content": "id: test\nname: Test\ntype: yaml\n",
				},
			},
			wantErrs: 0,
		},
		{
			name: "unknown field",
			input: SourceConfigInput{
				Properties: map[string]any{
					"content": "id: test\nname: Test\nunknownField: bad\n",
				},
			},
			wantErrs: 1,
		},
		{
			name: "no content field",
			input: SourceConfigInput{
				Properties: map[string]any{
					"other": "value",
				},
			},
			wantErrs: 0,
		},
		{
			name: "nil properties",
			input: SourceConfigInput{
				Properties: nil,
			},
			wantErrs: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := layer.Check(ctx, tc.input)
			assert.Len(t, errs, tc.wantErrs)
			if tc.wantErrs > 0 {
				assert.Equal(t, "properties.content", errs[0].Field)
				assert.Contains(t, errs[0].Message, "unknown or invalid fields")
			}
		})
	}
}

func TestSemanticLayer(t *testing.T) {
	layer := SemanticLayer()
	ctx := context.Background()

	assert.Equal(t, "semantic", layer.Name)
	assert.False(t, layer.Critical)

	tests := []struct {
		name      string
		input     SourceConfigInput
		wantErrs  int
		wantField string
		wantMsg   string
	}{
		{
			name:     "complete valid input",
			input:    SourceConfigInput{ID: "my-source", Name: "My Source", Type: "yaml"},
			wantErrs: 0,
		},
		{
			name:      "missing ID",
			input:     SourceConfigInput{Name: "My Source", Type: "yaml"},
			wantErrs:  1,
			wantField: "id",
			wantMsg:   "source ID is required",
		},
		{
			name:      "missing name",
			input:     SourceConfigInput{ID: "my-source", Type: "yaml"},
			wantErrs:  1,
			wantField: "name",
			wantMsg:   "source name is required",
		},
		{
			name:      "missing type",
			input:     SourceConfigInput{ID: "my-source", Name: "My Source"},
			wantErrs:  1,
			wantField: "type",
			wantMsg:   "source type is required",
		},
		{
			name:      "invalid ID with spaces",
			input:     SourceConfigInput{ID: "my source", Name: "My Source", Type: "yaml"},
			wantErrs:  1,
			wantField: "id",
			wantMsg:   "must contain only lowercase",
		},
		{
			name:      "invalid ID with special chars",
			input:     SourceConfigInput{ID: "my@source!", Name: "My Source", Type: "yaml"},
			wantErrs:  1,
			wantField: "id",
			wantMsg:   "must contain only lowercase",
		},
		{
			name:      "invalid ID with uppercase",
			input:     SourceConfigInput{ID: "MySource", Name: "My Source", Type: "yaml"},
			wantErrs:  1,
			wantField: "id",
			wantMsg:   "must contain only lowercase",
		},
		{
			name:      "name too long",
			input:     SourceConfigInput{ID: "my-source", Name: strings.Repeat("a", 257), Type: "yaml"},
			wantErrs:  1,
			wantField: "name",
			wantMsg:   "256 characters or fewer",
		},
		{
			name:     "valid ID with hyphens",
			input:    SourceConfigInput{ID: "my-source-v2", Name: "My Source", Type: "yaml"},
			wantErrs: 0,
		},
		{
			name:     "valid ID with underscores",
			input:    SourceConfigInput{ID: "my_source_v2", Name: "My Source", Type: "yaml"},
			wantErrs: 0,
		},
		{
			name:     "valid ID with digits only",
			input:    SourceConfigInput{ID: "123", Name: "My Source", Type: "yaml"},
			wantErrs: 0,
		},
		{
			name:     "all fields missing",
			input:    SourceConfigInput{},
			wantErrs: 3, // id, name, type
		},
		{
			name:     "name at exactly 256 chars",
			input:    SourceConfigInput{ID: "ok", Name: strings.Repeat("x", 256), Type: "yaml"},
			wantErrs: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := layer.Check(ctx, tc.input)
			assert.Len(t, errs, tc.wantErrs)
			if tc.wantErrs == 1 && tc.wantField != "" {
				assert.Equal(t, tc.wantField, errs[0].Field)
				assert.Contains(t, errs[0].Message, tc.wantMsg)
			}
		})
	}
}

func TestProviderLayer(t *testing.T) {
	ctx := context.Background()

	t.Run("provider returns valid", func(t *testing.T) {
		sm := &mgmtTestPlugin{
			validateFn: func(input SourceConfigInput) (*ValidationResult, error) {
				return &ValidationResult{Valid: true}, nil
			},
		}
		layer := ProviderLayer(sm)
		assert.Equal(t, "provider", layer.Name)

		errs := layer.Check(ctx, SourceConfigInput{ID: "test"})
		assert.Empty(t, errs)
	})

	t.Run("provider returns invalid", func(t *testing.T) {
		sm := &mgmtTestPlugin{
			validateFn: func(input SourceConfigInput) (*ValidationResult, error) {
				return &ValidationResult{
					Valid: false,
					Errors: []ValidationError{
						{Field: "provider.url", Message: "URL not reachable"},
					},
				}, nil
			},
		}
		layer := ProviderLayer(sm)

		errs := layer.Check(ctx, SourceConfigInput{ID: "test"})
		require.Len(t, errs, 1)
		assert.Equal(t, "provider.url", errs[0].Field)
		assert.Equal(t, "URL not reachable", errs[0].Message)
	})

	t.Run("provider returns error", func(t *testing.T) {
		sm := &mgmtTestPlugin{
			validateFn: func(input SourceConfigInput) (*ValidationResult, error) {
				return nil, fmt.Errorf("connection timeout")
			},
		}
		layer := ProviderLayer(sm)

		errs := layer.Check(ctx, SourceConfigInput{ID: "test"})
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Message, "provider validation error")
		assert.Contains(t, errs[0].Message, "connection timeout")
	})

	t.Run("provider returns nil result", func(t *testing.T) {
		sm := &mgmtTestPlugin{
			validateFn: func(input SourceConfigInput) (*ValidationResult, error) {
				return nil, nil
			},
		}
		layer := ProviderLayer(sm)

		errs := layer.Check(ctx, SourceConfigInput{ID: "test"})
		assert.Empty(t, errs)
	})
}

func TestMultiLayerValidator(t *testing.T) {
	ctx := context.Background()

	t.Run("all layers pass", func(t *testing.T) {
		v := NewMultiLayerValidator()
		v.AddLayer(ValidationLayer{
			Name:  "layer1",
			Check: func(_ context.Context, _ SourceConfigInput) []ValidationError { return nil },
		})
		v.AddLayer(ValidationLayer{
			Name:  "layer2",
			Check: func(_ context.Context, _ SourceConfigInput) []ValidationError { return nil },
		})

		result := v.Validate(ctx, SourceConfigInput{ID: "test"})
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
		assert.Len(t, result.LayerResults, 2)
		assert.True(t, result.LayerResults[0].Valid)
		assert.True(t, result.LayerResults[1].Valid)
	})

	t.Run("critical layer fails stops further layers", func(t *testing.T) {
		layer2Called := false
		v := NewMultiLayerValidator()
		v.AddLayer(ValidationLayer{
			Name:     "critical_layer",
			Critical: true,
			Check: func(_ context.Context, _ SourceConfigInput) []ValidationError {
				return []ValidationError{{Field: "x", Message: "critical failure"}}
			},
		})
		v.AddLayer(ValidationLayer{
			Name: "layer2",
			Check: func(_ context.Context, _ SourceConfigInput) []ValidationError {
				layer2Called = true
				return nil
			},
		})

		result := v.Validate(ctx, SourceConfigInput{})
		assert.False(t, result.Valid)
		require.Len(t, result.Errors, 1)
		assert.Equal(t, "critical failure", result.Errors[0].Message)
		// Only one layer result because critical failure stopped evaluation.
		assert.Len(t, result.LayerResults, 1)
		assert.False(t, layer2Called, "layer after critical failure should not be called")
	})

	t.Run("non-critical failure does not stop", func(t *testing.T) {
		layer2Called := false
		v := NewMultiLayerValidator()
		v.AddLayer(ValidationLayer{
			Name:     "non_critical",
			Critical: false,
			Check: func(_ context.Context, _ SourceConfigInput) []ValidationError {
				return []ValidationError{{Field: "a", Message: "warning"}}
			},
		})
		v.AddLayer(ValidationLayer{
			Name: "layer2",
			Check: func(_ context.Context, _ SourceConfigInput) []ValidationError {
				layer2Called = true
				return []ValidationError{{Field: "b", Message: "another issue"}}
			},
		})

		result := v.Validate(ctx, SourceConfigInput{})
		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 2)
		assert.Len(t, result.LayerResults, 2)
		assert.True(t, layer2Called, "layer after non-critical failure should be called")
		assert.False(t, result.LayerResults[0].Valid)
		assert.False(t, result.LayerResults[1].Valid)
	})

	t.Run("empty validator passes", func(t *testing.T) {
		v := NewMultiLayerValidator()
		result := v.Validate(ctx, SourceConfigInput{})
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.LayerResults)
	})

	t.Run("errors are flattened across layers", func(t *testing.T) {
		v := NewMultiLayerValidator()
		v.AddLayer(ValidationLayer{
			Name: "layer1",
			Check: func(_ context.Context, _ SourceConfigInput) []ValidationError {
				return []ValidationError{
					{Field: "a", Message: "err1"},
					{Field: "b", Message: "err2"},
				}
			},
		})
		v.AddLayer(ValidationLayer{
			Name: "layer2",
			Check: func(_ context.Context, _ SourceConfigInput) []ValidationError {
				return []ValidationError{
					{Field: "c", Message: "err3"},
				}
			},
		})

		result := v.Validate(ctx, SourceConfigInput{})
		assert.Len(t, result.Errors, 3)
		assert.Equal(t, "err1", result.Errors[0].Message)
		assert.Equal(t, "err2", result.Errors[1].Message)
		assert.Equal(t, "err3", result.Errors[2].Message)
	})

	t.Run("chaining AddLayer", func(t *testing.T) {
		v := NewMultiLayerValidator().
			AddLayer(ValidationLayer{Name: "a", Check: func(_ context.Context, _ SourceConfigInput) []ValidationError { return nil }}).
			AddLayer(ValidationLayer{Name: "b", Check: func(_ context.Context, _ SourceConfigInput) []ValidationError { return nil }})

		result := v.Validate(ctx, SourceConfigInput{})
		assert.True(t, result.Valid)
		assert.Len(t, result.LayerResults, 2)
	})
}

func TestNewDefaultValidator(t *testing.T) {
	t.Run("with nil SourceManager", func(t *testing.T) {
		v := NewDefaultValidator(nil)
		require.NotNil(t, v)
		// Should have 3 layers: yaml_parse, strict_fields, semantic
		assert.Len(t, v.layers, 3)
		assert.Equal(t, "yaml_parse", v.layers[0].Name)
		assert.Equal(t, "strict_fields", v.layers[1].Name)
		assert.Equal(t, "semantic", v.layers[2].Name)
	})

	t.Run("with non-nil SourceManager", func(t *testing.T) {
		sm := &mgmtTestPlugin{}
		v := NewDefaultValidator(sm)
		require.NotNil(t, v)
		// Should have 4 layers: yaml_parse, strict_fields, semantic, provider
		assert.Len(t, v.layers, 4)
		assert.Equal(t, "yaml_parse", v.layers[0].Name)
		assert.Equal(t, "strict_fields", v.layers[1].Name)
		assert.Equal(t, "semantic", v.layers[2].Name)
		assert.Equal(t, "provider", v.layers[3].Name)
	})

	t.Run("DefaultValidator alias works", func(t *testing.T) {
		v := DefaultValidator(nil)
		require.NotNil(t, v)
		assert.Len(t, v.layers, 3)
	})

	t.Run("full validation with valid input", func(t *testing.T) {
		v := NewDefaultValidator(nil)
		result := v.Validate(context.Background(), SourceConfigInput{
			ID:   "test-source",
			Name: "Test Source",
			Type: "yaml",
		})
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("full validation with missing fields", func(t *testing.T) {
		v := NewDefaultValidator(nil)
		result := v.Validate(context.Background(), SourceConfigInput{})
		assert.False(t, result.Valid)
		// Should fail at semantic layer for missing id, name, type
		assert.GreaterOrEqual(t, len(result.Errors), 3)
	})
}

func TestGetContentString(t *testing.T) {
	tests := []struct {
		name    string
		input   SourceConfigInput
		want    string
		wantOk  bool
	}{
		{
			name:   "valid content string",
			input:  SourceConfigInput{Properties: map[string]any{"content": "hello"}},
			want:   "hello",
			wantOk: true,
		},
		{
			name:   "nil properties",
			input:  SourceConfigInput{Properties: nil},
			want:   "",
			wantOk: false,
		},
		{
			name:   "no content key",
			input:  SourceConfigInput{Properties: map[string]any{"other": "val"}},
			want:   "",
			wantOk: false,
		},
		{
			name:   "content is not a string",
			input:  SourceConfigInput{Properties: map[string]any{"content": 42}},
			want:   "",
			wantOk: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := getContentString(tc.input)
			assert.Equal(t, tc.wantOk, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

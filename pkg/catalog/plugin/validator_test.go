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
			name: "valid envelope fields",
			input: SourceConfigInput{
				ID:   "test",
				Name: "Test",
				Type: "yaml",
			},
			wantErrs: 0,
		},
		{
			name: "valid with content property (content is not validated here)",
			input: SourceConfigInput{
				ID:   "test",
				Name: "Test",
				Type: "yaml",
				Properties: map[string]any{
					"content": "mcpservers:\n- name: foo\n",
				},
			},
			wantErrs: 0,
		},
		{
			name: "valid with non-content properties",
			input: SourceConfigInput{
				ID:   "test",
				Name: "Test",
				Type: "yaml",
				Properties: map[string]any{
					"path": "/data/servers.yaml",
				},
			},
			wantErrs: 0,
		},
		{
			name: "nil properties",
			input: SourceConfigInput{
				ID:         "test",
				Name:       "Test",
				Type:       "yaml",
				Properties: nil,
			},
			wantErrs: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := layer.Check(ctx, tc.input)
			assert.Len(t, errs, tc.wantErrs)
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

	t.Run("provider catches unknown fields in content", func(t *testing.T) {
		// Simulates a plugin ValidateSource that performs strict content decoding
		// and rejects unknown fields — the pattern implemented by the MCP plugin.
		sm := &mgmtTestPlugin{
			validateFn: func(input SourceConfigInput) (*ValidationResult, error) {
				if input.Properties == nil {
					return &ValidationResult{Valid: true}, nil
				}
				if _, ok := input.Properties["content"]; !ok {
					return &ValidationResult{Valid: true}, nil
				}
				// The plugin detected an unknown field in the content.
				return &ValidationResult{
					Valid: false,
					Errors: []ValidationError{
						{
							Field:   "properties.content",
							Message: "unknown or invalid fields in content: line 4: field unknownField not found",
						},
					},
				}, nil
			},
		}
		layer := ProviderLayer(sm)

		errs := layer.Check(ctx, SourceConfigInput{
			ID:   "test",
			Name: "Test",
			Type: "yaml",
			Properties: map[string]any{
				"content": "mcpservers:\n- name: foo\n  unknownField: true\n",
			},
		})
		require.Len(t, errs, 1)
		assert.Equal(t, "properties.content", errs[0].Field)
		assert.Contains(t, errs[0].Message, "unknown or invalid fields in content")
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
		// Should have 4 layers: yaml_parse, strict_fields, semantic, security_warnings
		assert.Len(t, v.layers, 4)
		assert.Equal(t, "yaml_parse", v.layers[0].Name)
		assert.Equal(t, "strict_fields", v.layers[1].Name)
		assert.Equal(t, "semantic", v.layers[2].Name)
		assert.Equal(t, "security_warnings", v.layers[3].Name)
	})

	t.Run("with non-nil SourceManager", func(t *testing.T) {
		sm := &mgmtTestPlugin{}
		v := NewDefaultValidator(sm)
		require.NotNil(t, v)
		// Should have 5 layers: yaml_parse, strict_fields, semantic, security_warnings, provider
		assert.Len(t, v.layers, 5)
		assert.Equal(t, "yaml_parse", v.layers[0].Name)
		assert.Equal(t, "strict_fields", v.layers[1].Name)
		assert.Equal(t, "semantic", v.layers[2].Name)
		assert.Equal(t, "security_warnings", v.layers[3].Name)
		assert.Equal(t, "provider", v.layers[4].Name)
	})

	t.Run("DefaultValidator alias works", func(t *testing.T) {
		v := DefaultValidator(nil)
		require.NotNil(t, v)
		assert.Len(t, v.layers, 4)
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

func TestSecurityWarningsLayer(t *testing.T) {
	layer := SecurityWarningsLayer()
	ctx := context.Background()

	assert.Equal(t, "security_warnings", layer.Name)
	assert.True(t, layer.WarningOnly)
	assert.False(t, layer.Critical)

	t.Run("no properties produces no warnings", func(t *testing.T) {
		errs := layer.Check(ctx, SourceConfigInput{
			ID:   "test",
			Name: "Test",
			Type: "yaml",
		})
		assert.Empty(t, errs)
	})

	t.Run("nil properties produces no warnings", func(t *testing.T) {
		errs := layer.Check(ctx, SourceConfigInput{
			ID:         "test",
			Name:       "Test",
			Type:       "yaml",
			Properties: nil,
		})
		assert.Empty(t, errs)
	})

	t.Run("non-sensitive properties produce no warnings", func(t *testing.T) {
		errs := layer.Check(ctx, SourceConfigInput{
			ID:   "test",
			Name: "Test",
			Type: "yaml",
			Properties: map[string]any{
				"url":  "https://example.com",
				"path": "/data/file.yaml",
			},
		})
		assert.Empty(t, errs)
	})

	t.Run("inline password produces warning", func(t *testing.T) {
		errs := layer.Check(ctx, SourceConfigInput{
			ID:   "test",
			Name: "Test",
			Type: "yaml",
			Properties: map[string]any{
				"url":      "https://example.com",
				"password": "hunter2",
			},
		})
		require.Len(t, errs, 1)
		assert.Equal(t, "properties.password", errs[0].Field)
		assert.Contains(t, errs[0].Message, "inline credential")
		assert.Contains(t, errs[0].Message, "SecretRef")
	})

	t.Run("SecretRef map value does not produce warning", func(t *testing.T) {
		errs := layer.Check(ctx, SourceConfigInput{
			ID:   "test",
			Name: "Test",
			Type: "yaml",
			Properties: map[string]any{
				"url": "https://example.com",
				"apiKeyRef": map[string]any{
					"name": "my-secret",
					"key":  "api-key",
				},
			},
		})
		assert.Empty(t, errs)
	})

	t.Run("multiple inline credentials produce multiple warnings", func(t *testing.T) {
		errs := layer.Check(ctx, SourceConfigInput{
			ID:   "test",
			Name: "Test",
			Type: "yaml",
			Properties: map[string]any{
				"password":   "pass1",
				"token":      "tok1",
				"credential": "cred1",
			},
		})
		assert.Len(t, errs, 3)
	})
}

func TestWarningOnlyLayer(t *testing.T) {
	ctx := context.Background()

	t.Run("warning-only layer does not affect Valid flag", func(t *testing.T) {
		v := NewMultiLayerValidator()
		v.AddLayer(ValidationLayer{
			Name:        "warnings",
			WarningOnly: true,
			Check: func(_ context.Context, _ SourceConfigInput) []ValidationError {
				return []ValidationError{{Field: "x", Message: "just a warning"}}
			},
		})

		result := v.Validate(ctx, SourceConfigInput{ID: "test"})
		assert.True(t, result.Valid, "warnings should not invalidate the result")
		assert.Empty(t, result.Errors)
		assert.Len(t, result.Warnings, 1)
		assert.Equal(t, "just a warning", result.Warnings[0].Message)
	})

	t.Run("warning-only layer plus error layer", func(t *testing.T) {
		v := NewMultiLayerValidator()
		v.AddLayer(ValidationLayer{
			Name:        "warnings",
			WarningOnly: true,
			Check: func(_ context.Context, _ SourceConfigInput) []ValidationError {
				return []ValidationError{{Field: "w", Message: "a warning"}}
			},
		})
		v.AddLayer(ValidationLayer{
			Name: "errors",
			Check: func(_ context.Context, _ SourceConfigInput) []ValidationError {
				return []ValidationError{{Field: "e", Message: "an error"}}
			},
		})

		result := v.Validate(ctx, SourceConfigInput{ID: "test"})
		assert.False(t, result.Valid, "error layer should invalidate")
		assert.Len(t, result.Errors, 1)
		assert.Len(t, result.Warnings, 1)
		assert.Equal(t, "a warning", result.Warnings[0].Message)
		assert.Equal(t, "an error", result.Errors[0].Message)
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

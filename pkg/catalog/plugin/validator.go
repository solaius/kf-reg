package plugin

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationLayer represents a single validation check that runs as part of
// the multi-layer validation pipeline.
type ValidationLayer struct {
	// Name identifies this layer (e.g., "yaml_parse", "strict_fields", "semantic", "provider").
	Name string

	// Critical indicates whether a failure in this layer should stop further validation.
	Critical bool

	// Check runs the validation and returns any errors found.
	Check func(ctx context.Context, input SourceConfigInput) []ValidationError
}

// LayerValidationResult holds the result of a single validation layer.
type LayerValidationResult struct {
	// Layer is the name of the validation layer.
	Layer string `json:"layer"`

	// Valid is true if no errors were found in this layer.
	Valid bool `json:"valid"`

	// Errors lists the validation errors found in this layer.
	Errors []ValidationError `json:"errors,omitempty"`
}

// DetailedValidationResult is the result of running multi-layer validation.
type DetailedValidationResult struct {
	// Valid is true if all layers passed without errors.
	Valid bool `json:"valid"`

	// Errors is the flattened list of all errors across all layers.
	Errors []ValidationError `json:"errors,omitempty"`

	// Warnings lists non-fatal issues found during validation.
	Warnings []ValidationError `json:"warnings,omitempty"`

	// LayerResults provides per-layer breakdown of validation results.
	LayerResults []LayerValidationResult `json:"layerResults,omitempty"`
}

// MultiLayerValidator runs a sequence of validation layers against a source
// configuration input. Layers are executed in order; if a critical layer
// fails, subsequent layers are skipped.
type MultiLayerValidator struct {
	layers []ValidationLayer
}

// NewMultiLayerValidator creates a new empty MultiLayerValidator.
func NewMultiLayerValidator() *MultiLayerValidator {
	return &MultiLayerValidator{}
}

// AddLayer appends a validation layer to the pipeline. Returns the validator
// for chaining.
func (v *MultiLayerValidator) AddLayer(layer ValidationLayer) *MultiLayerValidator {
	v.layers = append(v.layers, layer)
	return v
}

// Validate runs all layers in order against the input. If a critical layer
// fails, subsequent layers are skipped and the result is marked invalid.
func (v *MultiLayerValidator) Validate(ctx context.Context, input SourceConfigInput) *DetailedValidationResult {
	result := &DetailedValidationResult{
		Valid: true,
	}

	for _, layer := range v.layers {
		errs := layer.Check(ctx, input)

		layerResult := LayerValidationResult{
			Layer:  layer.Name,
			Valid:  len(errs) == 0,
			Errors: errs,
		}
		result.LayerResults = append(result.LayerResults, layerResult)

		if len(errs) > 0 {
			result.Valid = false
			result.Errors = append(result.Errors, errs...)

			// Stop validation on critical layer failure.
			if layer.Critical {
				break
			}
		}
	}

	return result
}

// YAMLParseLayer returns a validation layer that checks whether the source
// config's properties.content field contains valid YAML.
func YAMLParseLayer() ValidationLayer {
	return ValidationLayer{
		Name:     "yaml_parse",
		Critical: true,
		Check: func(_ context.Context, input SourceConfigInput) []ValidationError {
			content, ok := getContentString(input)
			if !ok {
				// No content field; nothing to parse-check.
				return nil
			}

			var out any
			if err := yaml.Unmarshal([]byte(content), &out); err != nil {
				return []ValidationError{
					{
						Field:   "properties.content",
						Message: fmt.Sprintf("YAML parse error: %v", err),
					},
				}
			}
			return nil
		},
	}
}

// StrictFieldsLayer returns a validation layer that uses strict YAML decoding
// to detect unknown fields in the source configuration content.
func StrictFieldsLayer() ValidationLayer {
	return ValidationLayer{
		Name: "strict_fields",
		Check: func(_ context.Context, input SourceConfigInput) []ValidationError {
			content, ok := getContentString(input)
			if !ok {
				return nil
			}

			// Try strict decoding into a SourceConfig to detect unknown fields.
			dec := yaml.NewDecoder(strings.NewReader(content))
			dec.KnownFields(true)

			var cfg SourceConfig
			if err := dec.Decode(&cfg); err != nil {
				return []ValidationError{
					{
						Field:   "properties.content",
						Message: fmt.Sprintf("unknown or invalid fields: %v", err),
					},
				}
			}
			return nil
		},
	}
}

// idPattern validates source IDs: lowercase alphanumeric, hyphens, underscores.
var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// SemanticLayer returns a validation layer that checks required fields and
// value constraints on the source configuration input.
func SemanticLayer() ValidationLayer {
	return ValidationLayer{
		Name: "semantic",
		Check: func(_ context.Context, input SourceConfigInput) []ValidationError {
			var errs []ValidationError

			if input.ID == "" {
				errs = append(errs, ValidationError{
					Field:   "id",
					Message: "source ID is required",
				})
			} else if !idPattern.MatchString(input.ID) {
				errs = append(errs, ValidationError{
					Field:   "id",
					Message: "source ID must contain only lowercase alphanumeric characters, hyphens, and underscores, and must start with a letter or digit",
				})
			}

			if input.Name == "" {
				errs = append(errs, ValidationError{
					Field:   "name",
					Message: "source name is required",
				})
			} else if len(input.Name) > 256 {
				errs = append(errs, ValidationError{
					Field:   "name",
					Message: "source name must be 256 characters or fewer",
				})
			}

			if input.Type == "" {
				errs = append(errs, ValidationError{
					Field:   "type",
					Message: "source type is required",
				})
			}

			return errs
		},
	}
}

// ProviderLayer returns a validation layer that delegates to the plugin's
// ValidateSource method for provider-specific checks.
func ProviderLayer(sm SourceManager) ValidationLayer {
	return ValidationLayer{
		Name: "provider",
		Check: func(ctx context.Context, input SourceConfigInput) []ValidationError {
			result, err := sm.ValidateSource(ctx, input)
			if err != nil {
				return []ValidationError{
					{
						Message: fmt.Sprintf("provider validation error: %v", err),
					},
				}
			}
			if result != nil && !result.Valid {
				return result.Errors
			}
			return nil
		},
	}
}

// DefaultValidator is an alias for NewDefaultValidator.
var DefaultValidator = NewDefaultValidator

// NewDefaultValidator creates a MultiLayerValidator with the standard built-in
// layers: YAML parse, strict fields, semantic, and optionally provider.
func NewDefaultValidator(sm SourceManager) *MultiLayerValidator {
	v := NewMultiLayerValidator()
	v.AddLayer(YAMLParseLayer())
	v.AddLayer(StrictFieldsLayer())
	v.AddLayer(SemanticLayer())
	if sm != nil {
		v.AddLayer(ProviderLayer(sm))
	}
	return v
}

// getContentString extracts the "content" key from Properties as a string.
// Returns the content string and true if found, empty and false otherwise.
func getContentString(input SourceConfigInput) (string, bool) {
	if input.Properties == nil {
		return "", false
	}
	raw, ok := input.Properties["content"]
	if !ok {
		return "", false
	}
	s, ok := raw.(string)
	if !ok {
		return "", false
	}
	return s, true
}

package plugin

import "fmt"

// UIHintsValidationError represents a validation error in UI hints.
type UIHintsValidationError struct {
	Field   string
	Message string
}

func (e UIHintsValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateUIHints validates the UI hints for an entity.
func ValidateUIHints(hints *EntityUIHints) []UIHintsValidationError {
	if hints == nil {
		return nil
	}
	var errors []UIHintsValidationError

	// Validate ListView hints
	if hints.ListView != nil {
		for i, col := range hints.ListView.Columns {
			if col.Field == "" {
				errors = append(errors, UIHintsValidationError{
					Field:   fmt.Sprintf("listView.columns[%d].field", i),
					Message: "field is required",
				})
			}
			if col.Label == "" {
				errors = append(errors, UIHintsValidationError{
					Field:   fmt.Sprintf("listView.columns[%d].label", i),
					Message: "label is required",
				})
			}
			if col.Display != "" && !IsValidDisplayType(col.Display) {
				errors = append(errors, UIHintsValidationError{
					Field:   fmt.Sprintf("listView.columns[%d].display", i),
					Message: fmt.Sprintf("invalid display type: %q", col.Display),
				})
			}
		}
		if hints.ListView.DefaultSort != nil {
			if hints.ListView.DefaultSort.Field == "" {
				errors = append(errors, UIHintsValidationError{
					Field:   "listView.defaultSort.field",
					Message: "field is required",
				})
			}
			dir := hints.ListView.DefaultSort.Direction
			if dir != "" && dir != "asc" && dir != "desc" {
				errors = append(errors, UIHintsValidationError{
					Field:   "listView.defaultSort.direction",
					Message: fmt.Sprintf("must be 'asc' or 'desc', got %q", dir),
				})
			}
		}
	}

	// Validate DetailView hints
	if hints.DetailView != nil {
		for i, section := range hints.DetailView.Sections {
			if section.Title == "" {
				errors = append(errors, UIHintsValidationError{
					Field:   fmt.Sprintf("detailView.sections[%d].title", i),
					Message: "title is required",
				})
			}
			for j, field := range section.Fields {
				if field.Field == "" {
					errors = append(errors, UIHintsValidationError{
						Field:   fmt.Sprintf("detailView.sections[%d].fields[%d].field", i, j),
						Message: "field is required",
					})
				}
				if field.Display != "" && !IsValidDisplayType(field.Display) {
					errors = append(errors, UIHintsValidationError{
						Field:   fmt.Sprintf("detailView.sections[%d].fields[%d].display", i, j),
						Message: fmt.Sprintf("invalid display type: %q", field.Display),
					})
				}
			}
		}
	}

	// Validate Search hints
	if hints.Search != nil {
		for i, facet := range hints.Search.Facets {
			if facet.Field == "" {
				errors = append(errors, UIHintsValidationError{
					Field:   fmt.Sprintf("search.facets[%d].field", i),
					Message: "field is required",
				})
			}
		}
	}

	// Validate Action hints
	if hints.ActionHints != nil {
		for i, conf := range hints.ActionHints.Confirmations {
			if conf.Action == "" {
				errors = append(errors, UIHintsValidationError{
					Field:   fmt.Sprintf("actionHints.confirmations[%d].action", i),
					Message: "action is required",
				})
			}
			if conf.Prompt == "" {
				errors = append(errors, UIHintsValidationError{
					Field:   fmt.Sprintf("actionHints.confirmations[%d].prompt", i),
					Message: "prompt is required",
				})
			}
		}
	}

	return errors
}

package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUIHints_NilHints(t *testing.T) {
	errs := ValidateUIHints(nil)
	assert.Nil(t, errs)
}

func TestValidateUIHints_EmptyHints(t *testing.T) {
	errs := ValidateUIHints(&EntityUIHints{})
	assert.Empty(t, errs)
}

func TestValidateUIHints_ValidComplete(t *testing.T) {
	hints := &EntityUIHints{
		Icon:           "server",
		Color:          "#0066CC",
		NameField:      "name",
		DetailSections: []string{"Overview", "Config"},
		ListView: &ListViewHints{
			TitleField: "name",
			Columns: []ColumnDisplay{
				{Field: "name", Label: "Name", Display: DisplayLink},
				{Field: "status", Label: "Status", Display: DisplayBadge, Width: "sm"},
			},
			DefaultSort: &SortHint{Field: "name", Direction: "asc"},
			DefaultFilters: []DefaultFilter{
				{FilterQuery: "state != 'archived'"},
			},
		},
		DetailView: &DetailViewHints{
			Sections: []DetailSection{
				{
					Title: "Overview",
					Fields: []FieldDisplay{
						{Field: "name", Label: "Name", Display: DisplayText},
						{Field: "description", Label: "Description", Display: DisplayMarkdown},
					},
				},
				{
					Title:  "Diagnostics",
					Panels: []string{"auditTrail", "refreshStatus"},
				},
			},
		},
		Search: &SearchHints{
			SearchableFields: []string{"name", "description"},
			Facets: []FacetHint{
				{Field: "protocol", Display: DisplayBadge},
			},
		},
		ActionHints: &ActionDisplayHints{
			Primary:   []string{"refresh"},
			Secondary: []string{"tag", "deprecate"},
			Confirmations: []ActionConfirmation{
				{Action: "deprecate", Prompt: "Are you sure you want to deprecate this?"},
			},
		},
	}

	errs := ValidateUIHints(hints)
	assert.Empty(t, errs)
}

func TestValidateUIHints_InvalidDisplayType(t *testing.T) {
	hints := &EntityUIHints{
		ListView: &ListViewHints{
			Columns: []ColumnDisplay{
				{Field: "name", Label: "Name", Display: "invalid_type"},
			},
		},
	}

	errs := ValidateUIHints(hints)
	require.Len(t, errs, 1)
	assert.Equal(t, "listView.columns[0].display", errs[0].Field)
	assert.Contains(t, errs[0].Message, "invalid display type")
}

func TestValidateUIHints_MissingColumnFields(t *testing.T) {
	hints := &EntityUIHints{
		ListView: &ListViewHints{
			Columns: []ColumnDisplay{
				{Field: "", Label: ""},
			},
		},
	}

	errs := ValidateUIHints(hints)
	require.Len(t, errs, 2)
	assert.Equal(t, "listView.columns[0].field", errs[0].Field)
	assert.Equal(t, "listView.columns[0].label", errs[1].Field)
}

func TestValidateUIHints_InvalidSortDirection(t *testing.T) {
	hints := &EntityUIHints{
		ListView: &ListViewHints{
			DefaultSort: &SortHint{Field: "name", Direction: "up"},
		},
	}

	errs := ValidateUIHints(hints)
	require.Len(t, errs, 1)
	assert.Equal(t, "listView.defaultSort.direction", errs[0].Field)
	assert.Contains(t, errs[0].Message, "'asc' or 'desc'")
}

func TestValidateUIHints_MissingSortField(t *testing.T) {
	hints := &EntityUIHints{
		ListView: &ListViewHints{
			DefaultSort: &SortHint{Direction: "asc"},
		},
	}

	errs := ValidateUIHints(hints)
	require.Len(t, errs, 1)
	assert.Equal(t, "listView.defaultSort.field", errs[0].Field)
}

func TestValidateUIHints_MissingSectionTitle(t *testing.T) {
	hints := &EntityUIHints{
		DetailView: &DetailViewHints{
			Sections: []DetailSection{
				{Title: ""},
			},
		},
	}

	errs := ValidateUIHints(hints)
	require.Len(t, errs, 1)
	assert.Equal(t, "detailView.sections[0].title", errs[0].Field)
}

func TestValidateUIHints_MissingDetailFieldField(t *testing.T) {
	hints := &EntityUIHints{
		DetailView: &DetailViewHints{
			Sections: []DetailSection{
				{
					Title: "Overview",
					Fields: []FieldDisplay{
						{Field: "", Label: "Name"},
					},
				},
			},
		},
	}

	errs := ValidateUIHints(hints)
	require.Len(t, errs, 1)
	assert.Equal(t, "detailView.sections[0].fields[0].field", errs[0].Field)
}

func TestValidateUIHints_InvalidDetailFieldDisplayType(t *testing.T) {
	hints := &EntityUIHints{
		DetailView: &DetailViewHints{
			Sections: []DetailSection{
				{
					Title: "Overview",
					Fields: []FieldDisplay{
						{Field: "name", Label: "Name", Display: "bogus"},
					},
				},
			},
		},
	}

	errs := ValidateUIHints(hints)
	require.Len(t, errs, 1)
	assert.Equal(t, "detailView.sections[0].fields[0].display", errs[0].Field)
	assert.Contains(t, errs[0].Message, "invalid display type")
}

func TestValidateUIHints_SecretRefIsValid(t *testing.T) {
	hints := &EntityUIHints{
		DetailView: &DetailViewHints{
			Sections: []DetailSection{
				{
					Title: "Credentials",
					Fields: []FieldDisplay{
						{Field: "apiKey", Label: "API Key", Display: DisplaySecretRef},
					},
				},
			},
		},
	}

	errs := ValidateUIHints(hints)
	assert.Empty(t, errs)
}

func TestValidateUIHints_MissingFacetField(t *testing.T) {
	hints := &EntityUIHints{
		Search: &SearchHints{
			Facets: []FacetHint{
				{Field: ""},
			},
		},
	}

	errs := ValidateUIHints(hints)
	require.Len(t, errs, 1)
	assert.Equal(t, "search.facets[0].field", errs[0].Field)
}

func TestValidateUIHints_MissingConfirmationAction(t *testing.T) {
	hints := &EntityUIHints{
		ActionHints: &ActionDisplayHints{
			Confirmations: []ActionConfirmation{
				{Action: "", Prompt: "Are you sure?"},
			},
		},
	}

	errs := ValidateUIHints(hints)
	require.Len(t, errs, 1)
	assert.Equal(t, "actionHints.confirmations[0].action", errs[0].Field)
}

func TestValidateUIHints_MissingConfirmationPrompt(t *testing.T) {
	hints := &EntityUIHints{
		ActionHints: &ActionDisplayHints{
			Confirmations: []ActionConfirmation{
				{Action: "delete", Prompt: ""},
			},
		},
	}

	errs := ValidateUIHints(hints)
	require.Len(t, errs, 1)
	assert.Equal(t, "actionHints.confirmations[0].prompt", errs[0].Field)
}

func TestValidateUIHints_MultipleErrors(t *testing.T) {
	hints := &EntityUIHints{
		ListView: &ListViewHints{
			Columns: []ColumnDisplay{
				{Field: "", Label: "", Display: "bogus"},
			},
			DefaultSort: &SortHint{Field: "", Direction: "sideways"},
		},
		DetailView: &DetailViewHints{
			Sections: []DetailSection{
				{Title: ""},
			},
		},
		Search: &SearchHints{
			Facets: []FacetHint{
				{Field: ""},
			},
		},
		ActionHints: &ActionDisplayHints{
			Confirmations: []ActionConfirmation{
				{Action: "", Prompt: ""},
			},
		},
	}

	errs := ValidateUIHints(hints)
	// Expected: field empty, label empty, invalid display, sort field empty, sort direction invalid,
	// section title empty, facet field empty, confirmation action empty, confirmation prompt empty
	assert.Len(t, errs, 9)
}

func TestValidateUIHints_EmptySubStructs(t *testing.T) {
	hints := &EntityUIHints{
		ListView:    &ListViewHints{},
		DetailView:  &DetailViewHints{},
		Search:      &SearchHints{},
		ActionHints: &ActionDisplayHints{},
	}

	errs := ValidateUIHints(hints)
	assert.Empty(t, errs)
}

func TestValidateUIHints_ValidSortDirections(t *testing.T) {
	tests := []struct {
		name      string
		direction string
		wantErr   bool
	}{
		{name: "asc", direction: "asc", wantErr: false},
		{name: "desc", direction: "desc", wantErr: false},
		{name: "empty", direction: "", wantErr: false},
		{name: "invalid", direction: "up", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := &EntityUIHints{
				ListView: &ListViewHints{
					DefaultSort: &SortHint{Field: "name", Direction: tt.direction},
				},
			}
			errs := ValidateUIHints(hints)
			if tt.wantErr {
				assert.NotEmpty(t, errs)
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

func TestIsValidDisplayType(t *testing.T) {
	tests := []struct {
		name  string
		dt    FieldDisplayType
		valid bool
	}{
		{"text", DisplayText, true},
		{"markdown", DisplayMarkdown, true},
		{"badge", DisplayBadge, true},
		{"tags", DisplayTags, true},
		{"link", DisplayLink, true},
		{"repoRef", DisplayRepoRef, true},
		{"imageRef", DisplayImageRef, true},
		{"dateTime", DisplayDateTime, true},
		{"code", DisplayCode, true},
		{"json", DisplayJSON, true},
		{"secretRef", DisplaySecretRef, true},
		{"invalid", FieldDisplayType("invalid"), false},
		{"empty", FieldDisplayType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidDisplayType(tt.dt))
		})
	}
}

func TestUIHintsValidationError_Error(t *testing.T) {
	err := UIHintsValidationError{
		Field:   "listView.columns[0].field",
		Message: "field is required",
	}
	assert.Equal(t, "listView.columns[0].field: field is required", err.Error())
}

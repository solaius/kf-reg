package plugin

// FieldDisplayType defines how a field should be rendered in the UI.
type FieldDisplayType string

const (
	DisplayText      FieldDisplayType = "text"
	DisplayMarkdown  FieldDisplayType = "markdown"
	DisplayBadge     FieldDisplayType = "badge"
	DisplayTags      FieldDisplayType = "tags"
	DisplayLink      FieldDisplayType = "link"
	DisplayRepoRef   FieldDisplayType = "repoRef"
	DisplayImageRef  FieldDisplayType = "imageRef"
	DisplayDateTime  FieldDisplayType = "dateTime"
	DisplayCode      FieldDisplayType = "code"
	DisplayJSON      FieldDisplayType = "json"
	DisplaySecretRef FieldDisplayType = "secretRef"
)

// ValidFieldDisplayTypes returns all valid FieldDisplayType values.
func ValidFieldDisplayTypes() []FieldDisplayType {
	return []FieldDisplayType{
		DisplayText, DisplayMarkdown, DisplayBadge, DisplayTags,
		DisplayLink, DisplayRepoRef, DisplayImageRef, DisplayDateTime,
		DisplayCode, DisplayJSON, DisplaySecretRef,
	}
}

// IsValidDisplayType checks if a display type is valid.
func IsValidDisplayType(dt FieldDisplayType) bool {
	for _, valid := range ValidFieldDisplayTypes() {
		if dt == valid {
			return true
		}
	}
	return false
}

// ListViewHints provides rendering hints for list/table views.
type ListViewHints struct {
	TitleField     string          `json:"titleField,omitempty"`
	Columns        []ColumnDisplay `json:"columns,omitempty"`
	DefaultSort    *SortHint       `json:"defaultSort,omitempty"`
	DefaultFilters []DefaultFilter `json:"defaultFilters,omitempty"`
}

// ColumnDisplay describes a column in the list view.
type ColumnDisplay struct {
	Field   string           `json:"field"`
	Label   string           `json:"label"`
	Display FieldDisplayType `json:"display,omitempty"`
	Width   string           `json:"width,omitempty"` // "sm", "md", "lg"
}

// SortHint specifies default sorting.
type SortHint struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // "asc" or "desc"
}

// DefaultFilter specifies a default filter applied to the list view.
type DefaultFilter struct {
	FilterQuery string `json:"filterQuery"`
}

// DetailViewHints provides rendering hints for detail/show views.
type DetailViewHints struct {
	Sections []DetailSection `json:"sections,omitempty"`
}

// DetailSection groups fields into a section on the detail view.
type DetailSection struct {
	Title  string         `json:"title"`
	Fields []FieldDisplay `json:"fields,omitempty"`
	Panels []string       `json:"panels,omitempty"` // special panels like "auditTrail", "refreshStatus"
}

// FieldDisplay describes how a field is rendered in the detail view.
type FieldDisplay struct {
	Field   string           `json:"field"`
	Label   string           `json:"label,omitempty"`
	Display FieldDisplayType `json:"display,omitempty"`
}

// SearchHints provides search and faceting configuration.
type SearchHints struct {
	SearchableFields []string    `json:"searchableFields,omitempty"`
	Facets           []FacetHint `json:"facets,omitempty"`
}

// FacetHint describes a faceted search dimension.
type FacetHint struct {
	Field   string           `json:"field"`
	Display FieldDisplayType `json:"display,omitempty"`
}

// ActionDisplayHints provides rendering hints for actions.
type ActionDisplayHints struct {
	Primary       []string             `json:"primary,omitempty"`       // action IDs for primary buttons
	Secondary     []string             `json:"secondary,omitempty"`     // action IDs for secondary/overflow menu
	Confirmations []ActionConfirmation `json:"confirmations,omitempty"` // confirmation dialogs
}

// ActionConfirmation defines a confirmation dialog for a destructive action.
type ActionConfirmation struct {
	Action string `json:"action"` // action ID
	Prompt string `json:"prompt"` // confirmation message
}

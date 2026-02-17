package conformance

import (
	"fmt"
	"testing"
)

func testCapabilities(t *testing.T, p pluginInfo) {
	t.Helper()

	// Check V2 capabilities via dedicated endpoint.
	var caps capabilitiesV2
	getJSON(t, fmt.Sprintf("/api/plugins/%s/capabilities", p.Name), &caps)

	if caps.SchemaVersion == "" {
		t.Error("schemaVersion is empty")
	}

	if caps.Plugin.Name == "" {
		t.Error("plugin.name is empty")
	}

	if caps.Plugin.Name != p.Name {
		t.Errorf("plugin.name mismatch: got %q, want %q", caps.Plugin.Name, p.Name)
	}

	if caps.Plugin.Version == "" {
		t.Error("plugin.version is empty")
	}

	if caps.Plugin.Description == "" {
		t.Error("plugin.description is empty")
	}

	if len(caps.Entities) == 0 {
		t.Error("no entities defined")
	}

	for _, entity := range caps.Entities {
		t.Run("entity_"+entity.Kind, func(t *testing.T) {
			if entity.Kind == "" {
				t.Error("entity kind is empty")
			}
			if entity.Plural == "" {
				t.Error("entity plural is empty")
			}
			if entity.DisplayName == "" {
				t.Error("entity displayName is empty")
			}
			if entity.Endpoints.List == "" {
				t.Error("entity list endpoint is empty")
			}
			if entity.Endpoints.Get == "" {
				t.Error("entity get endpoint is empty")
			}
			if len(entity.Fields.Columns) == 0 {
				t.Error("entity has no column definitions")
			}

			// Validate column definitions.
			for _, col := range entity.Fields.Columns {
				if col.Name == "" {
					t.Error("column has empty name")
				}
				if col.DisplayName == "" {
					t.Errorf("column %q has empty displayName", col.Name)
				}
				if col.Path == "" {
					t.Errorf("column %q has empty path", col.Name)
				}
				if col.Type == "" {
					t.Errorf("column %q has empty type", col.Name)
				}
			}

			// Validate filter field definitions.
			for _, f := range entity.Fields.FilterFields {
				if f.Name == "" {
					t.Error("filter field has empty name")
				}
				if f.Type == "" {
					t.Errorf("filter field %q has empty type", f.Name)
				}
			}

			// Validate detail field definitions.
			for _, d := range entity.Fields.DetailFields {
				if d.Name == "" {
					t.Error("detail field has empty name")
				}
				if d.Path == "" {
					t.Errorf("detail field %q has empty path", d.Name)
				}
			}

			// Validate action references point to real action definitions.
			for _, actionID := range entity.Actions {
				found := false
				for _, a := range caps.Actions {
					if a.ID == actionID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("entity references action %q but no such action definition exists", actionID)
				}
			}
		})
	}

	// Validate action definitions.
	for _, action := range caps.Actions {
		t.Run("action_def_"+action.ID, func(t *testing.T) {
			if action.ID == "" {
				t.Error("action ID is empty")
			}
			if action.DisplayName == "" {
				t.Errorf("action %q has empty displayName", action.ID)
			}
			if action.Description == "" {
				t.Errorf("action %q has empty description", action.ID)
			}
			if action.Scope == "" {
				t.Errorf("action %q has empty scope", action.ID)
			}
			if action.Scope != "source" && action.Scope != "asset" {
				t.Errorf("action %q has invalid scope %q (expected 'source' or 'asset')", action.ID, action.Scope)
			}
		})
	}

	// Inline V2 capabilities should also be present in /api/plugins response.
	if p.CapabilitiesV2 == nil {
		t.Error("V2 capabilities not included inline in /api/plugins response")
	}

	// Cross-check inline caps match dedicated endpoint.
	if p.CapabilitiesV2 != nil {
		if p.CapabilitiesV2.SchemaVersion != caps.SchemaVersion {
			t.Errorf("inline schemaVersion %q != endpoint schemaVersion %q",
				p.CapabilitiesV2.SchemaVersion, caps.SchemaVersion)
		}
		if len(p.CapabilitiesV2.Entities) != len(caps.Entities) {
			t.Errorf("inline entity count %d != endpoint entity count %d",
				len(p.CapabilitiesV2.Entities), len(caps.Entities))
		}
	}
}

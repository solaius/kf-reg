package conformance

import (
	"fmt"
	"testing"
)

func runCategoryCapabilities(t *testing.T, serverURL string, p PluginInfo) CategoryResult {
	t.Helper()

	cat := CategoryResult{Name: "capabilities"}

	record := func(name, status, msg string) {
		cat.Tests = append(cat.Tests, TestResult{Name: name, Status: status, Message: msg})
		switch status {
		case "passed":
			cat.Passed++
		case "failed":
			cat.Failed++
		case "skipped":
			cat.Skipped++
		}
	}

	// Fetch capabilities from dedicated endpoint.
	var caps CapabilitiesV2
	GetJSON(t, serverURL, fmt.Sprintf("/api/plugins/%s/capabilities", p.Name), &caps)

	// Validate schema version.
	if caps.SchemaVersion == "" {
		t.Errorf("plugin %s: schemaVersion is empty", p.Name)
		record("schemaVersion", "failed", "empty")
	} else {
		record("schemaVersion", "passed", "")
	}

	// Validate plugin meta.
	if caps.Plugin.Name == "" {
		t.Errorf("plugin %s: plugin.name is empty", p.Name)
		record("plugin.name", "failed", "empty")
	} else if caps.Plugin.Name != p.Name {
		t.Errorf("plugin %s: plugin.name mismatch: got %q", p.Name, caps.Plugin.Name)
		record("plugin.name", "failed", fmt.Sprintf("mismatch: %q", caps.Plugin.Name))
	} else {
		record("plugin.name", "passed", "")
	}

	if caps.Plugin.Version == "" {
		t.Errorf("plugin %s: plugin.version is empty", p.Name)
		record("plugin.version", "failed", "empty")
	} else {
		record("plugin.version", "passed", "")
	}

	if caps.Plugin.Description == "" {
		t.Errorf("plugin %s: plugin.description is empty", p.Name)
		record("plugin.description", "failed", "empty")
	} else {
		record("plugin.description", "passed", "")
	}

	// Validate entities exist.
	if len(caps.Entities) == 0 {
		t.Errorf("plugin %s: no entities defined", p.Name)
		record("entities.count", "failed", "no entities")
	} else {
		record("entities.count", "passed", fmt.Sprintf("%d entities", len(caps.Entities)))
	}

	// Validate each entity.
	for _, entity := range caps.Entities {
		prefix := fmt.Sprintf("entity.%s", entity.Kind)

		if entity.Kind == "" {
			t.Errorf("plugin %s: entity kind is empty", p.Name)
			record(prefix+".kind", "failed", "empty")
		} else {
			record(prefix+".kind", "passed", "")
		}

		if entity.Plural == "" {
			t.Errorf("plugin %s: entity %s plural is empty", p.Name, entity.Kind)
			record(prefix+".plural", "failed", "empty")
		} else {
			record(prefix+".plural", "passed", "")
		}

		if entity.DisplayName == "" {
			t.Errorf("plugin %s: entity %s displayName is empty", p.Name, entity.Kind)
			record(prefix+".displayName", "failed", "empty")
		} else {
			record(prefix+".displayName", "passed", "")
		}

		if entity.Endpoints.List == "" {
			t.Errorf("plugin %s: entity %s list endpoint is empty", p.Name, entity.Kind)
			record(prefix+".endpoints.list", "failed", "empty")
		} else {
			record(prefix+".endpoints.list", "passed", "")
		}

		if entity.Endpoints.Get == "" {
			t.Errorf("plugin %s: entity %s get endpoint is empty", p.Name, entity.Kind)
			record(prefix+".endpoints.get", "failed", "empty")
		} else {
			record(prefix+".endpoints.get", "passed", "")
		}

		if len(entity.Fields.Columns) == 0 {
			t.Errorf("plugin %s: entity %s has no column definitions", p.Name, entity.Kind)
			record(prefix+".columns", "failed", "no columns")
		} else {
			record(prefix+".columns", "passed", fmt.Sprintf("%d columns", len(entity.Fields.Columns)))
		}

		// Validate column definitions.
		for _, col := range entity.Fields.Columns {
			colPrefix := fmt.Sprintf("%s.col.%s", prefix, col.Name)
			if col.Name == "" {
				t.Errorf("plugin %s: entity %s: column has empty name", p.Name, entity.Kind)
				record(colPrefix+".name", "failed", "empty")
			}
			if col.DisplayName == "" {
				t.Errorf("plugin %s: entity %s: column %q has empty displayName", p.Name, entity.Kind, col.Name)
			}
			if col.Path == "" {
				t.Errorf("plugin %s: entity %s: column %q has empty path", p.Name, entity.Kind, col.Name)
			}
			if col.Type == "" {
				t.Errorf("plugin %s: entity %s: column %q has empty type", p.Name, entity.Kind, col.Name)
			}
		}

		// Validate filter fields.
		for _, f := range entity.Fields.FilterFields {
			if f.Name == "" {
				t.Errorf("plugin %s: entity %s: filter field has empty name", p.Name, entity.Kind)
			}
			if f.Type == "" {
				t.Errorf("plugin %s: entity %s: filter field %q has empty type", p.Name, entity.Kind, f.Name)
			}
		}

		// Validate detail fields.
		for _, d := range entity.Fields.DetailFields {
			if d.Name == "" {
				t.Errorf("plugin %s: entity %s: detail field has empty name", p.Name, entity.Kind)
			}
			if d.Path == "" {
				t.Errorf("plugin %s: entity %s: detail field %q has empty path", p.Name, entity.Kind, d.Name)
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
				t.Errorf("plugin %s: entity %s references action %q but no such action definition exists", p.Name, entity.Kind, actionID)
				record(prefix+".action."+actionID, "failed", "undefined action reference")
			} else {
				record(prefix+".action."+actionID, "passed", "")
			}
		}
	}

	// Validate action definitions.
	for _, action := range caps.Actions {
		aPrefix := fmt.Sprintf("action.%s", action.ID)
		if action.ID == "" {
			t.Errorf("plugin %s: action ID is empty", p.Name)
			record(aPrefix+".id", "failed", "empty")
		} else {
			record(aPrefix+".id", "passed", "")
		}
		if action.DisplayName == "" {
			t.Errorf("plugin %s: action %q has empty displayName", p.Name, action.ID)
		}
		if action.Description == "" {
			t.Errorf("plugin %s: action %q has empty description", p.Name, action.ID)
		}
		if action.Scope == "" {
			t.Errorf("plugin %s: action %q has empty scope", p.Name, action.ID)
			record(aPrefix+".scope", "failed", "empty")
		} else if action.Scope != "source" && action.Scope != "asset" {
			t.Errorf("plugin %s: action %q has invalid scope %q", p.Name, action.ID, action.Scope)
			record(aPrefix+".scope", "failed", fmt.Sprintf("invalid: %q", action.Scope))
		} else {
			record(aPrefix+".scope", "passed", "")
		}
	}

	// Cross-check inline capabilities match dedicated endpoint.
	if p.CapabilitiesV2 == nil {
		t.Errorf("plugin %s: V2 capabilities not included inline in /api/plugins response", p.Name)
		record("inline.present", "failed", "missing")
	} else {
		record("inline.present", "passed", "")
		if p.CapabilitiesV2.SchemaVersion != caps.SchemaVersion {
			t.Errorf("plugin %s: inline schemaVersion %q != endpoint schemaVersion %q",
				p.Name, p.CapabilitiesV2.SchemaVersion, caps.SchemaVersion)
			record("inline.schemaVersion.match", "failed", "mismatch")
		} else {
			record("inline.schemaVersion.match", "passed", "")
		}
		if len(p.CapabilitiesV2.Entities) != len(caps.Entities) {
			t.Errorf("plugin %s: inline entity count %d != endpoint entity count %d",
				p.Name, len(p.CapabilitiesV2.Entities), len(caps.Entities))
			record("inline.entities.count.match", "failed", "mismatch")
		} else {
			record("inline.entities.count.match", "passed", "")
		}
	}

	return cat
}

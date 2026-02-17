package conformance

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
)

func testFilters(t *testing.T, p pluginInfo) {
	t.Helper()

	if p.CapabilitiesV2 == nil {
		t.Skip("no V2 capabilities")
	}

	for _, entity := range p.CapabilitiesV2.Entities {
		if len(entity.Fields.FilterFields) == 0 {
			continue
		}

		for _, filter := range entity.Fields.FilterFields {
			t.Run(fmt.Sprintf("%s/%s", entity.Plural, filter.Name), func(t *testing.T) {
				// Build a filter query using an appropriate value for the field type.
				var filterQuery string
				switch filter.Type {
				case "boolean":
					filterQuery = fmt.Sprintf("%s=true", filter.Name)
				case "number", "integer", "numeric":
					filterQuery = fmt.Sprintf("%s>0", filter.Name)
				default:
					// Use a string value for text/select fields.
					filterQuery = fmt.Sprintf("%s='test'", filter.Name)
				}
				reqURL := fmt.Sprintf("%s?filterQuery=%s",
					entity.Endpoints.List, url.QueryEscape(filterQuery))

				resp, err := http.Get(serverURL + reqURL)
				if err != nil {
					t.Fatalf("GET %s failed: %v", reqURL, err)
				}
				defer resp.Body.Close()

				// 200 OK is expected even with 0 results. 400 means filter not accepted.
				if resp.StatusCode == http.StatusBadRequest {
					body, _ := io.ReadAll(resp.Body)
					t.Errorf("filter %q not accepted: %s", filter.Name, string(body))
				}

				if resp.StatusCode >= 500 {
					body, _ := io.ReadAll(resp.Body)
					t.Errorf("filter %q caused server error: %d %s",
						filter.Name, resp.StatusCode, string(body))
				}
			})
		}

		// Test ordering if sortable columns exist.
		for _, col := range entity.Fields.Columns {
			if !col.Sortable {
				continue
			}
			t.Run(fmt.Sprintf("%s/orderBy_%s", entity.Plural, col.Name), func(t *testing.T) {
				reqURL := fmt.Sprintf("%s?orderBy=%s&sortOrder=ASC",
					entity.Endpoints.List, url.QueryEscape(col.Name))

				resp, err := http.Get(serverURL + reqURL)
				if err != nil {
					t.Fatalf("GET %s failed: %v", reqURL, err)
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusBadRequest {
					body, _ := io.ReadAll(resp.Body)
					t.Errorf("orderBy %q not accepted: %s", col.Name, string(body))
				}

				if resp.StatusCode >= 500 {
					body, _ := io.ReadAll(resp.Body)
					t.Errorf("orderBy %q caused server error: %d %s",
						col.Name, resp.StatusCode, string(body))
				}
			})
		}
	}
}

package conformance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func runCategoryListGet(t *testing.T, serverURL string, p PluginInfo) CategoryResult {
	t.Helper()

	cat := CategoryResult{Name: "list_get"}

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

	if p.CapabilitiesV2 == nil {
		record("capabilities", "skipped", "no V2 capabilities")
		return cat
	}

	for _, entity := range p.CapabilitiesV2.Entities {
		testPrefix := fmt.Sprintf("%s.%s", p.Name, entity.Plural)

		// Test list endpoint returns valid JSON with items array.
		t.Run("list_"+entity.Plural, func(t *testing.T) {
			resp, err := http.Get(serverURL + entity.Endpoints.List)
			if err != nil {
				t.Fatalf("GET %s failed: %v", entity.Endpoints.List, err)
				record(testPrefix+".list", "failed", err.Error())
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("GET %s returned %d: %s", entity.Endpoints.List, resp.StatusCode, string(body))
				record(testPrefix+".list", "failed", fmt.Sprintf("status %d", resp.StatusCode))
				return
			}

			var result map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
				record(testPrefix+".list.json", "failed", err.Error())
				return
			}

			// Should have items array.
			if _, hasItems := result["items"]; !hasItems {
				if _, hasSize := result["size"]; !hasSize {
					t.Log("warning: response has neither 'items' nor 'size' field")
				}
			}

			if items, ok := result["items"]; ok {
				if _, isArr := items.([]any); !isArr {
					t.Error("'items' field is not an array")
					record(testPrefix+".list.items_type", "failed", "not an array")
				} else {
					record(testPrefix+".list", "passed", "")
				}
			} else {
				record(testPrefix+".list", "passed", "no items key (may be empty)")
			}

			if size, ok := result["size"]; ok {
				if _, isNum := size.(float64); !isNum {
					t.Error("'size' field is not a number")
					record(testPrefix+".list.size_type", "failed", "not a number")
				}
			}
		})

		// Test get-by-name works.
		t.Run("get_"+entity.Plural+"_first", func(t *testing.T) {
			useManagementEndpoint := strings.Count(entity.Endpoints.Get, "{") > 1

			var listResp map[string]any
			GetJSON(t, serverURL, entity.Endpoints.List, &listResp)

			items, ok := listResp["items"].([]any)
			if !ok || len(items) == 0 {
				record(testPrefix+".get", "skipped", "no items to test")
				t.Skip("no items to test get endpoint")
			}

			first, ok := items[0].(map[string]any)
			if !ok {
				t.Fatal("first item is not a JSON object")
			}

			name, _ := first["name"].(string)
			if name == "" {
				record(testPrefix+".get", "skipped", "first item has no name")
				t.Skip("first item has no name field")
			}

			var getURL string
			if useManagementEndpoint {
				listPath := entity.Endpoints.List
				lastSlash := strings.LastIndex(listPath, "/")
				basePath := listPath[:lastSlash]
				getURL = basePath + "/entities/" + name
			} else {
				getURL = strings.Replace(entity.Endpoints.Get, "{name}", name, 1)
			}

			resp, err := http.Get(serverURL + getURL)
			if err != nil {
				t.Fatalf("GET %s failed: %v", getURL, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("GET %s returned %d: %s", getURL, resp.StatusCode, string(body))
			}

			var detail map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
				t.Fatalf("detail response is not valid JSON: %v", err)
			}

			detailName, _ := detail["name"].(string)
			if detailName != name {
				t.Errorf("detail name %q does not match requested %q", detailName, name)
				record(testPrefix+".get", "failed", "name mismatch")
			} else {
				record(testPrefix+".get", "passed", "")
			}
		})

		// Test get for nonexistent entity returns 404.
		t.Run("get_"+entity.Plural+"_not_found", func(t *testing.T) {
			useManagementEndpoint := strings.Count(entity.Endpoints.Get, "{") > 1

			var getURL string
			if useManagementEndpoint {
				listPath := entity.Endpoints.List
				lastSlash := strings.LastIndex(listPath, "/")
				basePath := listPath[:lastSlash]
				getURL = basePath + "/entities/nonexistent-entity-12345"
			} else {
				getURL = strings.Replace(entity.Endpoints.Get, "{name}", "nonexistent-entity-12345", 1)
			}

			resp, err := http.Get(serverURL + getURL)
			if err != nil {
				t.Fatalf("GET %s failed: %v", getURL, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				t.Errorf("expected non-200 for nonexistent entity, got %d", resp.StatusCode)
				record(testPrefix+".not_found", "failed", "got 200")
			} else if resp.StatusCode == http.StatusNotFound {
				record(testPrefix+".not_found", "passed", "")
			} else {
				// 400 or 500 are acceptable but not ideal.
				body, _ := io.ReadAll(resp.Body)
				t.Logf("note: nonexistent entity returned %d (expected 404): %s", resp.StatusCode, string(body))
				record(testPrefix+".not_found", "passed", fmt.Sprintf("returned %d", resp.StatusCode))
			}
		})

		// Test pagination (pageSize parameter).
		t.Run("pagination_"+entity.Plural, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s?pageSize=1", entity.Endpoints.List)
			resp, err := http.Get(serverURL + reqURL)
			if err != nil {
				t.Fatalf("GET %s failed: %v", reqURL, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				record(testPrefix+".pagination", "skipped", "list endpoint not available")
				t.Skipf("GET %s returned 404", reqURL)
			}

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("GET %s returned %d: %s", reqURL, resp.StatusCode, string(body))
			}

			var result map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}

			items, ok := result["items"].([]any)
			if !ok {
				record(testPrefix+".pagination", "skipped", "no items array")
				t.Skip("response has no 'items' array")
			}

			if len(items) > 1 {
				t.Logf("note: pageSize=1 but got %d items (pagination may not be enforced)", len(items))
			}
			record(testPrefix+".pagination", "passed", "")
		})

		// Test filter fields are accepted.
		for _, filter := range entity.Fields.FilterFields {
			filterName := filter.Name
			t.Run(fmt.Sprintf("filter_%s/%s", entity.Plural, filterName), func(t *testing.T) {
				var filterQuery string
				switch filter.Type {
				case "boolean":
					filterQuery = fmt.Sprintf("%s=true", filterName)
				case "number", "integer", "numeric":
					filterQuery = fmt.Sprintf("%s>0", filterName)
				default:
					filterQuery = fmt.Sprintf("%s='test'", filterName)
				}
				reqURL := fmt.Sprintf("%s?filterQuery=%s",
					entity.Endpoints.List, url.QueryEscape(filterQuery))

				resp, err := http.Get(serverURL + reqURL)
				if err != nil {
					t.Fatalf("GET %s failed: %v", reqURL, err)
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusBadRequest {
					body, _ := io.ReadAll(resp.Body)
					t.Errorf("filter %q not accepted: %s", filterName, string(body))
					record(testPrefix+".filter."+filterName, "failed", "400 Bad Request")
				} else if resp.StatusCode >= 500 {
					body, _ := io.ReadAll(resp.Body)
					t.Errorf("filter %q caused server error: %d %s", filterName, resp.StatusCode, string(body))
					record(testPrefix+".filter."+filterName, "failed", fmt.Sprintf("status %d", resp.StatusCode))
				} else {
					record(testPrefix+".filter."+filterName, "passed", "")
				}
			})
		}

		// Test ordering on sortable columns.
		for _, col := range entity.Fields.Columns {
			if !col.Sortable {
				continue
			}
			colName := col.Name
			t.Run(fmt.Sprintf("orderBy_%s/%s", entity.Plural, colName), func(t *testing.T) {
				reqURL := fmt.Sprintf("%s?orderBy=%s&sortOrder=ASC",
					entity.Endpoints.List, url.QueryEscape(colName))

				resp, err := http.Get(serverURL + reqURL)
				if err != nil {
					t.Fatalf("GET %s failed: %v", reqURL, err)
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusBadRequest {
					body, _ := io.ReadAll(resp.Body)
					t.Errorf("orderBy %q not accepted: %s", colName, string(body))
					record(testPrefix+".orderBy."+colName, "failed", "400 Bad Request")
				} else if resp.StatusCode >= 500 {
					body, _ := io.ReadAll(resp.Body)
					t.Errorf("orderBy %q caused server error: %d %s", colName, resp.StatusCode, string(body))
					record(testPrefix+".orderBy."+colName, "failed", fmt.Sprintf("status %d", resp.StatusCode))
				} else {
					record(testPrefix+".orderBy."+colName, "passed", "")
				}
			})
		}
	}

	return cat
}

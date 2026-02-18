package plugin

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestParsePaginationParams(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantSize  int
		wantOrder string
		wantSort  string
	}{
		{
			name:      "defaults",
			query:     "",
			wantSize:  DefaultPageSize,
			wantOrder: "",
			wantSort:  "ASC",
		},
		{
			name:      "custom page size",
			query:     "pageSize=25",
			wantSize:  25,
			wantOrder: "",
			wantSort:  "ASC",
		},
		{
			name:      "page size over max clamped",
			query:     "pageSize=5000",
			wantSize:  MaxPageSize,
			wantOrder: "",
			wantSort:  "ASC",
		},
		{
			name:      "negative page size uses default",
			query:     "pageSize=-1",
			wantSize:  DefaultPageSize,
			wantOrder: "",
			wantSort:  "ASC",
		},
		{
			name:      "order and sort",
			query:     "orderBy=name&sortOrder=DESC",
			wantSize:  DefaultPageSize,
			wantOrder: "name",
			wantSort:  "DESC",
		},
		{
			name:      "sort order case insensitive",
			query:     "sortOrder=desc",
			wantSize:  DefaultPageSize,
			wantOrder: "",
			wantSort:  "DESC",
		},
		{
			name:      "invalid sort order defaults to ASC",
			query:     "sortOrder=RANDOM",
			wantSize:  DefaultPageSize,
			wantOrder: "",
			wantSort:  "ASC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/list?"+tt.query, nil)
			params := ParsePaginationParams(r)

			if params.PageSize != tt.wantSize {
				t.Errorf("PageSize = %d, want %d", params.PageSize, tt.wantSize)
			}
			if params.OrderBy != tt.wantOrder {
				t.Errorf("OrderBy = %q, want %q", params.OrderBy, tt.wantOrder)
			}
			if params.SortOrder != tt.wantSort {
				t.Errorf("SortOrder = %q, want %q", params.SortOrder, tt.wantSort)
			}
		})
	}
}

func TestPaginateSlice(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}

	tests := []struct {
		name          string
		pageSize      int
		pageToken     string
		wantLen       int
		wantFirst     string
		wantLast      string
		wantHasNext   bool
	}{
		{
			name:        "first page",
			pageSize:    3,
			pageToken:   "",
			wantLen:     3,
			wantFirst:   "a",
			wantLast:    "c",
			wantHasNext: true,
		},
		{
			name:        "second page",
			pageSize:    3,
			pageToken:   encodeOffset(3),
			wantLen:     3,
			wantFirst:   "d",
			wantLast:    "f",
			wantHasNext: true,
		},
		{
			name:        "last page partial",
			pageSize:    3,
			pageToken:   encodeOffset(9),
			wantLen:     1,
			wantFirst:   "j",
			wantLast:    "j",
			wantHasNext: false,
		},
		{
			name:        "exact last page",
			pageSize:    5,
			pageToken:   encodeOffset(5),
			wantLen:     5,
			wantFirst:   "f",
			wantLast:    "j",
			wantHasNext: false,
		},
		{
			name:        "offset beyond end",
			pageSize:    5,
			pageToken:   encodeOffset(100),
			wantLen:     0,
			wantHasNext: false,
		},
		{
			name:        "invalid token ignored",
			pageSize:    3,
			pageToken:   "invalid-token",
			wantLen:     3,
			wantFirst:   "a",
			wantLast:    "c",
			wantHasNext: true,
		},
		{
			name:        "all items in one page",
			pageSize:    100,
			pageToken:   "",
			wantLen:     10,
			wantFirst:   "a",
			wantLast:    "j",
			wantHasNext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := PaginationParams{PageSize: tt.pageSize, PageToken: tt.pageToken}
			page, nextToken := PaginateSlice(items, params)

			if len(page) != tt.wantLen {
				t.Errorf("page length = %d, want %d", len(page), tt.wantLen)
			}
			if tt.wantLen > 0 {
				if page[0] != tt.wantFirst {
					t.Errorf("first item = %q, want %q", page[0], tt.wantFirst)
				}
				if page[len(page)-1] != tt.wantLast {
					t.Errorf("last item = %q, want %q", page[len(page)-1], tt.wantLast)
				}
			}
			if tt.wantHasNext && nextToken == "" {
				t.Error("expected nextPageToken, got empty")
			}
			if !tt.wantHasNext && nextToken != "" {
				t.Errorf("expected no nextPageToken, got %q", nextToken)
			}
		})
	}
}

func TestPaginateSliceChainedPages(t *testing.T) {
	// Verify that iterating with returned page tokens covers all items exactly.
	items := make([]int, 27)
	for i := range items {
		items[i] = i
	}

	var collected []int
	params := PaginationParams{PageSize: 5}
	pages := 0

	for {
		page, nextToken := PaginateSlice(items, params)
		collected = append(collected, page...)
		pages++
		if nextToken == "" {
			break
		}
		params.PageToken = nextToken
		if pages > 100 { // safety
			t.Fatal("too many pages, possible infinite loop")
		}
	}

	if len(collected) != len(items) {
		t.Fatalf("collected %d items, expected %d", len(collected), len(items))
	}

	for i, v := range collected {
		if v != items[i] {
			t.Fatalf("item %d: got %d, want %d", i, v, items[i])
		}
	}

	// 27 items / 5 per page = 6 pages.
	if pages != 6 {
		t.Fatalf("expected 6 pages, got %d", pages)
	}
}

func TestSortByField(t *testing.T) {
	type item struct {
		Name string
	}

	items := []item{{Name: "Charlie"}, {Name: "alice"}, {Name: "Bob"}}

	SortByField(items, func(i item) string { return i.Name }, false)
	if items[0].Name != "alice" || items[1].Name != "Bob" || items[2].Name != "Charlie" {
		t.Errorf("ASC sort incorrect: %v", items)
	}

	SortByField(items, func(i item) string { return i.Name }, true)
	if items[0].Name != "Charlie" || items[1].Name != "Bob" || items[2].Name != "alice" {
		t.Errorf("DESC sort incorrect: %v", items)
	}
}

func TestSanitizeFilterQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{"empty", "", false},
		{"simple equality", "name='foo'", false},
		{"AND filter", "name='foo' AND version='1.0'", false},
		{"LIKE filter", "name LIKE '%test%'", false},
		{"SQL comment", "name='foo' -- drop", true},
		{"semicolon", "name='foo'; DROP TABLE", true},
		{"block comment", "name='foo' /* injection */", true},
		{"DROP", "name='foo' DROP TABLE users", true},
		{"DELETE", "DELETE FROM users", true},
		{"UNION", "name='foo' UNION SELECT 1", true},
		{"INSERT", "INSERT INTO users", true},
		{"UPDATE", "UPDATE users SET", true},
		{"ALTER", "ALTER TABLE users", true},
		{"CREATE", "CREATE TABLE evil", true},
		{"TRUNCATE", "TRUNCATE users", true},
		{"EXEC", "EXEC xp_cmdshell", true},
		{"EXECUTE", "EXECUTE sp_configure", true},
		{"hex literal", "0x414243", true},
		{"case insensitive block", "name='foo' union SELECT 1", true},
		{"too long", string(make([]byte, 1025)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeFilterQuery(tt.query)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for query %q, got nil", tt.query)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for query %q: %v", tt.query, err)
			}
			if !tt.wantErr && result != tt.query {
				t.Errorf("expected query passthrough, got %q", result)
			}
		})
	}
}

func TestBuildPaginatedResponse(t *testing.T) {
	items := []string{"a", "b"}
	resp := BuildPaginatedResponse(items, 10, 2, "next123")

	if resp["size"] != 10 {
		t.Errorf("size = %v, want 10", resp["size"])
	}
	if resp["pageSize"] != 2 {
		t.Errorf("pageSize = %v, want 2", resp["pageSize"])
	}
	if resp["nextPageToken"] != "next123" {
		t.Errorf("nextPageToken = %v, want next123", resp["nextPageToken"])
	}

	// Without next page token.
	resp2 := BuildPaginatedResponse(items, 2, 2, "")
	if _, ok := resp2["nextPageToken"]; ok {
		t.Error("expected no nextPageToken key when empty")
	}
}

func encodeOffset(n int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(n)))
}

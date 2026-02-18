package plugin

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// PaginationParams holds parsed pagination and ordering parameters from an
// HTTP request query string. Plugins should use ParsePaginationParams to
// extract these from incoming requests.
type PaginationParams struct {
	PageSize  int    // Max items per page (clamped to [1, MaxPageSize]).
	PageToken string // Opaque cursor for the next page (decoded offset).
	OrderBy   string // Field name to order by (plugin-specific).
	SortOrder string // "ASC" or "DESC" (default "ASC").
}

// PaginatedResult holds a page of items and an optional next-page token.
type PaginatedResult struct {
	Items         any    `json:"items"`
	Size          int    `json:"size"`
	PageSize      int    `json:"pageSize"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

const (
	// DefaultPageSize is used when pageSize is not specified.
	DefaultPageSize = 100
	// MaxPageSize is the upper bound for pageSize.
	MaxPageSize = 1000
)

// ParsePaginationParams extracts pagination parameters from the request URL
// query string. The parameters are: pageSize, pageToken, orderBy, sortOrder.
func ParsePaginationParams(r *http.Request) PaginationParams {
	q := r.URL.Query()

	pageSize := DefaultPageSize
	if v := q.Get("pageSize"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageSize = n
		}
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	sortOrder := "ASC"
	if v := strings.ToUpper(q.Get("sortOrder")); v == "DESC" {
		sortOrder = "DESC"
	}

	return PaginationParams{
		PageSize:  pageSize,
		PageToken: q.Get("pageToken"),
		OrderBy:   q.Get("orderBy"),
		SortOrder: sortOrder,
	}
}

// PaginateSlice applies offset-based pagination to a slice. The pageToken is
// a base64-encoded offset. Returns the items for the current page and the
// next-page token (empty if there are no more pages).
//
// The caller should sort the slice before calling PaginateSlice if ordering
// is desired.
func PaginateSlice[T any](items []T, params PaginationParams) (page []T, nextPageToken string) {
	offset := 0
	if params.PageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(params.PageToken)
		if err == nil {
			if n, err := strconv.Atoi(string(decoded)); err == nil && n > 0 {
				offset = n
			}
		}
	}

	total := len(items)

	if offset >= total {
		return nil, ""
	}

	end := offset + params.PageSize
	if end > total {
		end = total
	}

	page = items[offset:end]

	if end < total {
		nextPageToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(end)))
	}

	return page, nextPageToken
}

// SortByField sorts a slice in place using the provided field accessor.
// The accessor returns the string value for the sort field.
func SortByField[T any](items []T, accessor func(T) string, descending bool) {
	sort.SliceStable(items, func(i, j int) bool {
		a := strings.ToLower(accessor(items[i]))
		b := strings.ToLower(accessor(items[j]))
		if descending {
			return a > b
		}
		return a < b
	})
}

// BuildPaginatedResponse constructs a standard paginated response map.
func BuildPaginatedResponse(page any, totalSize, pageSize int, nextPageToken string) map[string]any {
	result := map[string]any{
		"items":    page,
		"size":     totalSize,
		"pageSize": pageSize,
	}
	if nextPageToken != "" {
		result["nextPageToken"] = nextPageToken
	}
	return result
}

// SanitizeFilterQuery validates that the filterQuery string does not contain
// SQL injection patterns. For in-memory plugins this is informational; for
// DB-backed plugins this is a defense-in-depth measure.
//
// Returns the original query if safe, or an error describing the problem.
func SanitizeFilterQuery(query string) (string, error) {
	if query == "" {
		return "", nil
	}

	upper := strings.ToUpper(query)

	// Block SQL injection patterns.
	blocked := []string{
		"--",        // SQL comment
		";",         // Statement terminator
		"/*",        // Block comment
		"*/",        // Block comment end
		"DROP ",     // DDL
		"DELETE ",   // DML
		"INSERT ",   // DML
		"UPDATE ",   // DML
		"ALTER ",    // DDL
		"CREATE ",   // DDL
		"TRUNCATE ", // DDL
		"EXEC ",     // Execution
		"EXECUTE ",  // Execution
		"UNION ",    // Set operation
		"0X",        // Hex literal
	}

	for _, pattern := range blocked {
		if strings.Contains(upper, pattern) {
			return "", fmt.Errorf("filterQuery contains blocked pattern %q", pattern)
		}
	}

	// Enforce maximum length to prevent abuse.
	if len(query) > 1024 {
		return "", fmt.Errorf("filterQuery exceeds maximum length of 1024 characters")
	}

	return query, nil
}

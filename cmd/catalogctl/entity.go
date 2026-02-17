package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func buildEntityCommand(p pluginInfo, entity entityCaps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   entity.Plural,
		Short: fmt.Sprintf("Manage %s entities", entity.DisplayName),
	}

	// list subcommand
	listCmd := buildEntityListCommand(p, entity)
	cmd.AddCommand(listCmd)

	// get subcommand
	getCmd := buildEntityGetCommand(p, entity)
	cmd.AddCommand(getCmd)

	// action subcommands
	if len(entity.Actions) > 0 && p.CapabilitiesV2 != nil {
		for _, actionID := range entity.Actions {
			actionCmd := buildEntityActionCommand(p, entity, actionID)
			if actionCmd != nil {
				cmd.AddCommand(actionCmd)
			}
		}
	}

	return cmd
}

func buildEntityListCommand(p pluginInfo, entity entityCaps) *cobra.Command {
	var filterQuery string
	var orderBy string
	var sortOrder string
	var pageSize int
	var nextPageToken string
	var fetchAll bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List all %s", entity.Plural),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()

			// Build query parameters.
			path := entity.Endpoints.List
			params := make([]string, 0)
			if filterQuery != "" {
				params = append(params, "filterQuery="+filterQuery)
			}
			if orderBy != "" {
				params = append(params, "orderBy="+orderBy)
			}
			if sortOrder != "" {
				params = append(params, "sortOrder="+sortOrder)
			}
			if pageSize > 0 {
				params = append(params, fmt.Sprintf("pageSize=%d", pageSize))
			}
			if nextPageToken != "" {
				params = append(params, "nextPageToken="+nextPageToken)
			}
			if len(params) > 0 {
				path += "?" + strings.Join(params, "&")
			}

			result, err := client.getRaw(path)
			if err != nil {
				return fmt.Errorf("failed to list %s: %w", entity.Plural, err)
			}

			if outputFmt == "json" || outputFmt == "yaml" {
				return printOutput(result)
			}

			// Extract the items array from the response.
			items := extractItems(result, entity.Plural)

			// If --all flag is set, fetch all remaining pages.
			if fetchAll {
				for {
					token, ok := result["nextPageToken"].(string)
					if !ok || token == "" {
						break
					}
					nextPath := entity.Endpoints.List
					nextParams := make([]string, 0)
					if filterQuery != "" {
						nextParams = append(nextParams, "filterQuery="+filterQuery)
					}
					if orderBy != "" {
						nextParams = append(nextParams, "orderBy="+orderBy)
					}
					if sortOrder != "" {
						nextParams = append(nextParams, "sortOrder="+sortOrder)
					}
					if pageSize > 0 {
						nextParams = append(nextParams, fmt.Sprintf("pageSize=%d", pageSize))
					}
					nextParams = append(nextParams, "nextPageToken="+token)
					nextPath += "?" + strings.Join(nextParams, "&")

					result, err = client.getRaw(nextPath)
					if err != nil {
						return fmt.Errorf("failed to fetch next page: %w", err)
					}
					pageItems := extractItems(result, entity.Plural)
					items = append(items, pageItems...)
				}
			}

			if len(items) == 0 {
				fmt.Printf("No %s found.\n", entity.Plural)
				return nil
			}

			// Use capabilities columns for table rendering.
			columns := entity.Fields.Columns
			if len(columns) == 0 {
				// Fallback: use first few keys from the first item.
				columns = inferColumns(items[0])
			}

			headers := make([]string, len(columns))
			for i, col := range columns {
				headers[i] = col.DisplayName
			}

			rows := make([][]string, 0, len(items))
			for _, item := range items {
				row := make([]string, len(columns))
				for i, col := range columns {
					row[i] = truncate(extractValue(item, col.Path), 50)
				}
				rows = append(rows, row)
			}

			printTable(headers, rows)

			// Show pagination token if available (only when not fetching all).
			if !fetchAll {
				if token, ok := result["nextPageToken"].(string); ok && token != "" {
					fmt.Printf("\nMore results available. Use --next-page-token %s\n", token)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&filterQuery, "filter", "", "Filter query (e.g. \"name LIKE '%server%'\")")
	cmd.Flags().StringVar(&orderBy, "order-by", "", "Field to order by")
	cmd.Flags().StringVar(&sortOrder, "sort", "", "Sort order: ASC or DESC")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Number of results per page")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Pagination token for next page")
	cmd.Flags().BoolVar(&fetchAll, "all", false, "Fetch all pages automatically")

	return cmd
}

func buildEntityGetCommand(p pluginInfo, entity entityCaps) *cobra.Command {
	return &cobra.Command{
		Use:   "get [name]",
		Short: fmt.Sprintf("Get a specific %s by name", entity.DisplayName),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()

			// Replace {name} placeholder in the get endpoint.
			path := strings.Replace(entity.Endpoints.Get, "{name}", args[0], 1)

			result, err := client.getRaw(path)
			if err != nil {
				return fmt.Errorf("failed to get %s %q: %w", entity.Kind, args[0], err)
			}

			if outputFmt == "json" || outputFmt == "yaml" {
				return printOutput(result)
			}

			// Detail view: use detail fields from capabilities.
			fields := entity.Fields.DetailFields
			if len(fields) == 0 {
				// Fallback to JSON output.
				return printJSON(result)
			}

			// Group fields by section.
			sections := make(map[string][]fieldHint)
			var sectionOrder []string
			for _, f := range fields {
				sec := f.Section
				if sec == "" {
					sec = "General"
				}
				if _, exists := sections[sec]; !exists {
					sectionOrder = append(sectionOrder, sec)
				}
				sections[sec] = append(sections[sec], f)
			}

			for _, sec := range sectionOrder {
				fmt.Printf("\n--- %s ---\n", sec)
				for _, f := range sections[sec] {
					val := extractValue(result, f.Path)
					if val == "" {
						val = "-"
					}
					fmt.Printf("  %-20s %s\n", f.DisplayName+":", val)
				}
			}
			fmt.Println()

			return nil
		},
	}
}

// extractItems tries to find the items array in a list response.
// It looks for the entity plural key first, then falls back to common keys.
func extractItems(data map[string]any, plural string) []map[string]any {
	// Try the plural key (e.g., "catalogmodels", "mcpservers").
	if items, ok := data[plural]; ok {
		return toMapSlice(items)
	}

	// Try common key names.
	for _, key := range []string{"items", "results", "data"} {
		if items, ok := data[key]; ok {
			return toMapSlice(items)
		}
	}

	return nil
}

func toMapSlice(v any) []map[string]any {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			result = append(result, m)
		}
	}
	return result
}

// inferColumns creates fallback column definitions from the first item's keys.
func inferColumns(item map[string]any) []columnHint {
	cols := make([]columnHint, 0)
	for key := range item {
		if len(cols) >= 5 {
			break
		}
		cols = append(cols, columnHint{
			Name:        key,
			DisplayName: key,
			Path:        key,
			Type:        "string",
		})
	}
	return cols
}

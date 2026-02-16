package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// sourceInfo mirrors the JSON structure from the server's SourceInfo type.
type sourceInfo struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Enabled    bool           `json:"enabled"`
	Labels     []string       `json:"labels,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	Status     sourceStatus   `json:"status"`
}

type sourceStatus struct {
	State           string     `json:"state"`
	LastRefreshTime *time.Time `json:"lastRefreshTime,omitempty"`
	EntityCount     int        `json:"entityCount"`
	Error           string     `json:"error,omitempty"`
}

type sourcesListResponse struct {
	Sources []sourceInfo `json:"sources"`
	Count   int          `json:"count"`
}

// sourceConfigInput is sent for validate and apply operations.
type sourceConfigInput struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Enabled    *bool          `json:"enabled,omitempty"`
	Labels     []string       `json:"labels,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

type validationResult struct {
	Valid  bool              `json:"valid"`
	Errors []validationError `json:"errors,omitempty"`
}

type validationError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func newSourcesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sources",
		Short: "Manage catalog data sources",
		Long:  "List, validate, apply, enable, disable, and delete data sources for a catalog plugin.",
	}

	cmd.AddCommand(newSourcesListCmd())
	cmd.AddCommand(newSourcesValidateCmd())
	cmd.AddCommand(newSourcesApplyCmd())
	cmd.AddCommand(newSourcesEnableCmd())
	cmd.AddCommand(newSourcesDisableCmd())
	cmd.AddCommand(newSourcesDeleteCmd())
	cmd.AddCommand(newSourcesDiagnosticsCmd())

	return cmd
}

// --- sources list ---

func newSourcesListCmd() *cobra.Command {
	var plugin string
	var apiVersion string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List data sources for a plugin",
		Long:  "List all configured data sources for the specified catalog plugin.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSourcesList(plugin, apiVersion)
		},
	}

	cmd.Flags().StringVar(&plugin, "plugin", "", "Plugin name (required)")
	cmd.Flags().StringVar(&apiVersion, "api-version", "v1alpha1", "API version")
	_ = cmd.MarkFlagRequired("plugin")

	return cmd
}

func runSourcesList(plugin, version string) error {
	path := fmt.Sprintf("/api/%s_catalog/%s/sources", plugin, version)

	body, err := globalClient.doRequest("GET", path, nil)
	if err != nil {
		return err
	}

	var resp sourcesListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	headers := []string{"ID", "Name", "Type", "Enabled", "State", "Entities", "Last Refresh"}
	var rows [][]string
	for _, s := range resp.Sources {
		enabled := "yes"
		if !s.Enabled {
			enabled = "no"
		}
		lastRefresh := "-"
		if s.Status.LastRefreshTime != nil {
			lastRefresh = s.Status.LastRefreshTime.Format(time.RFC3339)
		}
		rows = append(rows, []string{
			s.ID,
			s.Name,
			s.Type,
			enabled,
			s.Status.State,
			fmt.Sprintf("%d", s.Status.EntityCount),
			lastRefresh,
		})
	}

	return printOutput(os.Stdout, format, resp, headers, rows)
}

// --- sources validate ---

func newSourcesValidateCmd() *cobra.Command {
	var (
		plugin     string
		apiVersion string
		sourceID   string
		sourceName string
		sourceType string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a source configuration",
		Long:  "Validate a source configuration without applying it. Checks that all required fields and provider-specific settings are correct.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSourcesValidate(plugin, apiVersion, sourceID, sourceName, sourceType)
		},
	}

	cmd.Flags().StringVar(&plugin, "plugin", "", "Plugin name (required)")
	cmd.Flags().StringVar(&apiVersion, "api-version", "v1alpha1", "API version")
	cmd.Flags().StringVar(&sourceID, "id", "", "Source ID (required)")
	cmd.Flags().StringVar(&sourceName, "name", "", "Source display name (required)")
	cmd.Flags().StringVar(&sourceType, "type", "", "Source provider type (required)")
	_ = cmd.MarkFlagRequired("plugin")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}

func runSourcesValidate(plugin, version, id, name, srcType string) error {
	path := fmt.Sprintf("/api/%s_catalog/%s/sources/validate", plugin, version)

	input := sourceConfigInput{
		ID:   id,
		Name: name,
		Type: srcType,
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	body, err := globalClient.doRequest("POST", path, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	var result validationResult
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	if format == outputTable {
		if result.Valid {
			fmt.Fprintln(os.Stdout, "Validation: PASSED")
		} else {
			fmt.Fprintln(os.Stdout, "Validation: FAILED")
			for _, ve := range result.Errors {
				field := ve.Field
				if field == "" {
					field = "(general)"
				}
				fmt.Fprintf(os.Stdout, "  - %s: %s\n", field, ve.Message)
			}
		}
		return nil
	}

	return printOutput(os.Stdout, format, result, nil, nil)
}

// --- sources apply ---

func newSourcesApplyCmd() *cobra.Command {
	var (
		plugin     string
		apiVersion string
		sourceID   string
		sourceName string
		sourceType string
		enabled    bool
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply (create or update) a source configuration",
		Long:  "Apply a source configuration. If the source already exists, it is updated; otherwise, it is created.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSourcesApply(plugin, apiVersion, sourceID, sourceName, sourceType, enabled)
		},
	}

	cmd.Flags().StringVar(&plugin, "plugin", "", "Plugin name (required)")
	cmd.Flags().StringVar(&apiVersion, "api-version", "v1alpha1", "API version")
	cmd.Flags().StringVar(&sourceID, "id", "", "Source ID (required)")
	cmd.Flags().StringVar(&sourceName, "name", "", "Source display name (required)")
	cmd.Flags().StringVar(&sourceType, "type", "", "Source provider type (required)")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Whether the source should be enabled")
	_ = cmd.MarkFlagRequired("plugin")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}

func runSourcesApply(plugin, version, id, name, srcType string, enabled bool) error {
	path := fmt.Sprintf("/api/%s_catalog/%s/sources/apply", plugin, version)

	input := sourceConfigInput{
		ID:      id,
		Name:    name,
		Type:    srcType,
		Enabled: &enabled,
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	body, err := globalClient.doRequest("POST", path, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	if format == outputTable {
		var resp map[string]string
		if err := json.Unmarshal(body, &resp); err == nil {
			fmt.Fprintf(os.Stdout, "Source %q: %s\n", id, resp["status"])
		} else {
			fmt.Fprintln(os.Stdout, string(body))
		}
		return nil
	}

	var raw json.RawMessage = body
	return printOutput(os.Stdout, format, raw, nil, nil)
}

// --- sources enable ---

func newSourcesEnableCmd() *cobra.Command {
	var (
		plugin     string
		apiVersion string
	)

	cmd := &cobra.Command{
		Use:   "enable <source-id>",
		Short: "Enable a data source",
		Long:  "Enable a previously disabled data source so it participates in catalog ingestion.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSourcesSetEnabled(plugin, apiVersion, args[0], true)
		},
	}

	cmd.Flags().StringVar(&plugin, "plugin", "", "Plugin name (required)")
	cmd.Flags().StringVar(&apiVersion, "api-version", "v1alpha1", "API version")
	_ = cmd.MarkFlagRequired("plugin")

	return cmd
}

// --- sources disable ---

func newSourcesDisableCmd() *cobra.Command {
	var (
		plugin     string
		apiVersion string
	)

	cmd := &cobra.Command{
		Use:   "disable <source-id>",
		Short: "Disable a data source",
		Long:  "Disable a data source so it stops participating in catalog ingestion.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSourcesSetEnabled(plugin, apiVersion, args[0], false)
		},
	}

	cmd.Flags().StringVar(&plugin, "plugin", "", "Plugin name (required)")
	cmd.Flags().StringVar(&apiVersion, "api-version", "v1alpha1", "API version")
	_ = cmd.MarkFlagRequired("plugin")

	return cmd
}

func runSourcesSetEnabled(plugin, version, sourceID string, enabled bool) error {
	path := fmt.Sprintf("/api/%s_catalog/%s/sources/%s/enable", plugin, version, sourceID)

	payload, err := json.Marshal(map[string]bool{"enabled": enabled})
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	body, err := globalClient.doRequest("POST", path, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	if format == outputTable {
		action := "enabled"
		if !enabled {
			action = "disabled"
		}
		fmt.Fprintf(os.Stdout, "Source %q: %s\n", sourceID, action)
		return nil
	}

	var raw json.RawMessage = body
	return printOutput(os.Stdout, format, raw, nil, nil)
}

// --- sources delete ---

func newSourcesDeleteCmd() *cobra.Command {
	var (
		plugin     string
		apiVersion string
	)

	cmd := &cobra.Command{
		Use:   "delete <source-id>",
		Short: "Delete a data source",
		Long:  "Remove a data source and all of its associated entities from the catalog.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSourcesDelete(plugin, apiVersion, args[0])
		},
	}

	cmd.Flags().StringVar(&plugin, "plugin", "", "Plugin name (required)")
	cmd.Flags().StringVar(&apiVersion, "api-version", "v1alpha1", "API version")
	_ = cmd.MarkFlagRequired("plugin")

	return cmd
}

func runSourcesDelete(plugin, version, sourceID string) error {
	path := fmt.Sprintf("/api/%s_catalog/%s/sources/%s", plugin, version, sourceID)

	body, err := globalClient.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	if format == outputTable {
		fmt.Fprintf(os.Stdout, "Source %q: deleted\n", sourceID)
		return nil
	}

	var raw json.RawMessage = body
	return printOutput(os.Stdout, format, raw, nil, nil)
}

// --- sources diagnostics ---

func newSourcesDiagnosticsCmd() *cobra.Command {
	var (
		plugin     string
		apiVersion string
		sourceID   string
	)

	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "Show source diagnostics for a plugin",
		Long:  "Show detailed diagnostics for all sources or a specific source within a plugin.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSourcesDiagnostics(plugin, apiVersion, sourceID)
		},
	}

	cmd.Flags().StringVar(&plugin, "plugin", "", "Plugin name (required)")
	cmd.Flags().StringVar(&apiVersion, "api-version", "v1alpha1", "API version")
	cmd.Flags().StringVar(&sourceID, "source", "", "Filter diagnostics to a specific source ID")
	_ = cmd.MarkFlagRequired("plugin")

	return cmd
}

func runSourcesDiagnostics(plugin, version, sourceID string) error {
	path := fmt.Sprintf("/api/%s_catalog/%s/diagnostics", plugin, version)

	body, err := globalClient.doRequest("GET", path, nil)
	if err != nil {
		return err
	}

	var diag pluginDiagnostics
	if err := json.Unmarshal(body, &diag); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	// Filter to specific source if requested
	if sourceID != "" {
		filtered := make([]sourceDiagnostic, 0)
		for _, s := range diag.Sources {
			if s.ID == sourceID {
				filtered = append(filtered, s)
			}
		}
		diag.Sources = filtered
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	if format == outputTable {
		fmt.Fprintf(os.Stdout, "Plugin: %s\n", diag.PluginName)
		if diag.LastRefresh != nil {
			fmt.Fprintf(os.Stdout, "Last Refresh: %s\n", diag.LastRefresh.Format(time.RFC3339))
		}
		fmt.Fprintln(os.Stdout)

		if len(diag.Sources) == 0 {
			if sourceID != "" {
				fmt.Fprintf(os.Stdout, "No diagnostics found for source %q\n", sourceID)
			} else {
				fmt.Fprintln(os.Stdout, "No sources configured")
			}
			return nil
		}

		headers := []string{"ID", "Name", "State", "Entities", "Last Refresh", "Error"}
		var rows [][]string
		for _, s := range diag.Sources {
			lastRefresh := "-"
			if s.LastRefreshTime != nil {
				lastRefresh = s.LastRefreshTime.Format(time.RFC3339)
			}
			errMsg := "-"
			if s.Error != "" {
				errMsg = truncate(s.Error, 50)
			}
			rows = append(rows, []string{
				s.ID,
				s.Name,
				s.State,
				fmt.Sprintf("%d", s.EntityCount),
				lastRefresh,
				errMsg,
			})
		}
		return printTable(os.Stdout, headers, rows)
	}

	return printOutput(os.Stdout, format, diag, nil, nil)
}


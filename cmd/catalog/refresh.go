package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// refreshResult mirrors the server's RefreshResult JSON.
type refreshResult struct {
	SourceID        string        `json:"sourceId,omitempty"`
	EntitiesLoaded  int           `json:"entitiesLoaded"`
	EntitiesRemoved int           `json:"entitiesRemoved"`
	Duration        time.Duration `json:"duration"`
	Error           string        `json:"error,omitempty"`
}

func newRefreshCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Trigger catalog data refresh",
		Long:  "Trigger a refresh of catalog data from configured sources.",
	}

	cmd.AddCommand(newRefreshTriggerCmd())

	return cmd
}

func newRefreshTriggerCmd() *cobra.Command {
	var (
		plugin     string
		apiVersion string
		sourceID   string
	)

	cmd := &cobra.Command{
		Use:   "trigger",
		Short: "Trigger a refresh",
		Long: `Trigger a refresh of catalog data. By default, all sources for the
specified plugin are refreshed. Use --source to refresh a single source.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRefreshTrigger(plugin, apiVersion, sourceID)
		},
	}

	cmd.Flags().StringVar(&plugin, "plugin", "", "Plugin name (required)")
	cmd.Flags().StringVar(&apiVersion, "api-version", "v1alpha1", "API version")
	cmd.Flags().StringVar(&sourceID, "source", "", "Source ID to refresh (omit to refresh all)")
	_ = cmd.MarkFlagRequired("plugin")

	return cmd
}

func runRefreshTrigger(plugin, version, sourceID string) error {
	var path string
	if sourceID != "" {
		path = fmt.Sprintf("/api/%s_catalog/%s/refresh/%s", plugin, version, sourceID)
	} else {
		path = fmt.Sprintf("/api/%s_catalog/%s/refresh", plugin, version)
	}

	body, err := globalClient.doRequest("POST", path, nil)
	if err != nil {
		return err
	}

	var result refreshResult
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	if format == outputTable {
		target := "all sources"
		if sourceID != "" {
			target = fmt.Sprintf("source %q", sourceID)
		}
		fmt.Fprintf(os.Stdout, "Refresh completed for %s\n", target)
		fmt.Fprintf(os.Stdout, "  Entities loaded:  %d\n", result.EntitiesLoaded)
		fmt.Fprintf(os.Stdout, "  Entities removed: %d\n", result.EntitiesRemoved)
		fmt.Fprintf(os.Stdout, "  Duration:         %s\n", result.Duration)
		if result.Error != "" {
			fmt.Fprintf(os.Stdout, "  Error:            %s\n", result.Error)
		}
		return nil
	}

	return printOutput(os.Stdout, format, result, nil, nil)
}

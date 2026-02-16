package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// pluginDiagnostics mirrors the server's PluginDiagnostics JSON.
type pluginDiagnostics struct {
	PluginName  string             `json:"pluginName"`
	Sources     []sourceDiagnostic `json:"sources"`
	LastRefresh *time.Time         `json:"lastRefresh,omitempty"`
	Errors      []diagnosticError  `json:"errors,omitempty"`
}

type sourceDiagnostic struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	State               string         `json:"state"`
	EntityCount         int            `json:"entityCount"`
	LastRefreshTime     *time.Time     `json:"lastRefreshTime,omitempty"`
	LastRefreshDuration *time.Duration `json:"lastRefreshDuration,omitempty"`
	Error               string         `json:"error,omitempty"`
}

type diagnosticError struct {
	Source  string    `json:"source,omitempty"`
	Message string   `json:"message"`
	Time    time.Time `json:"time"`
}

// healthResponse mirrors the server's health endpoint response.
type healthResponse struct {
	Status string `json:"status"`
}

// readyResponse mirrors the server's readiness endpoint response.
type readyResponse struct {
	Status  string          `json:"status"`
	Plugins map[string]bool `json:"plugins"`
}

func newStatusCmd() *cobra.Command {
	var (
		plugin     string
		apiVersion string
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show catalog server status",
		Long: `Show the status of the catalog server. Without --plugin, shows the overall
server health and readiness. With --plugin, shows detailed diagnostics for
that plugin including per-source status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if plugin != "" {
				return runPluginDiagnostics(plugin, apiVersion)
			}
			return runServerStatus()
		},
	}

	cmd.Flags().StringVar(&plugin, "plugin", "", "Show diagnostics for a specific plugin")
	cmd.Flags().StringVar(&apiVersion, "api-version", "v1alpha1", "API version (used with --plugin)")

	return cmd
}

func runServerStatus() error {
	// Fetch health
	healthBody, err := globalClient.doRequest("GET", "/healthz", nil)
	if err != nil {
		return fmt.Errorf("checking server health: %w", err)
	}

	var health healthResponse
	if err := json.Unmarshal(healthBody, &health); err != nil {
		return fmt.Errorf("parsing health response: %w", err)
	}

	// Fetch readiness
	readyBody, err := globalClient.doRequest("GET", "/readyz", nil)
	if err != nil {
		// Readiness might return 503, but doRequest returns an error for 4xx/5xx.
		// Try to parse the error body in this case.
		return fmt.Errorf("checking server readiness: %w", err)
	}

	var ready readyResponse
	if err := json.Unmarshal(readyBody, &ready); err != nil {
		return fmt.Errorf("parsing readiness response: %w", err)
	}

	// Fetch plugins for enriched output
	pluginsBody, err := globalClient.doRequest("GET", "/api/plugins", nil)
	if err != nil {
		return fmt.Errorf("fetching plugins: %w", err)
	}

	var plugins pluginsResponse
	if err := json.Unmarshal(pluginsBody, &plugins); err != nil {
		return fmt.Errorf("parsing plugins response: %w", err)
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	if format == outputTable {
		fmt.Fprintf(os.Stdout, "Server:   %s\n", serverURL)
		fmt.Fprintf(os.Stdout, "Health:   %s\n", health.Status)
		fmt.Fprintf(os.Stdout, "Ready:    %s\n", ready.Status)
		fmt.Fprintf(os.Stdout, "Plugins:  %d\n\n", plugins.Count)

		if len(plugins.Plugins) > 0 {
			headers := []string{"Plugin", "Version", "Healthy", "Base Path"}
			var rows [][]string
			for _, p := range plugins.Plugins {
				healthy := "yes"
				if !p.Healthy {
					healthy = "no"
				}
				rows = append(rows, []string{
					p.Name,
					p.Version,
					healthy,
					p.BasePath,
				})
			}
			return printTable(os.Stdout, headers, rows)
		}
		return nil
	}

	// For JSON/YAML, combine into a single structure
	combined := map[string]any{
		"server":  serverURL,
		"health":  health,
		"ready":   ready,
		"plugins": plugins,
	}

	return printOutput(os.Stdout, format, combined, nil, nil)
}

func runPluginDiagnostics(plugin, version string) error {
	path := fmt.Sprintf("/api/%s_catalog/%s/diagnostics", plugin, version)

	body, err := globalClient.doRequest("GET", path, nil)
	if err != nil {
		return err
	}

	var diag pluginDiagnostics
	if err := json.Unmarshal(body, &diag); err != nil {
		return fmt.Errorf("parsing response: %w", err)
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

		if len(diag.Sources) > 0 {
			fmt.Fprintln(os.Stdout, "Sources:")
			headers := []string{"ID", "Name", "State", "Entities", "Last Refresh", "Duration", "Error"}
			var rows [][]string
			for _, s := range diag.Sources {
				lastRefresh := "-"
				if s.LastRefreshTime != nil {
					lastRefresh = s.LastRefreshTime.Format(time.RFC3339)
				}
				duration := "-"
				if s.LastRefreshDuration != nil {
					duration = s.LastRefreshDuration.String()
				}
				errMsg := "-"
				if s.Error != "" {
					errMsg = truncate(s.Error, 40)
				}
				rows = append(rows, []string{
					s.ID,
					s.Name,
					s.State,
					fmt.Sprintf("%d", s.EntityCount),
					lastRefresh,
					duration,
					errMsg,
				})
			}
			if err := printTable(os.Stdout, headers, rows); err != nil {
				return err
			}
		}

		if len(diag.Errors) > 0 {
			fmt.Fprintln(os.Stdout)
			fmt.Fprintln(os.Stdout, "Active Errors:")
			for _, e := range diag.Errors {
				source := "(plugin)"
				if e.Source != "" {
					source = e.Source
				}
				fmt.Fprintf(os.Stdout, "  [%s] %s: %s\n", e.Time.Format(time.RFC3339), source, e.Message)
			}
		}

		return nil
	}

	return printOutput(os.Stdout, format, diag, nil, nil)
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// pluginInfo mirrors the JSON structure returned by GET /api/plugins.
type pluginInfo struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	BasePath    string            `json:"basePath"`
	Healthy     bool              `json:"healthy"`
	EntityKinds []string          `json:"entityKinds,omitempty"`
	Management  *managementCaps   `json:"management,omitempty"`
	Status      *pluginStatusInfo `json:"status,omitempty"`
}

type managementCaps struct {
	SourceManager bool `json:"sourceManager"`
	Refresh       bool `json:"refresh"`
	Diagnostics   bool `json:"diagnostics"`
}

type pluginStatusInfo struct {
	Enabled     bool   `json:"enabled"`
	Initialized bool   `json:"initialized"`
	Serving     bool   `json:"serving"`
	LastError   string `json:"lastError,omitempty"`
}

type pluginsResponse struct {
	Plugins []pluginInfo `json:"plugins"`
	Count   int          `json:"count"`
}

func newPluginsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "Manage catalog plugins",
		Long:  "List and inspect catalog plugins registered with the server.",
	}

	cmd.AddCommand(newPluginsListCmd())
	cmd.AddCommand(newPluginsGetCmd())

	return cmd
}

func newPluginsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all registered plugins",
		Long:  "List all catalog plugins registered with the server, including their health status.",
		RunE:  runPluginsList,
	}
}

func newPluginsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Get details for a specific plugin",
		Long:  "Show detailed information about a specific catalog plugin by name.",
		Args:  cobra.ExactArgs(1),
		RunE:  runPluginsGet,
	}
}

func runPluginsList(cmd *cobra.Command, args []string) error {
	body, err := globalClient.doRequest("GET", "/api/plugins", nil)
	if err != nil {
		return err
	}

	var resp pluginsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	headers := []string{"Name", "Version", "Healthy", "Description", "Base Path"}
	var rows [][]string
	for _, p := range resp.Plugins {
		healthy := "yes"
		if !p.Healthy {
			healthy = "no"
		}
		rows = append(rows, []string{
			p.Name,
			p.Version,
			healthy,
			truncate(p.Description, 40),
			p.BasePath,
		})
	}

	return printOutput(os.Stdout, format, resp, headers, rows)
}

func runPluginsGet(cmd *cobra.Command, args []string) error {
	pluginName := args[0]

	body, err := globalClient.doRequest("GET", "/api/plugins", nil)
	if err != nil {
		return err
	}

	var resp pluginsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	// Find the requested plugin
	var found *pluginInfo
	for i := range resp.Plugins {
		if resp.Plugins[i].Name == pluginName {
			found = &resp.Plugins[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("plugin %q not found", pluginName)
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	if format == outputTable {
		// For table output, show a key-value detail view
		fmt.Fprintf(os.Stdout, "Name:         %s\n", found.Name)
		fmt.Fprintf(os.Stdout, "Version:      %s\n", found.Version)
		fmt.Fprintf(os.Stdout, "Description:  %s\n", found.Description)
		fmt.Fprintf(os.Stdout, "Base Path:    %s\n", found.BasePath)
		healthy := "yes"
		if !found.Healthy {
			healthy = "no"
		}
		fmt.Fprintf(os.Stdout, "Healthy:      %s\n", healthy)
		if len(found.EntityKinds) > 0 {
			fmt.Fprintf(os.Stdout, "Entity Kinds: %s\n", strings.Join(found.EntityKinds, ", "))
		}
		if found.Management != nil {
			fmt.Fprintf(os.Stdout, "Management:\n")
			fmt.Fprintf(os.Stdout, "  Sources:      %v\n", found.Management.SourceManager)
			fmt.Fprintf(os.Stdout, "  Refresh:      %v\n", found.Management.Refresh)
			fmt.Fprintf(os.Stdout, "  Diagnostics:  %v\n", found.Management.Diagnostics)
		}
		if found.Status != nil {
			fmt.Fprintf(os.Stdout, "Status:\n")
			fmt.Fprintf(os.Stdout, "  Enabled:      %v\n", found.Status.Enabled)
			fmt.Fprintf(os.Stdout, "  Initialized:  %v\n", found.Status.Initialized)
			fmt.Fprintf(os.Stdout, "  Serving:      %v\n", found.Status.Serving)
			if found.Status.LastError != "" {
				fmt.Fprintf(os.Stdout, "  Last Error:   %s\n", found.Status.LastError)
			}
		}
		return nil
	}

	return printOutput(os.Stdout, format, found, nil, nil)
}

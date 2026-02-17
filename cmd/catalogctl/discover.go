package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// discoverPlugins fetches all plugins and their capabilities,
// then registers dynamic subcommands for each plugin/entity.
func discoverPlugins() error {
	client := newClient()

	var resp pluginsResponse
	if err := client.getJSON("/api/plugins", &resp); err != nil {
		return err
	}

	for _, p := range resp.Plugins {
		if p.CapabilitiesV2 == nil {
			continue
		}

		pluginCmd := buildPluginCommand(p)
		rootCmd.AddCommand(pluginCmd)
	}

	return nil
}

// buildPluginCommand creates a command tree for a plugin from its capabilities.
func buildPluginCommand(p pluginInfo) *cobra.Command {
	desc := p.Description
	if p.CapabilitiesV2 != nil && p.CapabilitiesV2.Plugin.DisplayName != "" {
		desc = p.CapabilitiesV2.Plugin.DisplayName + " - " + desc
	}

	cmd := &cobra.Command{
		Use:   p.Name,
		Short: desc,
	}

	if p.CapabilitiesV2 != nil {
		for _, entity := range p.CapabilitiesV2.Entities {
			entityCmd := buildEntityCommand(p, entity)
			cmd.AddCommand(entityCmd)
		}

		// Add sources subcommand if sources are manageable.
		if p.CapabilitiesV2.Sources != nil {
			sourcesCmd := buildSourcesCommand(p)
			cmd.AddCommand(sourcesCmd)
		}

		// Add actions summary subcommand.
		if len(p.CapabilitiesV2.Actions) > 0 {
			actionsCmd := buildActionsListCommand(p)
			cmd.AddCommand(actionsCmd)
		}
	}

	return cmd
}

// buildActionsListCommand creates a command to list available actions for a plugin.
func buildActionsListCommand(p pluginInfo) *cobra.Command {
	actions := p.CapabilitiesV2.Actions
	return &cobra.Command{
		Use:   "actions",
		Short: "List available actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputFmt == "json" || outputFmt == "yaml" {
				return printOutput(actions)
			}

			headers := []string{"ID", "Display Name", "Scope", "Dry Run", "Destructive", "Description"}
			rows := make([][]string, 0, len(actions))
			for _, a := range actions {
				dryRun := "no"
				if a.SupportsDryRun {
					dryRun = "yes"
				}
				destructive := "no"
				if a.Destructive {
					destructive = "yes"
				}
				rows = append(rows, []string{
					a.ID,
					a.DisplayName,
					a.Scope,
					dryRun,
					destructive,
					truncate(a.Description, 50),
				})
			}
			printTable(headers, rows)
			return nil
		},
	}
}

// buildSourcesCommand creates a command tree for managing plugin sources.
func buildSourcesCommand(p pluginInfo) *cobra.Command {
	basePath := p.BasePath
	cmd := &cobra.Command{
		Use:   "sources",
		Short: "Manage data sources",
	}

	// list subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()

			var sources []sourceInfo
			if err := client.getJSON(basePath+"/sources", &sources); err != nil {
				return fmt.Errorf("failed to list sources: %w", err)
			}

			if outputFmt == "json" || outputFmt == "yaml" {
				return printOutput(sources)
			}

			headers := []string{"ID", "Name", "Type", "Enabled", "State"}
			rows := make([][]string, 0, len(sources))
			for _, s := range sources {
				enabled := "yes"
				if !s.Enabled {
					enabled = "no"
				}
				rows = append(rows, []string{
					s.ID,
					s.Name,
					s.Type,
					enabled,
					s.Status.State,
				})
			}
			printTable(headers, rows)
			return nil
		},
	}

	// refresh subcommand (if refreshable)
	var refreshCmd *cobra.Command
	if p.CapabilitiesV2.Sources.Refreshable {
		refreshCmd = &cobra.Command{
			Use:   "refresh [source-id]",
			Short: "Refresh a source (or all sources if no ID given)",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				client := newClient()

				path := basePath + "/refresh"
				if len(args) > 0 {
					path = basePath + "/sources/" + args[0] + "/refresh"
				}

				var result map[string]any
				if err := client.postJSON(path, nil, &result); err != nil {
					return fmt.Errorf("refresh failed: %w", err)
				}

				if outputFmt == "json" || outputFmt == "yaml" {
					return printOutput(result)
				}

				fmt.Println("Refresh completed successfully.")
				return nil
			},
		}
	}

	cmd.AddCommand(listCmd)
	if refreshCmd != nil {
		cmd.AddCommand(refreshCmd)
	}

	return cmd
}

// extractValue extracts a value from a nested JSON map using a dot-separated path.
// For simple paths like "name" or "protocol", it does a direct key lookup.
// For nested paths like "status.state", it traverses the map.
func extractValue(data map[string]any, path string) string {
	parts := strings.Split(path, ".")

	current := any(data)
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[part]
	}

	if current == nil {
		return ""
	}

	switch v := current.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []any:
		strs := make([]string, 0, len(v))
		for _, item := range v {
			strs = append(strs, fmt.Sprintf("%v", item))
		}
		return strings.Join(strs, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

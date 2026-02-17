package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "List all registered plugins",
	Long:  "List all catalog plugins registered on the server with their health, version, and capabilities.",
	RunE:  runPluginsList,
}

func runPluginsList(cmd *cobra.Command, args []string) error {
	client := newClient()

	var resp pluginsResponse
	if err := client.getJSON("/api/plugins", &resp); err != nil {
		return fmt.Errorf("failed to list plugins: %w", err)
	}

	if outputFmt == "json" || outputFmt == "yaml" {
		return printOutput(resp)
	}

	// Table output
	headers := []string{"Name", "Version", "Healthy", "Entities", "Description"}
	rows := make([][]string, 0, len(resp.Plugins))
	for _, p := range resp.Plugins {
		healthy := "yes"
		if !p.Healthy {
			healthy = "no"
			if p.Status != nil && p.Status.LastError != "" {
				healthy = "no (" + truncate(p.Status.LastError, 40) + ")"
			}
		}

		entities := ""
		if p.CapabilitiesV2 != nil {
			kinds := make([]string, 0, len(p.CapabilitiesV2.Entities))
			for _, e := range p.CapabilitiesV2.Entities {
				kinds = append(kinds, e.Kind)
			}
			if len(kinds) > 0 {
				entities = fmt.Sprintf("%v", kinds)
			}
		} else if len(p.EntityKinds) > 0 {
			entities = fmt.Sprintf("%v", p.EntityKinds)
		}

		rows = append(rows, []string{
			p.Name,
			p.Version,
			healthy,
			entities,
			truncate(p.Description, 50),
		})
	}

	printTable(headers, rows)
	return nil
}

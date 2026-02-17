package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check server health and readiness",
	RunE:  runHealth,
}

func runHealth(cmd *cobra.Command, args []string) error {
	client := newClient()

	// Fetch both health and readiness.
	var healthResp map[string]any
	if err := client.getJSON("/healthz", &healthResp); err != nil {
		return fmt.Errorf("server unreachable: %w", err)
	}

	var readyResp map[string]any
	if err := client.getJSON("/readyz", &readyResp); err != nil {
		// Readiness failure is not fatal; the server might still be starting.
		readyResp = map[string]any{"status": "unknown", "error": err.Error()}
	}

	if outputFmt == "json" || outputFmt == "yaml" {
		combined := map[string]any{
			"health":    healthResp,
			"readiness": readyResp,
		}
		return printOutput(combined)
	}

	// Table output
	status, _ := healthResp["status"].(string)
	uptime, _ := healthResp["uptime"].(string)
	ready, _ := readyResp["status"].(string)

	headers := []string{"Check", "Status"}
	rows := [][]string{
		{"Liveness", status},
		{"Uptime", uptime},
		{"Readiness", ready},
	}

	printTable(headers, rows)
	return nil
}

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var namespacesCmd = &cobra.Command{
	Use:   "namespaces",
	Short: "List available namespaces",
	Long:  "List namespaces accessible to the current user from the catalog server.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()

		var resp struct {
			Namespaces []string `json:"namespaces"`
			Mode       string   `json:"mode"`
		}
		if err := client.getJSON("/api/tenancy/v1alpha1/namespaces", &resp); err != nil {
			return fmt.Errorf("failed to list namespaces: %w", err)
		}

		if outputFmt == "json" || outputFmt == "yaml" {
			return printOutput(resp)
		}

		fmt.Printf("Tenancy mode: %s\n\n", resp.Mode)

		headers := []string{"Namespace"}
		rows := make([][]string, 0, len(resp.Namespaces))
		for _, ns := range resp.Namespaces {
			rows = append(rows, []string{ns})
		}
		printTable(headers, rows)

		return nil
	},
}

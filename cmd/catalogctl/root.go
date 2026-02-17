package main

import (
	"github.com/spf13/cobra"
)

var (
	serverURL string
	outputFmt string
)

var rootCmd = &cobra.Command{
	Use:   "catalogctl",
	Short: "CLI for the unified catalog server",
	Long: `catalogctl is a capabilities-driven CLI for interacting with catalog plugins.

It discovers plugins registered on the catalog server at startup and dynamically
generates subcommands for each plugin and its entity types.

Static commands (plugins, health) are always available.
Dynamic commands (per-plugin entity management) require a reachable server.`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:8080", "Catalog server URL")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table", "Output format: table, json, yaml")

	rootCmd.AddCommand(pluginsCmd)
	rootCmd.AddCommand(healthCmd)
}

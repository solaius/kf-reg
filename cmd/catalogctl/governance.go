package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const governanceAPIBase = "/api/governance/v1alpha1"

var governanceCmd = &cobra.Command{
	Use:   "governance",
	Short: "Manage governance metadata for assets",
}

var governanceGetCmd = &cobra.Command{
	Use:   "get <plugin> <kind> <name>",
	Short: "Get governance metadata for an asset",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		plugin, kind, name := args[0], args[1], args[2]
		client := newClient()

		path := fmt.Sprintf("%s/assets/%s/%s/%s", governanceAPIBase, plugin, kind, name)
		result, err := client.getRaw(path)
		if err != nil {
			return fmt.Errorf("failed to get governance: %w", err)
		}

		return printOutput(result)
	},
}

var (
	setOwner string
	setTeam  string
	setRisk  string
	setSLA   string
)

var governanceSetCmd = &cobra.Command{
	Use:   "set <plugin> <kind> <name>",
	Short: "Set governance metadata for an asset",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		plugin, kind, name := args[0], args[1], args[2]
		client := newClient()

		path := fmt.Sprintf("%s/assets/%s/%s/%s", governanceAPIBase, plugin, kind, name)

		overlay := make(map[string]any)
		if setOwner != "" {
			overlay["owner"] = map[string]any{"principal": setOwner}
		}
		if setTeam != "" {
			overlay["team"] = map[string]any{"name": setTeam}
		}
		if setRisk != "" {
			overlay["risk"] = map[string]any{"level": setRisk}
		}
		if setSLA != "" {
			overlay["sla"] = map[string]any{"tier": setSLA}
		}

		if len(overlay) == 0 {
			return fmt.Errorf("at least one of --owner, --team, --risk, --sla must be specified")
		}

		var result map[string]any
		if err := client.patchJSON(path, overlay, &result); err != nil {
			return fmt.Errorf("failed to set governance: %w", err)
		}

		return printOutput(result)
	},
}

var versionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "Manage asset versions",
}

var versionsListCmd = &cobra.Command{
	Use:   "list <plugin> <kind> <name>",
	Short: "List versions for an asset",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		plugin, kind, name := args[0], args[1], args[2]
		client := newClient()

		path := fmt.Sprintf("%s/assets/%s/%s/%s/versions", governanceAPIBase, plugin, kind, name)

		var result struct {
			Versions []struct {
				VersionID    string `json:"versionId"`
				VersionLabel string `json:"versionLabel"`
				CreatedBy    string `json:"createdBy"`
				CreatedAt    string `json:"createdAt"`
			} `json:"versions"`
			TotalSize int `json:"totalSize"`
		}
		if err := client.getJSON(path, &result); err != nil {
			return fmt.Errorf("failed to list versions: %w", err)
		}

		if outputFmt == "json" || outputFmt == "yaml" {
			return printOutput(result)
		}

		headers := []string{"Version ID", "Label", "Created By", "Created At"}
		rows := make([][]string, 0, len(result.Versions))
		for _, v := range result.Versions {
			rows = append(rows, []string{v.VersionID, v.VersionLabel, v.CreatedBy, v.CreatedAt})
		}
		printTable(headers, rows)
		fmt.Printf("Total: %d\n", result.TotalSize)
		return nil
	},
}

var (
	versionLabel  string
	versionReason string
)

var versionsCreateCmd = &cobra.Command{
	Use:   "create <plugin> <kind> <name>",
	Short: "Create a new version for an asset",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		plugin, kind, name := args[0], args[1], args[2]
		client := newClient()

		path := fmt.Sprintf("%s/assets/%s/%s/%s/versions", governanceAPIBase, plugin, kind, name)

		body := map[string]string{
			"versionLabel": versionLabel,
			"reason":       versionReason,
		}

		var result map[string]any
		if err := client.postJSON(path, body, &result); err != nil {
			return fmt.Errorf("failed to create version: %w", err)
		}

		return printOutput(result)
	},
}

var (
	promoteFrom string
	promoteTo   string
)

var promoteCmd = &cobra.Command{
	Use:   "promote <plugin> <kind> <name>",
	Short: "Promote an asset from one environment to another",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		plugin, kind, name := args[0], args[1], args[2]
		client := newClient()

		path := fmt.Sprintf("%s/assets/%s/%s/%s/actions/promotion.promote", governanceAPIBase, plugin, kind, name)

		body := map[string]any{
			"params": map[string]any{
				"fromEnvironment": promoteFrom,
				"toEnvironment":   promoteTo,
			},
		}

		var result map[string]any
		if err := client.postJSON(path, body, &result); err != nil {
			return fmt.Errorf("failed to promote: %w", err)
		}

		return printOutput(result)
	},
}

var (
	rollbackEnv     string
	rollbackVersion string
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback <plugin> <kind> <name>",
	Short: "Rollback an environment binding to a previous version",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		plugin, kind, name := args[0], args[1], args[2]
		client := newClient()

		path := fmt.Sprintf("%s/assets/%s/%s/%s/actions/promotion.rollback", governanceAPIBase, plugin, kind, name)

		body := map[string]any{
			"params": map[string]any{
				"environment": rollbackEnv,
				"versionId":   rollbackVersion,
			},
		}

		var result map[string]any
		if err := client.postJSON(path, body, &result); err != nil {
			return fmt.Errorf("failed to rollback: %w", err)
		}

		return printOutput(result)
	},
}

var bindingsCmd = &cobra.Command{
	Use:   "bindings <plugin> <kind> <name>",
	Short: "List environment bindings for an asset",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		plugin, kind, name := args[0], args[1], args[2]
		client := newClient()

		path := fmt.Sprintf("%s/assets/%s/%s/%s/bindings", governanceAPIBase, plugin, kind, name)

		var result struct {
			Bindings []struct {
				Environment       string `json:"environment"`
				VersionID         string `json:"versionId"`
				BoundBy           string `json:"boundBy"`
				BoundAt           string `json:"boundAt"`
				PreviousVersionID string `json:"previousVersionId"`
			} `json:"bindings"`
		}
		if err := client.getJSON(path, &result); err != nil {
			return fmt.Errorf("failed to list bindings: %w", err)
		}

		if outputFmt == "json" || outputFmt == "yaml" {
			return printOutput(result)
		}

		headers := []string{"Environment", "Version ID", "Bound By", "Bound At", "Previous"}
		rows := make([][]string, 0, len(result.Bindings))
		for _, b := range result.Bindings {
			prev := b.PreviousVersionID
			if prev == "" {
				prev = "-"
			}
			rows = append(rows, []string{b.Environment, b.VersionID, b.BoundBy, b.BoundAt, prev})
		}
		printTable(headers, rows)
		return nil
	},
}

var historyCmd = &cobra.Command{
	Use:   "history <plugin> <kind> <name>",
	Short: "Show audit history for an asset",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		plugin, kind, name := args[0], args[1], args[2]
		client := newClient()

		path := fmt.Sprintf("%s/assets/%s/%s/%s/history", governanceAPIBase, plugin, kind, name)

		var result struct {
			Events []struct {
				EventType string `json:"eventType"`
				Actor     string `json:"actor"`
				Action    string `json:"action"`
				Outcome   string `json:"outcome"`
				Reason    string `json:"reason"`
				CreatedAt string `json:"createdAt"`
			} `json:"events"`
			TotalSize int `json:"totalSize"`
		}
		if err := client.getJSON(path, &result); err != nil {
			return fmt.Errorf("failed to get history: %w", err)
		}

		if outputFmt == "json" || outputFmt == "yaml" {
			return printOutput(result)
		}

		headers := []string{"Event", "Actor", "Action", "Outcome", "Reason", "Time"}
		rows := make([][]string, 0, len(result.Events))
		for _, e := range result.Events {
			reason := e.Reason
			if reason == "" {
				reason = "-"
			}
			rows = append(rows, []string{e.EventType, e.Actor, e.Action, e.Outcome, truncate(reason, 40), e.CreatedAt})
		}
		printTable(headers, rows)
		fmt.Printf("Total: %d\n", result.TotalSize)
		return nil
	},
}

func init() {
	// governance set flags
	governanceSetCmd.Flags().StringVar(&setOwner, "owner", "", "Asset owner principal")
	governanceSetCmd.Flags().StringVar(&setTeam, "team", "", "Asset team name")
	governanceSetCmd.Flags().StringVar(&setRisk, "risk", "", "Risk level (low, medium, high, critical)")
	governanceSetCmd.Flags().StringVar(&setSLA, "sla", "", "SLA tier (gold, silver, bronze, none)")

	governanceCmd.AddCommand(governanceGetCmd)
	governanceCmd.AddCommand(governanceSetCmd)

	// versions create flags
	versionsCreateCmd.Flags().StringVar(&versionLabel, "label", "", "Version label (required)")
	versionsCreateCmd.Flags().StringVar(&versionReason, "reason", "", "Reason for creating version")
	_ = versionsCreateCmd.MarkFlagRequired("label")

	versionsCmd.AddCommand(versionsListCmd)
	versionsCmd.AddCommand(versionsCreateCmd)

	// promote flags
	promoteCmd.Flags().StringVar(&promoteFrom, "from", "", "Source environment (required)")
	promoteCmd.Flags().StringVar(&promoteTo, "to", "", "Target environment (required)")
	_ = promoteCmd.MarkFlagRequired("from")
	_ = promoteCmd.MarkFlagRequired("to")

	// rollback flags
	rollbackCmd.Flags().StringVar(&rollbackEnv, "env", "", "Target environment (required)")
	rollbackCmd.Flags().StringVar(&rollbackVersion, "version", "", "Target version ID (required)")
	_ = rollbackCmd.MarkFlagRequired("env")
	_ = rollbackCmd.MarkFlagRequired("version")

	// Register governance commands with root
	rootCmd.AddCommand(governanceCmd)
	rootCmd.AddCommand(versionsCmd)
	rootCmd.AddCommand(promoteCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(bindingsCmd)
	rootCmd.AddCommand(historyCmd)
}

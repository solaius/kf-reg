package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var approvalsCmd = &cobra.Command{
	Use:   "approvals",
	Short: "Manage approval requests",
}

var approvalsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending approval requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()

		path := governanceAPIBase + "/approvals?status=pending"

		var result struct {
			Requests []struct {
				ID        string `json:"id"`
				AssetRef  struct {
					Plugin string `json:"plugin"`
					Kind   string `json:"kind"`
					Name   string `json:"name"`
				} `json:"assetRef"`
				Action    string `json:"action"`
				Status    string `json:"status"`
				Requester string `json:"requester"`
				PolicyID  string `json:"policyId"`
				CreatedAt string `json:"createdAt"`
			} `json:"requests"`
			TotalSize int `json:"totalSize"`
		}
		if err := client.getJSON(path, &result); err != nil {
			return fmt.Errorf("failed to list approvals: %w", err)
		}

		if outputFmt == "json" || outputFmt == "yaml" {
			return printOutput(result)
		}

		headers := []string{"ID", "Asset", "Action", "Status", "Requester", "Policy", "Created"}
		rows := make([][]string, 0, len(result.Requests))
		for _, r := range result.Requests {
			asset := fmt.Sprintf("%s/%s/%s", r.AssetRef.Plugin, r.AssetRef.Kind, r.AssetRef.Name)
			rows = append(rows, []string{
				truncate(r.ID, 12),
				asset,
				r.Action,
				r.Status,
				r.Requester,
				r.PolicyID,
				r.CreatedAt,
			})
		}
		printTable(headers, rows)
		fmt.Printf("Total: %d\n", result.TotalSize)
		return nil
	},
}

var approvalsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get approval request details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()

		path := fmt.Sprintf("%s/approvals/%s", governanceAPIBase, args[0])

		result, err := client.getRaw(path)
		if err != nil {
			return fmt.Errorf("failed to get approval: %w", err)
		}

		return printOutput(result)
	},
}

var approvalComment string

var approvalsApproveCmd = &cobra.Command{
	Use:   "approve <id>",
	Short: "Approve a pending approval request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()

		path := fmt.Sprintf("%s/approvals/%s/decisions", governanceAPIBase, args[0])

		body := map[string]any{
			"verdict": "approve",
			"comment": approvalComment,
		}

		var result map[string]any
		if err := client.postJSON(path, body, &result); err != nil {
			return fmt.Errorf("failed to approve: %w", err)
		}

		return printOutput(result)
	},
}

var rejectComment string

var approvalsRejectCmd = &cobra.Command{
	Use:   "reject <id>",
	Short: "Reject a pending approval request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()

		path := fmt.Sprintf("%s/approvals/%s/decisions", governanceAPIBase, args[0])

		body := map[string]any{
			"verdict": "deny",
			"comment": rejectComment,
		}

		var result map[string]any
		if err := client.postJSON(path, body, &result); err != nil {
			return fmt.Errorf("failed to reject: %w", err)
		}

		return printOutput(result)
	},
}

var cancelReason string

var approvalsCancelCmd = &cobra.Command{
	Use:   "cancel <id>",
	Short: "Cancel a pending approval request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newClient()

		path := fmt.Sprintf("%s/approvals/%s/cancel", governanceAPIBase, args[0])

		body := map[string]any{
			"reason": cancelReason,
		}

		var result map[string]any
		if err := client.postJSON(path, body, &result); err != nil {
			return fmt.Errorf("failed to cancel: %w", err)
		}

		return printOutput(result)
	},
}

func init() {
	approvalsApproveCmd.Flags().StringVar(&approvalComment, "comment", "", "Approval comment")
	approvalsRejectCmd.Flags().StringVar(&rejectComment, "comment", "", "Rejection reason")
	approvalsCancelCmd.Flags().StringVar(&cancelReason, "comment", "", "Cancellation reason")

	approvalsCmd.AddCommand(approvalsListCmd)
	approvalsCmd.AddCommand(approvalsGetCmd)
	approvalsCmd.AddCommand(approvalsApproveCmd)
	approvalsCmd.AddCommand(approvalsRejectCmd)
	approvalsCmd.AddCommand(approvalsCancelCmd)

	rootCmd.AddCommand(approvalsCmd)
}

package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// buildEntityActionCommand creates a subcommand for a specific action on an entity.
func buildEntityActionCommand(p pluginInfo, entity entityCaps, actionID string) *cobra.Command {
	// Find the action definition.
	var actionInfo *actionDef
	if p.CapabilitiesV2 != nil {
		for i := range p.CapabilitiesV2.Actions {
			if p.CapabilitiesV2.Actions[i].ID == actionID {
				actionInfo = &p.CapabilitiesV2.Actions[i]
				break
			}
		}
	}

	if actionInfo == nil {
		return nil
	}

	var dryRun bool
	var paramsStr string

	cmd := &cobra.Command{
		Use:   actionID + " [entity-name]",
		Short: actionInfo.Description,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()

			// Build the action endpoint.
			// Convention: basePath/entities/{name}:action
			path := entity.Endpoints.Action
			if path == "" {
				// Fallback: construct from the get endpoint.
				path = strings.Replace(entity.Endpoints.Get, "{name}", args[0], 1) + ":action"
			} else {
				path = strings.Replace(path, "{name}", args[0], 1)
			}

			// Parse params.
			params := make(map[string]any)
			if paramsStr != "" {
				for _, kv := range strings.Split(paramsStr, ",") {
					parts := strings.SplitN(kv, "=", 2)
					if len(parts) == 2 {
						params[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
					}
				}
			}

			req := actionRequest{
				ActionID:   actionID,
				TargetName: args[0],
				Params:     params,
				DryRun:     dryRun,
			}

			var resp actionResponse
			if err := client.postJSON(path, req, &resp); err != nil {
				return fmt.Errorf("action %q failed: %w", actionID, err)
			}

			if outputFmt == "json" || outputFmt == "yaml" {
				return printOutput(resp)
			}

			if resp.DryRun {
				fmt.Printf("[dry-run] ")
			}
			fmt.Printf("Action %q on %q: %s\n", resp.ActionID, args[0], resp.Status)
			if resp.Message != "" {
				fmt.Printf("  %s\n", resp.Message)
			}

			return nil
		},
	}

	if actionInfo.SupportsDryRun {
		cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview the action without applying changes")
	}
	cmd.Flags().StringVar(&paramsStr, "params", "", "Action parameters as key=value pairs (comma-separated)")

	return cmd
}

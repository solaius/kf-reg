package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// mcpServer mirrors the JSON structure returned by the MCP catalog API.
type mcpServer struct {
	ID                       string         `json:"id,omitempty"`
	Name                     string         `json:"name"`
	ExternalID               string         `json:"externalId,omitempty"`
	Description              string         `json:"description,omitempty"`
	CustomProperties         map[string]any `json:"customProperties,omitempty"`
	CreateTimeSinceEpoch     string         `json:"createTimeSinceEpoch,omitempty"`
	LastUpdateTimeSinceEpoch string         `json:"lastUpdateTimeSinceEpoch,omitempty"`
	ServerUrl                string         `json:"serverUrl,omitempty"`
	TransportType            string         `json:"transportType,omitempty"`
	ToolCount                *int32         `json:"toolCount,omitempty"`
	ResourceCount            *int32         `json:"resourceCount,omitempty"`
	PromptCount              *int32         `json:"promptCount,omitempty"`
	DeploymentMode           string         `json:"deploymentMode,omitempty"`
	Image                    string         `json:"image,omitempty"`
	Endpoint                 string         `json:"endpoint,omitempty"`
	SupportedTransports      string         `json:"supportedTransports,omitempty"`
	License                  string         `json:"license,omitempty"`
	Verified                 *bool          `json:"verified,omitempty"`
	Certified                *bool          `json:"certified,omitempty"`
	Provider                 string         `json:"provider,omitempty"`
	Logo                     string         `json:"logo,omitempty"`
	Category                 string         `json:"category,omitempty"`
}

type mcpServerList struct {
	Items         []mcpServer `json:"items"`
	NextPageToken string      `json:"nextPageToken,omitempty"`
	Size          int32       `json:"size"`
}

func newMcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Browse MCP server catalog",
		Long:  "List, get, and search MCP servers from the catalog.",
	}

	cmd.AddCommand(newMcpListCmd())
	cmd.AddCommand(newMcpGetCmd())
	cmd.AddCommand(newMcpSearchCmd())

	return cmd
}

// --- mcp list ---

func newMcpListCmd() *cobra.Command {
	var (
		apiVersion     string
		deploymentMode string
		transport      string
		provider       string
		category       string
		pageSize       int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List MCP servers",
		Long:  "List MCP servers from the catalog with optional filters.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMcpList(apiVersion, deploymentMode, transport, provider, category, pageSize)
		},
	}

	cmd.Flags().StringVar(&apiVersion, "api-version", "v1alpha1", "API version")
	cmd.Flags().StringVar(&deploymentMode, "deployment-mode", "", "Filter by deployment mode (local, remote)")
	cmd.Flags().StringVar(&transport, "transport", "", "Filter by transport type (stdio, http, sse)")
	cmd.Flags().StringVar(&provider, "provider", "", "Filter by provider name")
	cmd.Flags().StringVar(&category, "category", "", "Filter by category")
	cmd.Flags().IntVar(&pageSize, "page-size", 100, "Maximum number of results")

	return cmd
}

func runMcpList(version, deploymentMode, transport, provider, category string, pageSize int) error {
	// Build filterQuery from flags
	var filters []string
	if deploymentMode != "" {
		filters = append(filters, fmt.Sprintf("deploymentMode='%s'", deploymentMode))
	}
	if transport != "" {
		filters = append(filters, fmt.Sprintf("transportType='%s'", transport))
	}
	if provider != "" {
		filters = append(filters, fmt.Sprintf("provider='%s'", provider))
	}
	if category != "" {
		filters = append(filters, fmt.Sprintf("category='%s'", category))
	}

	params := url.Values{}
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))
	if len(filters) > 0 {
		params.Set("filterQuery", strings.Join(filters, " AND "))
	}

	path := fmt.Sprintf("/api/mcp_catalog/%s/mcpservers?%s", version, params.Encode())

	body, err := globalClient.doRequest("GET", path, nil)
	if err != nil {
		return err
	}

	var resp mcpServerList
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	headers := []string{"Name", "Mode", "Transport", "Provider", "Category", "Tools", "Verified"}
	var rows [][]string
	for _, s := range resp.Items {
		verified := "-"
		if s.Verified != nil {
			if *s.Verified {
				verified = "yes"
			} else {
				verified = "no"
			}
		}
		toolCount := "-"
		if s.ToolCount != nil {
			toolCount = fmt.Sprintf("%d", *s.ToolCount)
		}
		rows = append(rows, []string{
			s.Name,
			s.DeploymentMode,
			s.TransportType,
			s.Provider,
			s.Category,
			toolCount,
			verified,
		})
	}

	return printOutput(os.Stdout, format, resp, headers, rows)
}

// --- mcp get ---

func newMcpGetCmd() *cobra.Command {
	var apiVersion string

	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get MCP server details",
		Long:  "Show full details for a specific MCP server by name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMcpGet(apiVersion, args[0])
		},
	}

	cmd.Flags().StringVar(&apiVersion, "api-version", "v1alpha1", "API version")

	return cmd
}

func runMcpGet(version, name string) error {
	path := fmt.Sprintf("/api/mcp_catalog/%s/mcpservers/%s", version, url.PathEscape(name))

	body, err := globalClient.doRequest("GET", path, nil)
	if err != nil {
		return err
	}

	var server mcpServer
	if err := json.Unmarshal(body, &server); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	if format == outputTable {
		fmt.Fprintf(os.Stdout, "Name:             %s\n", server.Name)
		if server.Description != "" {
			fmt.Fprintf(os.Stdout, "Description:      %s\n", server.Description)
		}
		fmt.Fprintf(os.Stdout, "Deployment Mode:  %s\n", server.DeploymentMode)
		fmt.Fprintf(os.Stdout, "Transport:        %s\n", server.TransportType)
		if server.ServerUrl != "" {
			fmt.Fprintf(os.Stdout, "Server URL:       %s\n", server.ServerUrl)
		}
		if server.Image != "" {
			fmt.Fprintf(os.Stdout, "Image:            %s\n", server.Image)
		}
		if server.Endpoint != "" {
			fmt.Fprintf(os.Stdout, "Endpoint:         %s\n", server.Endpoint)
		}
		if server.SupportedTransports != "" {
			fmt.Fprintf(os.Stdout, "Transports:       %s\n", server.SupportedTransports)
		}
		if server.Provider != "" {
			fmt.Fprintf(os.Stdout, "Provider:         %s\n", server.Provider)
		}
		if server.Category != "" {
			fmt.Fprintf(os.Stdout, "Category:         %s\n", server.Category)
		}
		if server.License != "" {
			fmt.Fprintf(os.Stdout, "License:          %s\n", server.License)
		}
		if server.ToolCount != nil {
			fmt.Fprintf(os.Stdout, "Tools:            %d\n", *server.ToolCount)
		}
		if server.ResourceCount != nil {
			fmt.Fprintf(os.Stdout, "Resources:        %d\n", *server.ResourceCount)
		}
		if server.PromptCount != nil {
			fmt.Fprintf(os.Stdout, "Prompts:          %d\n", *server.PromptCount)
		}
		if server.Verified != nil {
			fmt.Fprintf(os.Stdout, "Verified:         %v\n", *server.Verified)
		}
		if server.Certified != nil {
			fmt.Fprintf(os.Stdout, "Certified:        %v\n", *server.Certified)
		}
		return nil
	}

	return printOutput(os.Stdout, format, server, nil, nil)
}

// --- mcp search ---

func newMcpSearchCmd() *cobra.Command {
	var (
		apiVersion string
		pageSize   int
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search MCP servers",
		Long:  "Search MCP servers by name or description using free-text search.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMcpSearch(apiVersion, args[0], pageSize)
		},
	}

	cmd.Flags().StringVar(&apiVersion, "api-version", "v1alpha1", "API version")
	cmd.Flags().IntVar(&pageSize, "page-size", 100, "Maximum number of results")

	return cmd
}

func runMcpSearch(version, query string, pageSize int) error {
	params := url.Values{}
	params.Set("q", query)
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))

	path := fmt.Sprintf("/api/mcp_catalog/%s/mcpservers?%s", version, params.Encode())

	body, err := globalClient.doRequest("GET", path, nil)
	if err != nil {
		return err
	}

	var resp mcpServerList
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	format, err := parseOutputFormat(outputFlag)
	if err != nil {
		return err
	}

	headers := []string{"Name", "Mode", "Provider", "Category", "Description"}
	var rows [][]string
	for _, s := range resp.Items {
		rows = append(rows, []string{
			s.Name,
			s.DeploymentMode,
			s.Provider,
			s.Category,
			truncate(s.Description, 50),
		})
	}

	return printOutput(os.Stdout, format, resp, headers, rows)
}

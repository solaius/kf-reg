// Package main provides the catalog CLI binary for managing the catalog server.
// This is a management-plane tool that communicates with the catalog-server HTTP API.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	version = "dev"

	// Global flags
	serverURL    string
	outputFlag   string
	roleFlag     string
	globalClient *catalogClient
)

// catalogClient wraps an HTTP client and the server base URL.
type catalogClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// newCatalogClient creates a new client targeting the given server URL.
func newCatalogClient(baseURL string) *catalogClient {
	return &catalogClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: slog.Default(),
	}
}

// doRequest performs an HTTP request and returns the response body bytes.
// It returns an error if the status code indicates a failure.
func (c *catalogClient) doRequest(method, path string, body io.Reader) ([]byte, error) {
	url := c.baseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if roleFlag != "" {
		req.Header.Set("X-Role", roleFlag)
	}

	c.logger.Debug("sending request", "method", method, "url", url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to catalog server at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Try to extract error message from JSON response
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, errResp.Message)
		}
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "catalog",
		Short: "CLI for the Kubeflow Model Registry catalog management plane",
		Long: `catalog is a command-line tool for managing the catalog server.

It provides commands for inspecting plugins, managing data sources,
triggering refreshes, and viewing server diagnostics.

The CLI communicates with the catalog-server HTTP API.`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize the global client
			globalClient = newCatalogClient(serverURL)

			// Set up slog level based on verbosity (could be extended with a --verbose flag)
			return nil
		},
		SilenceUsage: true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:8081", "Catalog server URL")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "table", "Output format: table, json, yaml")
	rootCmd.PersistentFlags().StringVar(&roleFlag, "role", "", "Role for RBAC (viewer, operator); sets X-Role header")

	// Register subcommands
	rootCmd.AddCommand(newPluginsCmd())
	rootCmd.AddCommand(newSourcesCmd())
	rootCmd.AddCommand(newRefreshCmd())
	rootCmd.AddCommand(newStatusCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

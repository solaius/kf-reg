package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newBumpVersionCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "bump-version [major|minor|patch]",
		Short: "Bump the plugin version in plugin.yaml",
		Long: `Bump the semver version in plugin.yaml by the specified part.

Examples:
  catalog-gen bump-version patch    # 0.1.0 -> 0.1.1
  catalog-gen bump-version minor    # 0.1.0 -> 0.2.0
  catalog-gen bump-version major    # 0.1.0 -> 1.0.0`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			part := args[0]
			return runBumpVersion(dir, part)
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Plugin directory containing plugin.yaml")

	return cmd
}

func runBumpVersion(dir, part string) error {
	pluginPath := filepath.Join(dir, "plugin.yaml")

	spec, err := plugin.LoadPluginMetadata(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to load plugin.yaml: %w", err)
	}

	oldVersion := spec.Spec.Version
	newVersion, err := plugin.BumpVersion(oldVersion, part)
	if err != nil {
		return err
	}

	spec.Spec.Version = newVersion

	data, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal plugin.yaml: %w", err)
	}

	if err := os.WriteFile(pluginPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write plugin.yaml: %w", err)
	}

	fmt.Printf("Bumped version: %s -> %s\n", oldVersion, newVersion)
	return nil
}

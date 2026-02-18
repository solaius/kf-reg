package governance

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadGovernanceConfig loads governance configuration from a YAML file.
// If the file does not exist, default configuration is returned.
func LoadGovernanceConfig(path string) (*GovernanceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultGovernanceConfig(), nil
		}
		return nil, fmt.Errorf("read governance config: %w", err)
	}

	var cfg GovernanceConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse governance config: %w", err)
	}

	return &cfg, nil
}

// DefaultGovernanceConfig returns the default governance configuration.
func DefaultGovernanceConfig() *GovernanceConfig {
	return &GovernanceConfig{
		Environments:   []string{"dev", "stage", "prod"},
		TrustedSources: []string{},
		AuditRetention: AuditRetentionConfig{
			Days: 90,
		},
	}
}

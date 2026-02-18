package audit

import (
	"os"
	"strconv"
)

// AuditConfig controls audit behavior.
type AuditConfig struct {
	RetentionDays int  // Default 90
	LogDenied     bool // Whether to log denied (403) actions
	Enabled       bool // Whether audit middleware is active
}

// DefaultAuditConfig returns the default configuration.
func DefaultAuditConfig() *AuditConfig {
	return &AuditConfig{
		RetentionDays: 90,
		LogDenied:     true,
		Enabled:       true,
	}
}

// AuditConfigFromEnv loads config from environment variables.
// CATALOG_AUDIT_RETENTION_DAYS, CATALOG_AUDIT_LOG_DENIED, CATALOG_AUDIT_ENABLED
func AuditConfigFromEnv() *AuditConfig {
	cfg := DefaultAuditConfig()

	if v := os.Getenv("CATALOG_AUDIT_RETENTION_DAYS"); v != "" {
		if days, err := strconv.Atoi(v); err == nil && days > 0 {
			cfg.RetentionDays = days
		}
	}

	if v := os.Getenv("CATALOG_AUDIT_LOG_DENIED"); v != "" {
		cfg.LogDenied, _ = strconv.ParseBool(v)
	}

	if v := os.Getenv("CATALOG_AUDIT_ENABLED"); v != "" {
		cfg.Enabled, _ = strconv.ParseBool(v)
	}

	return cfg
}

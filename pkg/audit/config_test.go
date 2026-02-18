package audit

import (
	"os"
	"testing"
)

func TestDefaultAuditConfig(t *testing.T) {
	cfg := DefaultAuditConfig()

	if cfg.RetentionDays != 90 {
		t.Errorf("expected RetentionDays 90, got %d", cfg.RetentionDays)
	}
	if !cfg.LogDenied {
		t.Error("expected LogDenied to be true")
	}
	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}
}

func TestAuditConfigFromEnv(t *testing.T) {
	tests := []struct {
		name          string
		envs          map[string]string
		wantRetention int
		wantLogDenied bool
		wantEnabled   bool
	}{
		{
			name:          "defaults",
			envs:          map[string]string{},
			wantRetention: 90,
			wantLogDenied: true,
			wantEnabled:   true,
		},
		{
			name: "custom values",
			envs: map[string]string{
				"CATALOG_AUDIT_RETENTION_DAYS": "30",
				"CATALOG_AUDIT_LOG_DENIED":     "false",
				"CATALOG_AUDIT_ENABLED":        "false",
			},
			wantRetention: 30,
			wantLogDenied: false,
			wantEnabled:   false,
		},
		{
			name: "invalid retention falls back to default",
			envs: map[string]string{
				"CATALOG_AUDIT_RETENTION_DAYS": "invalid",
			},
			wantRetention: 90,
			wantLogDenied: true,
			wantEnabled:   true,
		},
		{
			name: "negative retention falls back to default",
			envs: map[string]string{
				"CATALOG_AUDIT_RETENTION_DAYS": "-5",
			},
			wantRetention: 90,
			wantLogDenied: true,
			wantEnabled:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars.
			for k, v := range tt.envs {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envs {
					os.Unsetenv(k)
				}
			}()

			cfg := AuditConfigFromEnv()

			if cfg.RetentionDays != tt.wantRetention {
				t.Errorf("RetentionDays = %d, want %d", cfg.RetentionDays, tt.wantRetention)
			}
			if cfg.LogDenied != tt.wantLogDenied {
				t.Errorf("LogDenied = %v, want %v", cfg.LogDenied, tt.wantLogDenied)
			}
			if cfg.Enabled != tt.wantEnabled {
				t.Errorf("Enabled = %v, want %v", cfg.Enabled, tt.wantEnabled)
			}
		})
	}
}

package ha

import (
	"os"
	"testing"
	"time"
)

func TestDefaultHAConfig(t *testing.T) {
	os.Unsetenv("POD_NAMESPACE")

	cfg := DefaultHAConfig()

	if cfg.LeaderElectionEnabled {
		t.Error("LeaderElectionEnabled should be false by default")
	}
	if cfg.LeaseName != "catalog-server-leader" {
		t.Errorf("LeaseName = %q, want %q", cfg.LeaseName, "catalog-server-leader")
	}
	if cfg.LeaseNamespace != "catalog-system" {
		t.Errorf("LeaseNamespace = %q, want %q", cfg.LeaseNamespace, "catalog-system")
	}
	if cfg.LeaseDuration != 15*time.Second {
		t.Errorf("LeaseDuration = %v, want %v", cfg.LeaseDuration, 15*time.Second)
	}
	if cfg.RenewDeadline != 10*time.Second {
		t.Errorf("RenewDeadline = %v, want %v", cfg.RenewDeadline, 10*time.Second)
	}
	if cfg.RetryPeriod != 2*time.Second {
		t.Errorf("RetryPeriod = %v, want %v", cfg.RetryPeriod, 2*time.Second)
	}
	if !cfg.MigrationLockEnabled {
		t.Error("MigrationLockEnabled should be true by default")
	}
}

func TestDefaultHAConfig_NamespaceFromEnv(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "my-namespace")

	cfg := DefaultHAConfig()
	if cfg.LeaseNamespace != "my-namespace" {
		t.Errorf("LeaseNamespace = %q, want %q", cfg.LeaseNamespace, "my-namespace")
	}
}

func TestDefaultHAConfig_IdentityFromPodName(t *testing.T) {
	t.Setenv("POD_NAME", "catalog-server-abc-123")

	cfg := DefaultHAConfig()
	if cfg.Identity != "catalog-server-abc-123" {
		t.Errorf("Identity = %q, want %q", cfg.Identity, "catalog-server-abc-123")
	}
}

func TestHAConfigFromEnv(t *testing.T) {
	tests := []struct {
		name  string
		envs  map[string]string
		check func(t *testing.T, cfg *HAConfig)
	}{
		{
			name: "defaults when no env vars set",
			envs: map[string]string{},
			check: func(t *testing.T, cfg *HAConfig) {
				if cfg.LeaderElectionEnabled {
					t.Error("expected LeaderElectionEnabled=false")
				}
				if cfg.LeaseName != "catalog-server-leader" {
					t.Errorf("LeaseName = %q, want %q", cfg.LeaseName, "catalog-server-leader")
				}
			},
		},
		{
			name: "enabled via env",
			envs: map[string]string{
				"CATALOG_LEADER_ELECTION_ENABLED": "true",
			},
			check: func(t *testing.T, cfg *HAConfig) {
				if !cfg.LeaderElectionEnabled {
					t.Error("expected LeaderElectionEnabled=true")
				}
			},
		},
		{
			name: "enabled via 1",
			envs: map[string]string{
				"CATALOG_LEADER_ELECTION_ENABLED": "1",
			},
			check: func(t *testing.T, cfg *HAConfig) {
				if !cfg.LeaderElectionEnabled {
					t.Error("expected LeaderElectionEnabled=true")
				}
			},
		},
		{
			name: "custom lease name",
			envs: map[string]string{
				"CATALOG_LEADER_LEASE_NAME": "my-lease",
			},
			check: func(t *testing.T, cfg *HAConfig) {
				if cfg.LeaseName != "my-lease" {
					t.Errorf("LeaseName = %q, want %q", cfg.LeaseName, "my-lease")
				}
			},
		},
		{
			name: "custom namespace",
			envs: map[string]string{
				"CATALOG_LEADER_LEASE_NAMESPACE": "prod",
			},
			check: func(t *testing.T, cfg *HAConfig) {
				if cfg.LeaseNamespace != "prod" {
					t.Errorf("LeaseNamespace = %q, want %q", cfg.LeaseNamespace, "prod")
				}
			},
		},
		{
			name: "custom durations",
			envs: map[string]string{
				"CATALOG_LEADER_LEASE_DURATION": "30",
				"CATALOG_LEADER_RENEW_DEADLINE": "20",
				"CATALOG_LEADER_RETRY_PERIOD":   "5",
			},
			check: func(t *testing.T, cfg *HAConfig) {
				if cfg.LeaseDuration != 30*time.Second {
					t.Errorf("LeaseDuration = %v, want %v", cfg.LeaseDuration, 30*time.Second)
				}
				if cfg.RenewDeadline != 20*time.Second {
					t.Errorf("RenewDeadline = %v, want %v", cfg.RenewDeadline, 20*time.Second)
				}
				if cfg.RetryPeriod != 5*time.Second {
					t.Errorf("RetryPeriod = %v, want %v", cfg.RetryPeriod, 5*time.Second)
				}
			},
		},
		{
			name: "migration lock disabled",
			envs: map[string]string{
				"CATALOG_MIGRATION_LOCK_ENABLED": "false",
			},
			check: func(t *testing.T, cfg *HAConfig) {
				if cfg.MigrationLockEnabled {
					t.Error("expected MigrationLockEnabled=false")
				}
			},
		},
		{
			name: "pod name as identity",
			envs: map[string]string{
				"POD_NAME": "pod-xyz",
			},
			check: func(t *testing.T, cfg *HAConfig) {
				if cfg.Identity != "pod-xyz" {
					t.Errorf("Identity = %q, want %q", cfg.Identity, "pod-xyz")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars.
			for _, key := range []string{
				"CATALOG_LEADER_ELECTION_ENABLED",
				"CATALOG_LEADER_LEASE_NAME",
				"CATALOG_LEADER_LEASE_NAMESPACE",
				"CATALOG_LEADER_LEASE_DURATION",
				"CATALOG_LEADER_RENEW_DEADLINE",
				"CATALOG_LEADER_RETRY_PERIOD",
				"CATALOG_MIGRATION_LOCK_ENABLED",
				"POD_NAME",
			} {
				t.Setenv(key, "")
				os.Unsetenv(key)
			}
			// Set test env vars.
			for k, v := range tt.envs {
				t.Setenv(k, v)
			}

			cfg := HAConfigFromEnv()
			tt.check(t, cfg)
		})
	}
}

package jobs

import (
	"os"
	"testing"
	"time"
)

func TestDefaultJobConfig(t *testing.T) {
	cfg := DefaultJobConfig()

	if cfg.Concurrency != 3 {
		t.Errorf("expected Concurrency 3, got %d", cfg.Concurrency)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", cfg.MaxRetries)
	}
	if cfg.PollInterval != 5*time.Second {
		t.Errorf("expected PollInterval 5s, got %v", cfg.PollInterval)
	}
	if cfg.ClaimTimeout != 10*time.Minute {
		t.Errorf("expected ClaimTimeout 10m, got %v", cfg.ClaimTimeout)
	}
	if cfg.RetentionDays != 7 {
		t.Errorf("expected RetentionDays 7, got %d", cfg.RetentionDays)
	}
	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}
}

func TestJobConfigFromEnv(t *testing.T) {
	tests := []struct {
		name            string
		envs            map[string]string
		wantConcurrency int
		wantMaxRetries  int
		wantEnabled     bool
	}{
		{
			name:            "defaults",
			envs:            map[string]string{},
			wantConcurrency: 3,
			wantMaxRetries:  3,
			wantEnabled:     true,
		},
		{
			name: "custom values",
			envs: map[string]string{
				"CATALOG_JOB_CONCURRENCY": "5",
				"CATALOG_JOB_MAX_RETRIES": "1",
				"CATALOG_JOB_ENABLED":     "false",
			},
			wantConcurrency: 5,
			wantMaxRetries:  1,
			wantEnabled:     false,
		},
		{
			name: "invalid concurrency falls back to default",
			envs: map[string]string{
				"CATALOG_JOB_CONCURRENCY": "invalid",
			},
			wantConcurrency: 3,
			wantMaxRetries:  3,
			wantEnabled:     true,
		},
		{
			name: "zero retries allowed",
			envs: map[string]string{
				"CATALOG_JOB_MAX_RETRIES": "0",
			},
			wantConcurrency: 3,
			wantMaxRetries:  0,
			wantEnabled:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envs {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envs {
					os.Unsetenv(k)
				}
			}()

			cfg := JobConfigFromEnv()

			if cfg.Concurrency != tt.wantConcurrency {
				t.Errorf("Concurrency = %d, want %d", cfg.Concurrency, tt.wantConcurrency)
			}
			if cfg.MaxRetries != tt.wantMaxRetries {
				t.Errorf("MaxRetries = %d, want %d", cfg.MaxRetries, tt.wantMaxRetries)
			}
			if cfg.Enabled != tt.wantEnabled {
				t.Errorf("Enabled = %v, want %v", cfg.Enabled, tt.wantEnabled)
			}
		})
	}
}

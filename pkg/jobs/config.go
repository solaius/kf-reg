package jobs

import (
	"os"
	"strconv"
	"time"
)

// JobConfig controls job queue and worker behavior.
type JobConfig struct {
	Concurrency    int           // Max concurrent workers. Default 3.
	MaxRetries     int           // Max retry attempts per job. Default 3.
	PollInterval   time.Duration // How often workers poll for new jobs. Default 5s.
	ClaimTimeout   time.Duration // Max time a job can be in "running" before considered stuck. Default 10m.
	RetentionDays  int           // How long to keep completed/failed jobs. Default 7.
	Enabled        bool          // Whether the job system is active. Default true.
}

// DefaultJobConfig returns the default job configuration.
func DefaultJobConfig() *JobConfig {
	return &JobConfig{
		Concurrency:   3,
		MaxRetries:    3,
		PollInterval:  5 * time.Second,
		ClaimTimeout:  10 * time.Minute,
		RetentionDays: 7,
		Enabled:       true,
	}
}

// JobConfigFromEnv loads config from environment variables.
// CATALOG_JOB_CONCURRENCY, CATALOG_JOB_MAX_RETRIES, CATALOG_JOB_POLL_INTERVAL_SECONDS,
// CATALOG_JOB_CLAIM_TIMEOUT_MINUTES, CATALOG_JOB_RETENTION_DAYS, CATALOG_JOB_ENABLED
func JobConfigFromEnv() *JobConfig {
	cfg := DefaultJobConfig()

	if v := os.Getenv("CATALOG_JOB_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Concurrency = n
		}
	}

	if v := os.Getenv("CATALOG_JOB_MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.MaxRetries = n
		}
	}

	if v := os.Getenv("CATALOG_JOB_POLL_INTERVAL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.PollInterval = time.Duration(n) * time.Second
		}
	}

	if v := os.Getenv("CATALOG_JOB_CLAIM_TIMEOUT_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ClaimTimeout = time.Duration(n) * time.Minute
		}
	}

	if v := os.Getenv("CATALOG_JOB_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.RetentionDays = n
		}
	}

	if v := os.Getenv("CATALOG_JOB_ENABLED"); v != "" {
		cfg.Enabled, _ = strconv.ParseBool(v)
	}

	return cfg
}

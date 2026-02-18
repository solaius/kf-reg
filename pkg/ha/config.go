// Package ha provides high-availability primitives for running the catalog
// server with multiple replicas: migration locking and Kubernetes
// Lease-based leader election.
package ha

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// HAConfig holds configuration for high-availability features.
type HAConfig struct {
	// LeaderElectionEnabled controls whether Kubernetes Lease-based leader
	// election is active. When false, the instance behaves as the sole
	// leader (suitable for single-replica deployments).
	LeaderElectionEnabled bool

	// LeaseName is the name of the Kubernetes Lease resource used for
	// leader election.
	LeaseName string

	// LeaseNamespace is the namespace of the Lease resource.
	LeaseNamespace string

	// LeaseDuration is the duration that non-leader candidates will wait
	// before trying to acquire the lease.
	LeaseDuration time.Duration

	// RenewDeadline is the duration that the acting leader will retry
	// refreshing the lease before giving up.
	RenewDeadline time.Duration

	// RetryPeriod is the duration between leader election retries.
	RetryPeriod time.Duration

	// MigrationLockEnabled controls whether database migration locking
	// is used to prevent concurrent schema changes.
	MigrationLockEnabled bool

	// Identity is the unique identity of this instance for leader election.
	// Defaults to the pod name (from POD_NAME env var or hostname).
	Identity string
}

// DefaultHAConfig returns an HAConfig with sensible defaults.
func DefaultHAConfig() *HAConfig {
	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		ns = "catalog-system"
	}
	return &HAConfig{
		LeaderElectionEnabled: false,
		LeaseName:             "catalog-server-leader",
		LeaseNamespace:        ns,
		LeaseDuration:         15 * time.Second,
		RenewDeadline:         10 * time.Second,
		RetryPeriod:           2 * time.Second,
		MigrationLockEnabled:  true,
		Identity:              defaultIdentity(),
	}
}

// HAConfigFromEnv reads HA configuration from environment variables,
// falling back to defaults for any unset variable.
//
// Environment variables:
//   - CATALOG_LEADER_ELECTION_ENABLED: "true" or "false" (default: "false")
//   - CATALOG_LEADER_LEASE_NAME: Lease resource name (default: "catalog-server-leader")
//   - CATALOG_LEADER_LEASE_NAMESPACE: Lease namespace (default from POD_NAMESPACE or "catalog-system")
//   - CATALOG_LEADER_LEASE_DURATION: seconds (default: 15)
//   - CATALOG_LEADER_RENEW_DEADLINE: seconds (default: 10)
//   - CATALOG_LEADER_RETRY_PERIOD: seconds (default: 2)
//   - CATALOG_MIGRATION_LOCK_ENABLED: "true" or "false" (default: "true")
//   - POD_NAME: pod identity for leader election
func HAConfigFromEnv() *HAConfig {
	cfg := DefaultHAConfig()

	if v := os.Getenv("CATALOG_LEADER_ELECTION_ENABLED"); v != "" {
		cfg.LeaderElectionEnabled = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("CATALOG_LEADER_LEASE_NAME"); v != "" {
		cfg.LeaseName = v
	}
	if v := os.Getenv("CATALOG_LEADER_LEASE_NAMESPACE"); v != "" {
		cfg.LeaseNamespace = v
	}
	if v := os.Getenv("CATALOG_LEADER_LEASE_DURATION"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			cfg.LeaseDuration = time.Duration(secs) * time.Second
		}
	}
	if v := os.Getenv("CATALOG_LEADER_RENEW_DEADLINE"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			cfg.RenewDeadline = time.Duration(secs) * time.Second
		}
	}
	if v := os.Getenv("CATALOG_LEADER_RETRY_PERIOD"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			cfg.RetryPeriod = time.Duration(secs) * time.Second
		}
	}
	if v := os.Getenv("CATALOG_MIGRATION_LOCK_ENABLED"); v != "" {
		cfg.MigrationLockEnabled = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("POD_NAME"); v != "" {
		cfg.Identity = v
	}

	return cfg
}

func defaultIdentity() string {
	if v := os.Getenv("POD_NAME"); v != "" {
		return v
	}
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

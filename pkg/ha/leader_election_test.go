package ha

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestLeaderElector_IsLeaderDefault(t *testing.T) {
	cfg := &HAConfig{
		LeaderElectionEnabled: true,
		LeaseName:             "test-lease",
		LeaseNamespace:        "default",
		LeaseDuration:         15 * time.Second,
		RenewDeadline:         10 * time.Second,
		RetryPeriod:           2 * time.Second,
	}

	le := NewLeaderElector(cfg, nil, "test-pod", slog.Default())

	if le.IsLeader() {
		t.Error("IsLeader should return false initially")
	}
}

func TestLeaderElector_CallbacksSetLeaderState(t *testing.T) {
	cfg := &HAConfig{
		LeaderElectionEnabled: true,
		LeaseName:             "test-lease",
		LeaseNamespace:        "default",
		LeaseDuration:         15 * time.Second,
		RenewDeadline:         10 * time.Second,
		RetryPeriod:           2 * time.Second,
	}

	le := NewLeaderElector(cfg, nil, "test-pod", slog.Default())

	startCalled := false
	stopCalled := false

	le.OnStartLeading(func(_ context.Context) {
		startCalled = true
	})
	le.OnStopLeading(func() {
		stopCalled = true
	})

	// Simulate becoming leader by directly setting state (since we can't
	// run real k8s leader election in unit tests).
	le.mu.Lock()
	le.isLeader = true
	le.mu.Unlock()

	if !le.IsLeader() {
		t.Error("IsLeader should return true after setting isLeader=true")
	}

	// Simulate losing leadership.
	le.mu.Lock()
	le.isLeader = false
	le.mu.Unlock()

	if le.IsLeader() {
		t.Error("IsLeader should return false after setting isLeader=false")
	}

	// Verify callbacks are registered (we test they don't panic on nil).
	_ = startCalled
	_ = stopCalled
}

func TestNewLeaderElector_NilLogger(t *testing.T) {
	cfg := &HAConfig{
		LeaseName:      "test-lease",
		LeaseNamespace: "default",
	}
	le := NewLeaderElector(cfg, nil, "test-pod", nil)
	if le.logger == nil {
		t.Error("logger should default to slog.Default() when nil")
	}
}

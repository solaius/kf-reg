package ha

import (
	"context"
	"log/slog"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// LeaderElector manages Kubernetes Lease-based leader election for singleton
// background loops. Only the elected leader replica runs loops such as config
// reconciliation, audit retention, and job workers.
type LeaderElector struct {
	config   *HAConfig
	client   kubernetes.Interface
	identity string
	isLeader bool
	mu       sync.RWMutex
	logger   *slog.Logger
	onStart  func(ctx context.Context)
	onStop   func()
}

// NewLeaderElector creates a new LeaderElector. The identity should be unique
// per replica (typically the pod name or hostname).
func NewLeaderElector(cfg *HAConfig, client kubernetes.Interface, identity string, logger *slog.Logger) *LeaderElector {
	if logger == nil {
		logger = slog.Default()
	}
	return &LeaderElector{
		config:   cfg,
		client:   client,
		identity: identity,
		logger:   logger,
	}
}

// OnStartLeading registers a callback invoked when this instance becomes leader.
// The provided context is cancelled when leadership is lost.
func (le *LeaderElector) OnStartLeading(fn func(ctx context.Context)) {
	le.onStart = fn
}

// OnStopLeading registers a callback invoked when this instance loses leadership.
func (le *LeaderElector) OnStopLeading(fn func()) {
	le.onStop = fn
}

// IsLeader returns true if this instance is the current leader.
func (le *LeaderElector) IsLeader() bool {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.isLeader
}

// Run starts leader election. It blocks until the context is cancelled.
// When this instance becomes leader, it calls the OnStartLeading callback.
// When leadership is lost, it calls OnStopLeading.
func (le *LeaderElector) Run(ctx context.Context) {
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      le.config.LeaseName,
			Namespace: le.config.LeaseNamespace,
		},
		Client: le.client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: le.identity,
		},
	}

	le.logger.Info("starting leader election",
		"identity", le.identity,
		"lease", le.config.LeaseName,
		"namespace", le.config.LeaseNamespace,
		"leaseDuration", le.config.LeaseDuration,
		"renewDeadline", le.config.RenewDeadline,
		"retryPeriod", le.config.RetryPeriod,
	)

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   le.config.LeaseDuration,
		RenewDeadline:   le.config.RenewDeadline,
		RetryPeriod:     le.config.RetryPeriod,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				le.mu.Lock()
				le.isLeader = true
				le.mu.Unlock()
				le.logger.Info("elected as leader", "identity", le.identity)
				if le.onStart != nil {
					le.onStart(ctx)
				}
			},
			OnStoppedLeading: func() {
				le.mu.Lock()
				le.isLeader = false
				le.mu.Unlock()
				le.logger.Info("lost leadership", "identity", le.identity)
				if le.onStop != nil {
					le.onStop()
				}
			},
			OnNewLeader: func(identity string) {
				if identity != le.identity {
					le.logger.Info("new leader elected", "leader", identity)
				}
			},
		},
	})
}

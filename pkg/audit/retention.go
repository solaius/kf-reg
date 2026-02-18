package audit

import (
	"context"
	"log/slog"
	"time"

	"github.com/kubeflow/model-registry/pkg/catalog/governance"
)

// RetentionWorker periodically cleans up old audit events.
type RetentionWorker struct {
	store     *governance.AuditStore
	retention time.Duration
	interval  time.Duration
	logger    *slog.Logger
}

// NewRetentionWorker creates a new RetentionWorker.
// retentionDays controls how many days of events to keep.
// The worker runs daily by default.
func NewRetentionWorker(store *governance.AuditStore, retentionDays int, logger *slog.Logger) *RetentionWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &RetentionWorker{
		store:     store,
		retention: time.Duration(retentionDays) * 24 * time.Hour,
		interval:  24 * time.Hour,
		logger:    logger,
	}
}

// Run starts the retention worker. It runs until the context is cancelled.
func (w *RetentionWorker) Run(ctx context.Context) {
	if w.store == nil || w.retention <= 0 {
		w.logger.Info("audit retention worker disabled",
			"hasStore", w.store != nil,
			"retentionDays", int(w.retention.Hours()/24))
		return
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.Info("audit retention worker started",
		"retentionDays", int(w.retention.Hours()/24),
		"interval", w.interval.String())

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("audit retention worker stopped")
			return
		case <-ticker.C:
			w.cleanup()
		}
	}
}

// cleanup performs a single retention pass.
func (w *RetentionWorker) cleanup() {
	cutoff := time.Now().Add(-w.retention)
	deleted, err := w.store.DeleteOlderThan(cutoff)
	if err != nil {
		w.logger.Error("audit retention cleanup failed", "error", err)
	} else if deleted > 0 {
		w.logger.Info("audit retention cleanup completed",
			"deleted", deleted,
			"cutoff", cutoff.Format(time.RFC3339))
	}
}

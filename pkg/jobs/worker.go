package jobs

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// PluginRefresher is the interface that the worker uses to execute refresh operations.
// It is satisfied by plugin.RefreshProvider but avoids a circular dependency.
type PluginRefresher interface {
	Refresh(ctx context.Context, sourceID string) (entitiesLoaded, entitiesRemoved int, duration time.Duration, err error)
	RefreshAll(ctx context.Context) (entitiesLoaded, entitiesRemoved int, duration time.Duration, err error)
}

// PluginLookup resolves a plugin refresher by name.
type PluginLookup func(pluginName string) (PluginRefresher, bool)

// WorkerPool processes queued refresh jobs using a pool of goroutines.
type WorkerPool struct {
	store        *JobStore
	pluginLookup PluginLookup
	cfg          *JobConfig
	logger       *slog.Logger
	wg           sync.WaitGroup
}

// NewWorkerPool creates a new worker pool.
func NewWorkerPool(store *JobStore, pluginLookup PluginLookup, cfg *JobConfig, logger *slog.Logger) *WorkerPool {
	if logger == nil {
		logger = slog.Default()
	}
	return &WorkerPool{
		store:        store,
		pluginLookup: pluginLookup,
		cfg:          cfg,
		logger:       logger,
	}
}

// Run starts the worker pool. It spawns cfg.Concurrency goroutines,
// each polling for jobs. It blocks until the context is cancelled,
// then waits for all workers to finish.
func (wp *WorkerPool) Run(ctx context.Context) {
	if wp.store == nil || !wp.cfg.Enabled {
		wp.logger.Info("job worker pool disabled")
		return
	}

	wp.logger.Info("job worker pool starting",
		"concurrency", wp.cfg.Concurrency,
		"maxRetries", wp.cfg.MaxRetries,
		"pollInterval", wp.cfg.PollInterval.String())

	// Start stuck job cleanup goroutine.
	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		wp.cleanupLoop(ctx)
	}()

	// Start worker goroutines.
	for i := 0; i < wp.cfg.Concurrency; i++ {
		wp.wg.Add(1)
		go func(workerID int) {
			defer wp.wg.Done()
			wp.workerLoop(ctx, workerID)
		}(i)
	}

	<-ctx.Done()
	wp.logger.Info("job worker pool shutting down, waiting for workers to finish")
	wp.wg.Wait()
	wp.logger.Info("job worker pool stopped")
}

// workerLoop is the main loop for a single worker goroutine.
func (wp *WorkerPool) workerLoop(ctx context.Context, workerID int) {
	ticker := time.NewTicker(wp.cfg.PollInterval)
	defer ticker.Stop()

	wp.logger.Info("worker started", "workerID", workerID)

	for {
		select {
		case <-ctx.Done():
			wp.logger.Info("worker stopped", "workerID", workerID)
			return
		case <-ticker.C:
			wp.processOne(ctx, workerID)
		}
	}
}

// processOne tries to claim and process a single job.
func (wp *WorkerPool) processOne(ctx context.Context, workerID int) {
	job, err := wp.store.Claim(wp.cfg.MaxRetries)
	if err != nil {
		wp.logger.Error("failed to claim job", "workerID", workerID, "error", err)
		return
	}
	if job == nil {
		return // No jobs available.
	}

	wp.logger.Info("processing job",
		"workerID", workerID,
		"jobID", job.ID,
		"plugin", job.Plugin,
		"sourceID", job.SourceID,
		"attempt", job.AttemptCount)

	// Look up the plugin's refresher.
	refresher, ok := wp.pluginLookup(job.Plugin)
	if !ok {
		errMsg := "plugin not found or does not support refresh: " + job.Plugin
		wp.logger.Error(errMsg, "jobID", job.ID)
		if err := wp.store.Fail(job.ID, errMsg, wp.cfg.MaxRetries); err != nil {
			wp.logger.Error("failed to mark job as failed", "jobID", job.ID, "error", err)
		}
		return
	}

	// Execute the refresh.
	var entitiesLoaded, entitiesRemoved int
	var duration time.Duration

	if job.SourceID == "" || job.SourceID == "_all" {
		entitiesLoaded, entitiesRemoved, duration, err = refresher.RefreshAll(ctx)
	} else {
		entitiesLoaded, entitiesRemoved, duration, err = refresher.Refresh(ctx, job.SourceID)
	}

	if err != nil {
		wp.logger.Error("job failed",
			"workerID", workerID,
			"jobID", job.ID,
			"error", err)
		if failErr := wp.store.Fail(job.ID, err.Error(), wp.cfg.MaxRetries); failErr != nil {
			wp.logger.Error("failed to mark job as failed", "jobID", job.ID, "error", failErr)
		}
		return
	}

	wp.logger.Info("job completed",
		"workerID", workerID,
		"jobID", job.ID,
		"entitiesLoaded", entitiesLoaded,
		"entitiesRemoved", entitiesRemoved,
		"duration", duration.String())

	if err := wp.store.Complete(job.ID, entitiesLoaded, entitiesRemoved, duration.Milliseconds()); err != nil {
		wp.logger.Error("failed to mark job as complete", "jobID", job.ID, "error", err)
	}
}

// cleanupLoop periodically cleans up stuck jobs and old completed jobs.
func (wp *WorkerPool) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Recover stuck jobs.
			if wp.cfg.ClaimTimeout > 0 {
				recovered, err := wp.store.CleanupStuckJobs(wp.cfg.ClaimTimeout)
				if err != nil {
					wp.logger.Error("failed to cleanup stuck jobs", "error", err)
				} else if recovered > 0 {
					wp.logger.Info("recovered stuck jobs", "count", recovered)
				}
			}

			// Delete old terminal jobs.
			if wp.cfg.RetentionDays > 0 {
				cutoff := time.Now().AddDate(0, 0, -wp.cfg.RetentionDays)
				deleted, err := wp.store.DeleteOlderThan(cutoff)
				if err != nil {
					wp.logger.Error("failed to delete old jobs", "error", err)
				} else if deleted > 0 {
					wp.logger.Info("deleted old jobs", "count", deleted)
				}
			}
		}
	}
}
